package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	_ "github.com/lib/pq"
)

var db *sql.DB
var sqsClient *sqs.Client
var sqsQueueURL string

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	sqsQueueURL = os.Getenv("SQS_QUEUE_URL")
	if sqsQueueURL == "" {
		log.Fatal("SQS_QUEUE_URL is required")
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Failed to load AWS configuration: %v", err)
	}
	sqsClient = sqs.NewFromConfig(awsConfig)

	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	waitForDB()

	if getEnv("RUN_MIGRATIONS", "false") == "true" {
		migrate()
		if getEnv("MIGRATIONS_ONLY", "false") == "true" {
			log.Println("Migration-only mode complete")
			return
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/charge", handleCharge)
	mux.HandleFunc("/payments", handlePayments)
	mux.HandleFunc("/refund", handleRefund)
	mux.HandleFunc("/refund-requests", handleRefundRequests)
	mux.HandleFunc("/refund-requests/", handleRefundRequestAction)
	mux.HandleFunc("/ledger", handleLedger)
	mux.HandleFunc("/balance/", handleBalance)

	port := getEnv("PORT", "8083")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Payment service listening on :%s", port)
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
		`CREATE TABLE IF NOT EXISTS payments (
			id VARCHAR(50) PRIMARY KEY,
			order_id INTEGER NOT NULL,
			customer_id VARCHAR(255) NOT NULL,
			amount DECIMAL(12,2) NOT NULL,
			currency VARCHAR(3) NOT NULL DEFAULT 'GBP',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			method VARCHAR(50),
			reference VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_order ON payments(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_customer ON payments(customer_id)`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id SERIAL PRIMARY KEY,
			payment_id VARCHAR(50) NOT NULL REFERENCES payments(id),
			entry_type VARCHAR(20) NOT NULL,
			debit DECIMAL(12,2) NOT NULL DEFAULT 0,
			credit DECIMAL(12,2) NOT NULL DEFAULT 0,
			currency VARCHAR(3) NOT NULL DEFAULT 'GBP',
			description TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_payment ON ledger_entries(payment_id)`,
		`CREATE TABLE IF NOT EXISTS refund_requests (
			id SERIAL PRIMARY KEY,
			payment_id VARCHAR(50) NOT NULL,
			order_id INTEGER NOT NULL,
			customer_id VARCHAR(255) NOT NULL,
			amount DECIMAL(12,2) NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			status VARCHAR(20) NOT NULL DEFAULT 'requested',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_requests_payment ON refund_requests(payment_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_requests_customer ON refund_requests(customer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refund_requests_status ON refund_requests(status)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}
	log.Println("Payment service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	if err := db.Ping(); err != nil {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": status, "service": "payment-service"})
}

func handleCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID    int     `json:"order_id"`
		CustomerID string  `json:"customer_id"`
		Amount     float64 `json:"amount"`
		Currency   string  `json:"currency"`
		Method     string  `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrderID <= 0 {
		httpError(w, "order_id is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.CustomerID) == "" {
		httpError(w, "customer_id is required", http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		httpError(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	paymentID := generatePaymentID()
	currency := req.Currency
	if currency == "" {
		currency = "GBP"
	}

	// Simulate payment processing (90% success rate)
	status := "completed"
	if rand.Intn(10) == 0 {
		status = "failed"
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		r.Context(),
		`INSERT INTO payments (id, order_id, customer_id, amount, currency, status, method)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		paymentID, req.OrderID, req.CustomerID, req.Amount, currency, status, req.Method,
	)
	if err != nil {
		httpError(w, "failed to process payment", http.StatusInternalServerError)
		return
	}

	if status == "completed" {
		// Double-entry: debit customer, credit revenue
		if _, err := tx.ExecContext(
			r.Context(),
			`INSERT INTO ledger_entries (payment_id, entry_type, debit, currency, description)
			 VALUES ($1, 'charge', $2, $3, $4)`,
			paymentID, req.Amount, currency, fmt.Sprintf("Order #%d payment", req.OrderID),
		); err != nil {
			log.Printf("Failed to create charge ledger entry: %v", err)
			httpError(w, "failed to record payment ledger", http.StatusInternalServerError)
			return
		}

		if _, err := tx.ExecContext(
			r.Context(),
			`INSERT INTO ledger_entries (payment_id, entry_type, credit, currency, description)
			 VALUES ($1, 'revenue', $2, $3, $4)`,
			paymentID, req.Amount, currency, fmt.Sprintf("Order #%d revenue", req.OrderID),
		); err != nil {
			log.Printf("Failed to create revenue ledger entry: %v", err)
			httpError(w, "failed to record payment ledger", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit payment transaction: %v", err)
		httpError(w, "failed to process payment", http.StatusInternalServerError)
		return
	}

	// Publish event
	if err := publishEvent(r.Context(), "payment."+status, map[string]interface{}{
		"payment_id":  paymentID,
		"order_id":    req.OrderID,
		"customer_id": req.CustomerID,
		"amount":      req.Amount,
		"currency":    currency,
		"status":      status,
	}); err != nil {
		log.Printf("Failed to publish payment.%s event for payment %s: %v", status, paymentID, err)
		httpError(w, "payment processed but event publishing failed", http.StatusInternalServerError)
		return
	}

	code := http.StatusCreated
	if status == "failed" {
		code = http.StatusPaymentRequired
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payment_id": paymentID,
		"order_id":   req.OrderID,
		"amount":     req.Amount,
		"currency":   currency,
		"status":     status,
	})
}

func handlePayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.QueryContext(
		r.Context(),
		`SELECT id, order_id, customer_id, amount, currency, status
		 FROM payments
		 WHERE method IS NULL OR method <> 'refund'
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		log.Printf("Failed to query payments: %v", err)
		httpError(w, "failed to retrieve payments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Payment struct {
		ID         string  `json:"id"`
		OrderID    int     `json:"order_id"`
		CustomerID string  `json:"customer_id"`
		Amount     float64 `json:"amount"`
		Currency   string  `json:"currency"`
		Status     string  `json:"status"`
	}

	payments := []Payment{}
	for rows.Next() {
		var payment Payment
		if err := rows.Scan(
			&payment.ID,
			&payment.OrderID,
			&payment.CustomerID,
			&payment.Amount,
			&payment.Currency,
			&payment.Status,
		); err != nil {
			log.Printf("Failed to scan payment: %v", err)
			httpError(w, "failed to retrieve payments", http.StatusInternalServerError)
			return
		}
		payments = append(payments, payment)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Failed while reading payments: %v", err)
		httpError(w, "failed to retrieve payments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payments)
}

type refundRequest struct {
	ID         int       `json:"id"`
	PaymentID  string    `json:"payment_id"`
	OrderID    int       `json:"order_id"`
	CustomerID string    `json:"customer_id"`
	Amount     float64   `json:"amount"`
	Reason     string    `json:"reason"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type refundInput struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
}

type refundExecutionResult struct {
	RefundID  string  `json:"refund_id"`
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
}

func handleRefund(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req refundInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.PaymentID) == "" {
		httpError(w, "payment_id is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		httpError(w, "refund amount must be positive", http.StatusBadRequest)
		return
	}

	result, _, _, code, message := executeRefund(r.Context(), req, 0)
	if message != "" {
		httpError(w, message, code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleRefundRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		createRefundRequest(w, r)
	case http.MethodGet:
		listRefundRequests(w, r)
	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func createRefundRequest(w http.ResponseWriter, r *http.Request) {
	var req refundInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.PaymentID) == "" {
		httpError(w, "payment_id is required", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		httpError(w, "refund amount must be positive", http.StatusBadRequest)
		return
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var originalAmount float64
	var paymentStatus, customerID string
	var orderID int
	if err := tx.QueryRowContext(
		r.Context(),
		`SELECT amount, order_id, status, customer_id
		 FROM payments
		 WHERE id = $1
		 FOR UPDATE`,
		req.PaymentID,
	).Scan(&originalAmount, &orderID, &paymentStatus, &customerID); err != nil {
		if err == sql.ErrNoRows {
			httpError(w, "payment not found", http.StatusNotFound)
		} else {
			log.Printf("Failed to retrieve payment for refund request: %v", err)
			httpError(w, "failed to retrieve payment", http.StatusInternalServerError)
		}
		return
	}
	if paymentStatus != "completed" && paymentStatus != "partially_refunded" {
		httpError(w, "can only refund completed or partially refunded payments", http.StatusConflict)
		return
	}

	var alreadyRefunded, activeRequested float64
	if err := tx.QueryRowContext(
		r.Context(),
		`SELECT COALESCE(SUM(amount), 0)
		 FROM payments
		 WHERE reference = $1 AND method = 'refund'`,
		req.PaymentID,
	).Scan(&alreadyRefunded); err != nil {
		log.Printf("Failed to calculate existing refunds: %v", err)
		httpError(w, "failed to process refund request", http.StatusInternalServerError)
		return
	}
	if err := tx.QueryRowContext(
		r.Context(),
		`SELECT COALESCE(SUM(amount), 0)
		 FROM refund_requests
		 WHERE payment_id = $1 AND status IN ('requested', 'approved')`,
		req.PaymentID,
	).Scan(&activeRequested); err != nil {
		log.Printf("Failed to calculate outstanding refund requests: %v", err)
		httpError(w, "failed to process refund request", http.StatusInternalServerError)
		return
	}

	if req.Amount > originalAmount-alreadyRefunded-activeRequested {
		httpError(w, "refund amount exceeds remaining refundable balance after outstanding requests", http.StatusBadRequest)
		return
	}

	var created refundRequest
	err = tx.QueryRowContext(
		r.Context(),
		`INSERT INTO refund_requests (payment_id, order_id, customer_id, amount, reason, status)
		 VALUES ($1, $2, $3, $4, $5, 'requested')
		 RETURNING id, payment_id, order_id, customer_id, amount, reason, status, created_at, updated_at`,
		req.PaymentID, orderID, customerID, req.Amount, req.Reason,
	).Scan(
		&created.ID, &created.PaymentID, &created.OrderID, &created.CustomerID,
		&created.Amount, &created.Reason, &created.Status, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		log.Printf("Failed to create refund request: %v", err)
		httpError(w, "failed to create refund request", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit refund request: %v", err)
		httpError(w, "failed to create refund request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func listRefundRequests(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		r.Context(),
		`SELECT id, payment_id, order_id, customer_id, amount, reason, status, created_at, updated_at
		 FROM refund_requests
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		log.Printf("Failed to query refund requests: %v", err)
		httpError(w, "failed to retrieve refund requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	requests := []refundRequest{}
	for rows.Next() {
		var req refundRequest
		if err := rows.Scan(
			&req.ID, &req.PaymentID, &req.OrderID, &req.CustomerID, &req.Amount,
			&req.Reason, &req.Status, &req.CreatedAt, &req.UpdatedAt,
		); err != nil {
			log.Printf("Failed to scan refund request: %v", err)
			httpError(w, "failed to retrieve refund requests", http.StatusInternalServerError)
			return
		}
		requests = append(requests, req)
	}
	if err := rows.Err(); err != nil {
		log.Printf("Failed while reading refund requests: %v", err)
		httpError(w, "failed to retrieve refund requests", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

func handleRefundRequestAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/refund-requests/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		httpError(w, "invalid refund request path", http.StatusNotFound)
		return
	}
	requestID, err := strconv.Atoi(parts[0])
	if err != nil || requestID <= 0 {
		httpError(w, "invalid refund request id", http.StatusBadRequest)
		return
	}

	switch parts[1] {
	case "reject":
		rejectRefundRequest(w, r, requestID)
	case "approve":
		approveRefundRequest(w, r, requestID)
	default:
		httpError(w, "refund request action not found", http.StatusNotFound)
	}
}

func rejectRefundRequest(w http.ResponseWriter, r *http.Request, requestID int) {
	req, err := updateRefundRequestStatus(r, requestID, "rejected")
	if err != nil {
		writeRefundRequestActionError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(req)
}

func approveRefundRequest(w http.ResponseWriter, r *http.Request, requestID int) {
	req, err := updateRefundRequestStatus(r, requestID, "approved")
	if err != nil {
		writeRefundRequestActionError(w, err)
		return
	}

	result, committed, retrySafe, code, message := executeRefund(
		r.Context(),
		refundInput{PaymentID: req.PaymentID, Amount: req.Amount, Reason: req.Reason},
		req.ID,
	)
	if !committed && retrySafe {
		if _, err := db.ExecContext(
			r.Context(),
			`UPDATE refund_requests
			 SET status = 'requested', updated_at = NOW()
			 WHERE id = $1 AND status = 'approved'`,
			req.ID,
		); err != nil {
			log.Printf("Failed to release refund request %d after unsuccessful approval: %v", req.ID, err)
		}
	}
	if message != "" {
		if committed {
			updated, getErr := getRefundRequest(r.Context(), req.ID)
			if getErr == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(code)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   message,
					"request": updated,
					"refund":  result,
				})
				return
			}
			log.Printf("Refund request %d was refunded but could not be reloaded: %v", req.ID, getErr)
		}
		httpError(w, message, code)
		return
	}

	updated, loadErr := getRefundRequest(r.Context(), req.ID)
	if loadErr != nil {
		log.Printf("Refund request %d was refunded but could not be reloaded: %v", req.ID, loadErr)
		httpError(w, "refund completed but failed to retrieve refund request", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request": updated,
		"refund":  result,
	})
}

func updateRefundRequestStatus(r *http.Request, requestID int, newStatus string) (refundRequest, *refundRequestActionError) {
	var req refundRequest
	err := db.QueryRowContext(
		r.Context(),
		`UPDATE refund_requests
		 SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND status = 'requested'
		 RETURNING id, payment_id, order_id, customer_id, amount, reason, status, created_at, updated_at`,
		newStatus, requestID,
	).Scan(
		&req.ID, &req.PaymentID, &req.OrderID, &req.CustomerID, &req.Amount,
		&req.Reason, &req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == nil {
		return req, nil
	}
	if err != sql.ErrNoRows {
		log.Printf("Failed to update refund request %d: %v", requestID, err)
		return refundRequest{}, &refundRequestActionError{code: http.StatusInternalServerError, message: "failed to update refund request"}
	}

	var status string
	err = db.QueryRowContext(r.Context(), "SELECT status FROM refund_requests WHERE id = $1", requestID).Scan(&status)
	if err == sql.ErrNoRows {
		return refundRequest{}, &refundRequestActionError{code: http.StatusNotFound, message: "refund request not found"}
	}
	if err != nil {
		log.Printf("Failed to retrieve refund request %d status: %v", requestID, err)
		return refundRequest{}, &refundRequestActionError{code: http.StatusInternalServerError, message: "failed to retrieve refund request"}
	}
	return refundRequest{}, &refundRequestActionError{code: http.StatusConflict, message: "only requested refunds can be approved or rejected"}
}

type refundRequestActionError struct {
	code    int
	message string
}

func writeRefundRequestActionError(w http.ResponseWriter, err *refundRequestActionError) {
	httpError(w, err.message, err.code)
}

func getRefundRequest(ctx context.Context, requestID int) (refundRequest, error) {
	var req refundRequest
	err := db.QueryRowContext(
		ctx,
		`SELECT id, payment_id, order_id, customer_id, amount, reason, status, created_at, updated_at
		 FROM refund_requests WHERE id = $1`,
		requestID,
	).Scan(
		&req.ID, &req.PaymentID, &req.OrderID, &req.CustomerID, &req.Amount,
		&req.Reason, &req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	return req, err
}

func executeRefund(ctx context.Context, req refundInput, refundRequestID int) (refundExecutionResult, bool, bool, int, string) {
	result := refundExecutionResult{PaymentID: req.PaymentID, Amount: req.Amount, Status: "completed"}
	if strings.TrimSpace(req.PaymentID) == "" {
		return result, false, true, http.StatusBadRequest, "payment_id is required"
	}
	if req.Amount <= 0 {
		return result, false, true, http.StatusBadRequest, "refund amount must be positive"
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, false, true, http.StatusInternalServerError, "transaction failed"
	}
	defer tx.Rollback()

	var originalAmount float64
	var orderID int
	var paymentStatus, customerID, currency string
	err = tx.QueryRowContext(
		ctx,
		`SELECT amount, order_id, status, customer_id, currency
		 FROM payments
		 WHERE id = $1
		 FOR UPDATE`,
		req.PaymentID,
	).Scan(&originalAmount, &orderID, &paymentStatus, &customerID, &currency)
	if err == sql.ErrNoRows {
		return result, false, true, http.StatusNotFound, "payment not found"
	}
	if err != nil {
		log.Printf("Failed to retrieve payment for refund: %v", err)
		return result, false, true, http.StatusInternalServerError, "failed to retrieve payment"
	}
	if paymentStatus != "completed" && paymentStatus != "partially_refunded" {
		return result, false, true, http.StatusConflict, "can only refund completed or partially refunded payments"
	}

	var alreadyRefunded, activeRequested float64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(amount), 0)
		 FROM payments
		 WHERE reference = $1 AND method = 'refund'`,
		req.PaymentID,
	).Scan(&alreadyRefunded); err != nil {
		log.Printf("Failed to calculate existing refunds: %v", err)
		return result, false, true, http.StatusInternalServerError, "failed to process refund"
	}
	activeQuery := `SELECT COALESCE(SUM(amount), 0)
		FROM refund_requests
		WHERE payment_id = $1 AND status IN ('requested', 'approved')`
	activeArgs := []interface{}{req.PaymentID}
	if refundRequestID > 0 {
		activeQuery += " AND id <> $2"
		activeArgs = append(activeArgs, refundRequestID)
	}
	if err := tx.QueryRowContext(ctx, activeQuery, activeArgs...).Scan(&activeRequested); err != nil {
		log.Printf("Failed to calculate outstanding refund requests: %v", err)
		return result, false, true, http.StatusInternalServerError, "failed to process refund"
	}

	available := originalAmount - alreadyRefunded - activeRequested
	if available <= 0 {
		return result, false, true, http.StatusConflict, "payment has no remaining refundable balance"
	}
	if req.Amount > available {
		return result, false, true, http.StatusBadRequest, "refund amount exceeds remaining refundable balance"
	}

	refundID := generatePaymentID()
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO payments (id, order_id, customer_id, amount, currency, status, method, reference)
		 VALUES ($1, $2, $3, $4, $5, 'completed', 'refund', $6)`,
		refundID, orderID, customerID, req.Amount, currency, req.PaymentID,
	); err != nil {
		log.Printf("Failed to insert refund payment: %v", err)
		return result, false, true, http.StatusInternalServerError, "refund insert failed"
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO ledger_entries (payment_id, entry_type, credit, currency, description)
		 VALUES ($1, 'refund', $2, $3, $4)`,
		refundID, req.Amount, currency, fmt.Sprintf("Refund for payment %s: %s", req.PaymentID, req.Reason),
	); err != nil {
		log.Printf("Failed to create refund ledger entry: %v", err)
		return result, false, true, http.StatusInternalServerError, "failed to record refund ledger"
	}

	newStatus := "partially_refunded"
	if alreadyRefunded+req.Amount >= originalAmount {
		newStatus = "refunded"
	}
	if _, err := tx.ExecContext(
		ctx,
		"UPDATE payments SET status = $1, updated_at = NOW() WHERE id = $2",
		newStatus, req.PaymentID,
	); err != nil {
		log.Printf("Failed to update original payment status: %v", err)
		return result, false, true, http.StatusInternalServerError, "failed to update payment status"
	}
	if refundRequestID > 0 {
		updateResult, err := tx.ExecContext(
			ctx,
			`UPDATE refund_requests
			 SET status = 'refunded', updated_at = NOW()
			 WHERE id = $1 AND status = 'approved'`,
			refundRequestID,
		)
		if err != nil {
			log.Printf("Failed to mark refund request %d as refunded: %v", refundRequestID, err)
			return result, false, true, http.StatusInternalServerError, "failed to update refund request"
		}
		if count, err := updateResult.RowsAffected(); err != nil || count != 1 {
			log.Printf("Refund request %d was not in approved state during refund execution", refundRequestID)
			return result, false, true, http.StatusConflict, "refund request is no longer approved"
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit refund transaction: %v", err)
		return result, false, false, http.StatusInternalServerError, "refund outcome is uncertain; request remains approved to prevent duplicate execution"
	}

	result.RefundID = refundID
	if err := publishEvent(ctx, "payment.refunded", map[string]interface{}{
		"refund_id":  refundID,
		"payment_id": req.PaymentID,
		"order_id":   orderID,
		"amount":     req.Amount,
		"reason":     req.Reason,
	}); err != nil {
		log.Printf("Failed to publish payment.refunded event for payment %s: %v", req.PaymentID, err)
		return result, true, false, http.StatusInternalServerError, "refund processed but event publishing failed"
	}
	return result, true, false, 0, ""
}

func handleLedger(w http.ResponseWriter, r *http.Request) {
	rows, err := db.QueryContext(
		r.Context(),
		`SELECT le.id, le.payment_id, le.entry_type, le.debit, le.credit, le.currency, le.description, le.created_at
		 FROM ledger_entries le ORDER BY le.created_at DESC LIMIT 100`,
	)
	if err != nil {
		log.Printf("Failed to query ledger: %v", err)
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Entry struct {
		ID          int     `json:"id"`
		PaymentID   string  `json:"payment_id"`
		EntryType   string  `json:"entry_type"`
		Debit       float64 `json:"debit"`
		Credit      float64 `json:"credit"`
		Currency    string  `json:"currency"`
		Description string  `json:"description"`
		CreatedAt   string  `json:"created_at"`
	}

	entries := []Entry{}
	for rows.Next() {
		var e Entry

		if err := rows.Scan(
			&e.ID,
			&e.PaymentID,
			&e.EntryType,
			&e.Debit,
			&e.Credit,
			&e.Currency,
			&e.Description,
			&e.CreatedAt,
		); err != nil {
			log.Printf("Failed to scan ledger entry: %v", err)
			httpError(w, "failed to read ledger", http.StatusInternalServerError)
			return
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Failed while reading ledger entries: %v", err)
		httpError(w, "failed to read ledger", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func handleBalance(w http.ResponseWriter, r *http.Request) {
	customerID := strings.TrimPrefix(r.URL.Path, "/balance/")
	if customerID == "" {
		httpError(w, "customer id required", http.StatusBadRequest)
		return
	}

	var totalCharged, totalRefunded float64

	err := db.QueryRowContext(
		r.Context(),
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE customer_id = $1 AND status IN ('completed', 'partially_refunded', 'refunded') AND (method IS NULL OR method != 'refund')",
		customerID,
	).Scan(&totalCharged)
	if err != nil {
		log.Printf("Failed to calculate charged balance: %v", err)
		httpError(w, "failed to calculate balance", http.StatusInternalServerError)
		return
	}

	err = db.QueryRowContext(
		r.Context(),
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE customer_id = $1 AND method = 'refund'",
		customerID,
	).Scan(&totalRefunded)
	if err != nil {
		log.Printf("Failed to calculate refunded balance: %v", err)
		httpError(w, "failed to calculate balance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"customer_id":    customerID,
		"total_charged":  totalCharged,
		"total_refunded": totalRefunded,
		"net":            totalCharged - totalRefunded,
	})
}

func generatePaymentID() string {
	return fmt.Sprintf("pay_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func publishEvent(ctx context.Context, eventType string, payload map[string]interface{}) error {
	event := map[string]interface{}{
		"type":      eventType,
		"payload":   payload,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	messageBody := string(data)

	_, err = sqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &sqsQueueURL,
		MessageBody: &messageBody,
	})
	if err != nil {
		return fmt.Errorf("send SQS message: %w", err)
	}

	log.Printf("Event -> SQS: %s", string(data))
	return nil
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
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
