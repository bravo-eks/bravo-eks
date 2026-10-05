package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	_ "github.com/lib/pq"
)

var (
	db          *sql.DB
	sqsClient   *sqs.Client
	sqsQueueURL string
)

var validShipmentStatuses = map[string]bool{
	"label_created": true,
	"in_transit":    true,
	"delivered":     true,
}

func main() {
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	sqsQueueURL = strings.TrimSpace(os.Getenv("SQS_QUEUE_URL"))
	if sqsQueueURL == "" {
		log.Fatal("SQS_QUEUE_URL is required")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(15)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(5 * time.Minute)

	waitForDB()

	if strings.EqualFold(os.Getenv("RUN_MIGRATIONS"), "true") {
		migrate()
		if strings.EqualFold(os.Getenv("MIGRATIONS_ONLY"), "true") {
			log.Println("Migration-only mode complete")
			return
		}
	} else {
		log.Println("Database migrations disabled; set RUN_MIGRATIONS=true to run migrations")
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load AWS configuration: %v", err)
	}

	sqsClient = sqs.NewFromConfig(awsConfig, func(options *sqs.Options) {
		if endpoint := strings.TrimSpace(os.Getenv("SQS_ENDPOINT_URL")); endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/shipments", handleShipments)
	mux.HandleFunc("/shipments/", handleShipment)
	mux.HandleFunc("/track/", handleTrack)
	mux.HandleFunc("/webhook", handleCarrierWebhook)

	port := getEnv("PORT", "8085")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Shipping service listening on :%s", port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
}

func migrate() {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS shipments (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL,
			carrier VARCHAR(50) NOT NULL,
			tracking_number VARCHAR(100) UNIQUE,
			status VARCHAR(30) NOT NULL DEFAULT 'label_created',
			recipient_name VARCHAR(255),
			address_line1 TEXT,
			address_line2 TEXT,
			city VARCHAR(100),
			postcode VARCHAR(20),
			country VARCHAR(2) DEFAULT 'GB',
			weight_kg DECIMAL(8,2),
			estimated_delivery DATE,
			shipped_at TIMESTAMP,
			delivered_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shipments_order ON shipments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_shipments_tracking ON shipments(tracking_number)`,
		`CREATE TABLE IF NOT EXISTS tracking_events (
			id SERIAL PRIMARY KEY,
			shipment_id INTEGER NOT NULL REFERENCES shipments(id),
			status VARCHAR(30) NOT NULL,
			location VARCHAR(255),
			description TEXT,
			occurred_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tracking_shipment ON tracking_events(shipment_id)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	}

	log.Println("Shipping service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"

	if err := db.Ping(); err != nil {
		status = "unhealthy"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  status,
			"service": "shipping-service",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  status,
		"service": "shipping-service",
	})
}

func handleShipments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query(
			`SELECT id, order_id, carrier, tracking_number, status, recipient_name,
			        city, country, estimated_delivery, created_at
			 FROM shipments ORDER BY created_at DESC LIMIT 100`,
		)
		if err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type ShipmentSummary struct {
			ID                int    `json:"id"`
			OrderID           int    `json:"order_id"`
			Carrier           string `json:"carrier"`
			TrackingNumber    string `json:"tracking_number"`
			Status            string `json:"status"`
			RecipientName     string `json:"recipient_name"`
			City              string `json:"city"`
			Country           string `json:"country"`
			EstimatedDelivery string `json:"estimated_delivery,omitempty"`
			CreatedAt         string `json:"created_at"`
		}

		shipments := []ShipmentSummary{}

		for rows.Next() {
			var shipment ShipmentSummary
			var estimatedDelivery sql.NullString
			var trackingNumber sql.NullString

			if err := rows.Scan(
				&shipment.ID,
				&shipment.OrderID,
				&shipment.Carrier,
				&trackingNumber,
				&shipment.Status,
				&shipment.RecipientName,
				&shipment.City,
				&shipment.Country,
				&estimatedDelivery,
				&shipment.CreatedAt,
			); err != nil {
				httpError(w, "failed to read shipment", http.StatusInternalServerError)
				return
			}

			if trackingNumber.Valid {
				shipment.TrackingNumber = trackingNumber.String
			}

			if estimatedDelivery.Valid {
				shipment.EstimatedDelivery = estimatedDelivery.String
			}

			shipments = append(shipments, shipment)
		}

		if err := rows.Err(); err != nil {
			httpError(w, "failed to read shipments", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(shipments)

	case http.MethodPost:
		createShipment(w, r)

	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func createShipment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID       int     `json:"order_id"`
		Carrier       string  `json:"carrier"`
		RecipientName string  `json:"recipient_name"`
		AddressLine1  string  `json:"address_line1"`
		AddressLine2  string  `json:"address_line2"`
		City          string  `json:"city"`
		Postcode      string  `json:"postcode"`
		Country       string  `json:"country"`
		WeightKg      float64 `json:"weight_kg"`
	}

	if err := decodeJSON(r, &req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.RecipientName = strings.TrimSpace(req.RecipientName)
	req.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	req.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	req.City = strings.TrimSpace(req.City)
	req.Postcode = strings.TrimSpace(req.Postcode)
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))
	req.Carrier = strings.ToLower(strings.TrimSpace(req.Carrier))

	if req.OrderID <= 0 {
		httpError(w, "order_id must be greater than zero", http.StatusBadRequest)
		return
	}

	if req.RecipientName == "" {
		httpError(w, "recipient_name is required", http.StatusBadRequest)
		return
	}

	if req.AddressLine1 == "" {
		httpError(w, "address_line1 is required", http.StatusBadRequest)
		return
	}

	if req.City == "" {
		httpError(w, "city is required", http.StatusBadRequest)
		return
	}

	if req.Postcode == "" {
		httpError(w, "postcode is required", http.StatusBadRequest)
		return
	}

	if req.Country == "" {
		req.Country = "GB"
	}

	if len(req.Country) != 2 {
		httpError(w, "country must be a 2-letter country code", http.StatusBadRequest)
		return
	}

	if req.WeightKg <= 0 {
		httpError(w, "weight_kg must be greater than zero", http.StatusBadRequest)
		return
	}

	if req.WeightKg > 1000 {
		httpError(w, "weight_kg is too large", http.StatusBadRequest)
		return
	}

	if req.Carrier == "" {
		req.Carrier = "royal_mail"
	}

	trackingNumber := generateTrackingNumber(req.Carrier)
	estimatedDelivery := time.Now().AddDate(0, 0, 3+rand.Intn(5))

	tx, err := db.Begin()
	if err != nil {
		httpError(w, "failed to start database transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var shipmentID int

	err = tx.QueryRow(
		`INSERT INTO shipments (
			order_id,
			carrier,
			tracking_number,
			recipient_name,
			address_line1,
			address_line2,
			city,
			postcode,
			country,
			weight_kg,
			estimated_delivery
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		req.OrderID,
		req.Carrier,
		trackingNumber,
		req.RecipientName,
		req.AddressLine1,
		req.AddressLine2,
		req.City,
		req.Postcode,
		req.Country,
		req.WeightKg,
		estimatedDelivery,
	).Scan(&shipmentID)

	if err != nil {
		httpError(w, "failed to create shipment", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO tracking_events (
			shipment_id,
			status,
			location,
			description,
			occurred_at
		)
		VALUES ($1, 'label_created', $2, 'Shipping label created', NOW())`,
		shipmentID,
		req.City,
	)

	if err != nil {
		httpError(w, "failed to create tracking event", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		httpError(w, "failed to commit shipment", http.StatusInternalServerError)
		return
	}

	err = publishEvent(r.Context(), "shipment.created", map[string]interface{}{
		"shipment_id":     shipmentID,
		"order_id":        req.OrderID,
		"tracking_number": trackingNumber,
		"carrier":         req.Carrier,
	})

	if err != nil {
		log.Printf(
			"failed to publish shipment.created event for shipment %d: %v",
			shipmentID,
			err,
		)

		httpError(w, "shipment created but event publishing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"shipment_id":        shipmentID,
		"tracking_number":    trackingNumber,
		"carrier":            req.Carrier,
		"estimated_delivery": estimatedDelivery.Format("2006-01-02"),
	})
}

func handleShipment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/shipments/"))

	if id == "" {
		httpError(w, "shipment id required", http.StatusBadRequest)
		return
	}

	var shipment struct {
		ID                int     `json:"id"`
		OrderID           int     `json:"order_id"`
		Carrier           string  `json:"carrier"`
		TrackingNumber    string  `json:"tracking_number"`
		Status            string  `json:"status"`
		RecipientName     string  `json:"recipient_name"`
		City              string  `json:"city"`
		Country           string  `json:"country"`
		WeightKg          float64 `json:"weight_kg"`
		EstimatedDelivery string  `json:"estimated_delivery"`
		CreatedAt         string  `json:"created_at"`
	}

	err := db.QueryRow(
		`SELECT id, order_id, carrier, tracking_number, status, recipient_name,
		        city, country, weight_kg, estimated_delivery, created_at
		 FROM shipments
		 WHERE id = $1`,
		id,
	).Scan(
		&shipment.ID,
		&shipment.OrderID,
		&shipment.Carrier,
		&shipment.TrackingNumber,
		&shipment.Status,
		&shipment.RecipientName,
		&shipment.City,
		&shipment.Country,
		&shipment.WeightKg,
		&shipment.EstimatedDelivery,
		&shipment.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			httpError(w, "shipment not found", http.StatusNotFound)
			return
		}

		httpError(w, "failed to query shipment", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(shipment)
}

func handleTrack(w http.ResponseWriter, r *http.Request) {
	tracking := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/track/"))

	if tracking == "" {
		httpError(w, "tracking number required", http.StatusBadRequest)
		return
	}

	var shipmentID int
	var status string

	err := db.QueryRow(
		"SELECT id, status FROM shipments WHERE tracking_number = $1",
		tracking,
	).Scan(&shipmentID, &status)

	if err != nil {
		if err == sql.ErrNoRows {
			httpError(w, "tracking number not found", http.StatusNotFound)
			return
		}

		httpError(w, "failed to query tracking number", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(
		`SELECT status, location, description, occurred_at
		 FROM tracking_events
		 WHERE shipment_id = $1
		 ORDER BY occurred_at DESC`,
		shipmentID,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Event struct {
		Status      string `json:"status"`
		Location    string `json:"location"`
		Description string `json:"description"`
		OccurredAt  string `json:"occurred_at"`
	}

	events := []Event{}

	for rows.Next() {
		var event Event

		if err := rows.Scan(
			&event.Status,
			&event.Location,
			&event.Description,
			&event.OccurredAt,
		); err != nil {
			httpError(w, "failed to read tracking event", http.StatusInternalServerError)
			return
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		httpError(w, "failed to read tracking events", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tracking_number": tracking,
		"current_status":  status,
		"events":          events,
	})
}

func handleCarrierWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TrackingNumber string `json:"tracking_number"`
		Status         string `json:"status"`
		Location       string `json:"location"`
		Description    string `json:"description"`
	}

	if err := decodeJSON(r, &req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.TrackingNumber = strings.TrimSpace(req.TrackingNumber)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	req.Location = strings.TrimSpace(req.Location)
	req.Description = strings.TrimSpace(req.Description)

	if req.TrackingNumber == "" {
		httpError(w, "tracking_number is required", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		httpError(w, "status is required", http.StatusBadRequest)
		return
	}

	if !validShipmentStatuses[req.Status] {
		httpError(w, "unsupported shipment status", http.StatusBadRequest)
		return
	}

	var shipmentID int
	var orderID int

	tx, err := db.Begin()
	if err != nil {
		httpError(w, "failed to start database transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	err = tx.QueryRow(
		`SELECT id, order_id
		 FROM shipments
		 WHERE tracking_number = $1
		 FOR UPDATE`,
		req.TrackingNumber,
	).Scan(&shipmentID, &orderID)

	if err != nil {
		if err == sql.ErrNoRows {
			httpError(w, "tracking number not found", http.StatusNotFound)
			return
		}

		httpError(w, "failed to query shipment", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		`UPDATE shipments
		 SET status = $1, updated_at = NOW()
		 WHERE id = $2`,
		req.Status,
		shipmentID,
	)

	if err != nil {
		httpError(w, "failed to update shipment status", http.StatusInternalServerError)
		return
	}

	switch req.Status {
	case "delivered":
		_, err = tx.Exec(
			`UPDATE shipments
			 SET delivered_at = NOW()
			 WHERE id = $1`,
			shipmentID,
		)

	case "in_transit":
		_, err = tx.Exec(
			`UPDATE shipments
			 SET shipped_at = COALESCE(shipped_at, NOW())
			 WHERE id = $1`,
			shipmentID,
		)
	}

	if err != nil {
		httpError(w, "failed to update shipment timestamps", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(
		`INSERT INTO tracking_events (
			shipment_id,
			status,
			location,
			description,
			occurred_at
		)
		VALUES ($1, $2, $3, $4, NOW())`,
		shipmentID,
		req.Status,
		req.Location,
		req.Description,
	)

	if err != nil {
		httpError(w, "failed to create tracking event", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		httpError(w, "failed to commit shipment update", http.StatusInternalServerError)
		return
	}

	err = publishEvent(r.Context(), "shipment."+req.Status, map[string]interface{}{
		"shipment_id":     shipmentID,
		"order_id":        orderID,
		"tracking_number": req.TrackingNumber,
		"status":          req.Status,
		"location":        req.Location,
	})

	if err != nil {
		log.Printf(
			"failed to publish shipment.%s event for shipment %d: %v",
			req.Status,
			shipmentID,
			err,
		)

		httpError(w, "shipment updated but event publishing failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "accepted",
	})
}

func generateTrackingNumber(carrier string) string {
	prefix := "RM"

	switch carrier {
	case "dpd":
		prefix = "DPD"
	case "hermes":
		prefix = "EVR"
	case "ups":
		prefix = "1Z"
	}

	return fmt.Sprintf(
		"%s%d%d",
		prefix,
		time.Now().UnixNano(),
		rand.Intn(100000),
	)
}

func publishEvent(
	ctx context.Context,
	eventType string,
	payload map[string]interface{},
) error {
	if sqsClient == nil {
		return fmt.Errorf("SQS client is not initialized")
	}

	if sqsQueueURL == "" {
		return fmt.Errorf("SQS_QUEUE_URL is not configured")
	}

	event := map[string]interface{}{
		"type":      eventType,
		"payload":   payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(sqsQueueURL),
		MessageBody: aws.String(string(data)),
	})
	if err != nil {
		return fmt.Errorf("failed to send SQS message: %w", err)
	}

	log.Printf("Event published to SQS: type=%s", eventType)

	return nil
}

func decodeJSON(r *http.Request, destination interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON object")
	}

	return nil
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func waitForDB() {
	for i := 0; i < 120; i++ {
		if err := db.Ping(); err == nil {
			return
		}

		log.Printf("Waiting for database... (%d/120)", i+1)
		time.Sleep(time.Second)
	}

	log.Fatal("Database not ready after 120s")
}
