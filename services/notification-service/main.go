package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(5 * time.Minute)

	waitForDB()

	if getEnv("RUN_MIGRATIONS", "false") == "true" {
		migrate()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/send", handleSend)
	mux.HandleFunc("/history", handleHistory)
	mux.HandleFunc("/templates", handleTemplates)

	port := getEnv("PORT", "8084")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Notification service listening on :%s", port)
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
		`CREATE TABLE IF NOT EXISTS notifications (
			id SERIAL PRIMARY KEY,
			recipient VARCHAR(255) NOT NULL,
			channel VARCHAR(20) NOT NULL,
			template VARCHAR(100) NOT NULL,
			subject TEXT,
			body TEXT NOT NULL,
			metadata JSONB,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			sent_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_recipient ON notifications(recipient)`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status)`,
		`CREATE TABLE IF NOT EXISTS notification_templates (
			id VARCHAR(100) PRIMARY KEY,
			channel VARCHAR(20) NOT NULL,
			subject TEXT,
			body TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	templates := []struct {
		id      string
		channel string
		subject string
		body    string
	}{
		{
			"order_confirmed",
			"email",
			"Order Confirmed - #{{.OrderID}}",
			"Hi {{.CustomerName}}, your order #{{.OrderID}} has been confirmed. Total: {{.Currency}} {{.Total}}",
		},
		{
			"order_shipped",
			"email",
			"Your Order Has Shipped - #{{.OrderID}}",
			"Good news! Your order #{{.OrderID}} has been shipped. Tracking: {{.TrackingNumber}}",
		},
		{
			"order_delivered",
			"email",
			"Order Delivered - #{{.OrderID}}",
			"Your order #{{.OrderID}} has been delivered. Thank you for your purchase!",
		},
		{
			"payment_failed",
			"email",
			"Payment Failed - Order #{{.OrderID}}",
			"We were unable to process your payment for order #{{.OrderID}}. Please update your payment method.",
		},
		{
			"low_stock_alert",
			"email",
			"Low Stock Alert - {{.ProductName}}",
			"Product {{.ProductName}} ({{.SKU}}) is running low. Available: {{.Available}}",
		},
		{
			"order_confirmed_sms",
			"sms",
			"",
			"Order #{{.OrderID}} confirmed. Total: {{.Currency}} {{.Total}}",
		},
		{
			"order_shipped_sms",
			"sms",
			"",
			"Order #{{.OrderID}} shipped. Track: {{.TrackingNumber}}",
		},
	}

	for _, t := range templates {
		if _, err := db.Exec(
			`INSERT INTO notification_templates (id, channel, subject, body)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (id) DO NOTHING`,
			t.id,
			t.channel,
			t.subject,
			t.body,
		); err != nil {
			log.Fatalf("Failed to seed notification template %s: %v", t.id, err)
		}
	}

	log.Println("Notification service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := "ok"
	statusCode := http.StatusOK

	if err := db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  status,
		"service": "notification-service",
	})
}

func handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Recipient string                 `json:"recipient"`
		Channel   string                 `json:"channel"`
		Template  string                 `json:"template"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Recipient = strings.TrimSpace(req.Recipient)
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.Template = strings.TrimSpace(req.Template)

	if req.Recipient == "" {
		httpError(w, "recipient is required", http.StatusBadRequest)
		return
	}

	if req.Template == "" {
		httpError(w, "template is required", http.StatusBadRequest)
		return
	}

	if req.Channel == "" {
		req.Channel = "email"
	}

	if req.Channel != "email" && req.Channel != "sms" {
		httpError(w, "channel must be email or sms", http.StatusBadRequest)
		return
	}

	var templateChannel string
	var subject string
	var body string

	err := db.QueryRowContext(
		r.Context(),
		`SELECT channel, subject, body
		 FROM notification_templates
		 WHERE id = $1`,
		req.Template,
	).Scan(
		&templateChannel,
		&subject,
		&body,
	)

	if errors.Is(err, sql.ErrNoRows) {
		httpError(w, "notification template not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("template lookup failed for %s: %v", req.Template, err)
		httpError(w, "failed to retrieve notification template", http.StatusInternalServerError)
		return
	}

	if templateChannel != req.Channel {
		httpError(w, "template does not support requested channel", http.StatusBadRequest)
		return
	}

	metadata, err := json.Marshal(req.Data)
	if err != nil {
		log.Printf("failed to encode notification metadata: %v", err)
		httpError(w, "invalid notification data", http.StatusBadRequest)
		return
	}

	// Email and SMS delivery are intentionally simulated for this project.
	// A future improvement could integrate AWS SES for email and SNS for SMS.
	status := "sent"
	sentAt := time.Now()

	var notifID int

	err = db.QueryRowContext(
		r.Context(),
		`INSERT INTO notifications (
			recipient,
			channel,
			template,
			subject,
			body,
			metadata,
			status,
			sent_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		req.Recipient,
		req.Channel,
		req.Template,
		subject,
		body,
		metadata,
		status,
		sentAt,
	).Scan(&notifID)

	if err != nil {
		log.Printf("failed to persist notification: %v", err)
		httpError(w, "failed to save notification", http.StatusInternalServerError)
		return
	}

	log.Printf(
		"Notification sent (simulated): [%s] %s -> %s (template: %s)",
		req.Channel,
		req.Template,
		req.Recipient,
		req.Template,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        notifID,
		"recipient": req.Recipient,
		"channel":   req.Channel,
		"status":    status,
	})
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recipient := strings.TrimSpace(r.URL.Query().Get("recipient"))

	query := `
		SELECT id, recipient, channel, template, subject, status, sent_at, created_at
		FROM notifications
	`

	args := []interface{}{}

	if recipient != "" {
		query += " WHERE recipient = $1"
		args = append(args, recipient)
	}

	query += " ORDER BY created_at DESC LIMIT 50"

	rows, err := db.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("notification history query failed: %v", err)
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Notif struct {
		ID        int    `json:"id"`
		Recipient string `json:"recipient"`
		Channel   string `json:"channel"`
		Template  string `json:"template"`
		Subject   string `json:"subject"`
		Status    string `json:"status"`
		SentAt    string `json:"sent_at"`
		CreatedAt string `json:"created_at"`
	}

	notifs := []Notif{}

	for rows.Next() {
		var n Notif
		var sentAt sql.NullTime

		if err := rows.Scan(
			&n.ID,
			&n.Recipient,
			&n.Channel,
			&n.Template,
			&n.Subject,
			&n.Status,
			&sentAt,
			&n.CreatedAt,
		); err != nil {
			log.Printf("notification history scan failed: %v", err)
			httpError(w, "failed to read notification history", http.StatusInternalServerError)
			return
		}

		if sentAt.Valid {
			n.SentAt = sentAt.Time.Format(time.RFC3339)
		}

		notifs = append(notifs, n)
	}

	if err := rows.Err(); err != nil {
		log.Printf("notification history iteration failed: %v", err)
		httpError(w, "failed to read notification history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(notifs)
}

func handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.QueryContext(
		r.Context(),
		"SELECT id, channel, subject, body FROM notification_templates ORDER BY id",
	)
	if err != nil {
		log.Printf("notification template query failed: %v", err)
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Template struct {
		ID      string `json:"id"`
		Channel string `json:"channel"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
	}

	templates := []Template{}

	for rows.Next() {
		var t Template

		if err := rows.Scan(
			&t.ID,
			&t.Channel,
			&t.Subject,
			&t.Body,
		); err != nil {
			log.Printf("notification template scan failed: %v", err)
			httpError(w, "failed to read notification templates", http.StatusInternalServerError)
			return
		}

		templates = append(templates, t)
	}

	if err := rows.Err(); err != nil {
		log.Printf("notification template iteration failed: %v", err)
		httpError(w, "failed to read notification templates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(templates)
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
