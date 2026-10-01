package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	waitForDB()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Health check
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			status := "ok"
			if err := db.Ping(); err != nil {
				status = "unhealthy"
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  status,
				"service": "scheduler",
			})
		})

		port := getEnv("HEALTH_PORT", "8091")
		log.Printf("Scheduler health check on :%s", port)
		http.ListenAndServe(":"+port, mux)
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down scheduler...")
		cancel()
	}()

	log.Println("Scheduler started")

	// Run scheduled jobs
	go runEvery(ctx, 1*time.Minute, "expire_reservations", expireReservations)
	go runEvery(ctx, 5*time.Minute, "abandoned_carts", detectAbandonedOrders)
	go runEvery(ctx, 15*time.Minute, "retry_failed_payments", retryFailedPayments)
	go runEvery(ctx, 1*time.Hour, "daily_digest", generateDigest)
	go runEvery(ctx, 30*time.Minute, "cleanup_old_events", cleanupOldEvents)

	<-ctx.Done()
	log.Println("Scheduler stopped")
}

func runEvery(ctx context.Context, interval time.Duration, name string, fn func() error) {
	log.Printf("Scheduled job '%s' every %s", name, interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	run := func() {
		log.Printf("Running job: %s", name)
		start := time.Now()

		if err := fn(); err != nil {
			log.Printf("Job '%s' failed: %v", name, err)
			return
		}

		log.Printf("Job '%s' completed in %s", name, time.Since(start))
	}

	// Run immediately on start
	run()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run()
		}
	}
}

// expireReservations releases inventory reservations that have expired
func expireReservations() error {
	rows, err := db.Query(
		`SELECT r.id, r.order_id, r.product_id, r.quantity
		 FROM reservations r
		 WHERE r.status = 'active' AND r.expires_at < NOW()`,
	)
	if err != nil {
		return fmt.Errorf("expire_reservations query error: %w", err)
	}
	defer rows.Close()

	count := 0
	failures := 0

	for rows.Next() {
		var reservationID int
		var orderID int
		var productID string
		var quantity int

		if err := rows.Scan(
			&reservationID,
			&orderID,
			&productID,
			&quantity,
		); err != nil {
			log.Printf("expire_reservations scan error: %v", err)
			failures++
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			log.Printf(
				"expire_reservations begin error for reservation %d: %v",
				reservationID,
				err,
			)
			failures++
			continue
		}

		var expiredProductID string
		var expiredQuantity int

		err = tx.QueryRow(
			`UPDATE reservations
			 SET status = 'expired'
			 WHERE id = $1 AND status = 'active'
			 RETURNING product_id, quantity`,
			reservationID,
		).Scan(&expiredProductID, &expiredQuantity)

		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			continue
		}

		if err != nil {
			_ = tx.Rollback()
			log.Printf(
				"expire_reservations reservation update error for reservation %d: %v",
				reservationID,
				err,
			)
			failures++
			continue
		}

		result, err := tx.Exec(
			`UPDATE products
			 SET reserved = GREATEST(reserved - $1, 0),
			     updated_at = NOW()
			 WHERE id = $2`,
			expiredQuantity,
			expiredProductID,
		)
		if err != nil {
			_ = tx.Rollback()
			log.Printf(
				"expire_reservations product update error for reservation %d: %v",
				reservationID,
				err,
			)
			failures++
			continue
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			log.Printf(
				"expire_reservations product rows affected error for reservation %d: %v",
				reservationID,
				err,
			)
			failures++
			continue
		}

		if rowsAffected != 1 {
			_ = tx.Rollback()
			log.Printf(
				"expire_reservations product %s not updated for reservation %d",
				expiredProductID,
				reservationID,
			)
			failures++
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Printf(
				"expire_reservations commit error for reservation %d: %v",
				reservationID,
				err,
			)
			failures++
			continue
		}

		count++

		_ = orderID
		_ = productID
		_ = quantity
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("expire_reservations iteration error: %w", err)
	}

	if count > 0 {
		log.Printf("Expired %d reservations", count)
	}

	if failures > 0 {
		return fmt.Errorf(
			"expire_reservations completed with %d failed operation(s)",
			failures,
		)
	}

	return nil
}

// detectAbandonedOrders finds orders stuck in pending for too long
func detectAbandonedOrders() error {
	cutoff := time.Now().Add(-30 * time.Minute)

	rows, err := db.Query(
		`SELECT id, customer_id FROM orders
		 WHERE status = 'pending' AND created_at < $1`,
		cutoff,
	)
	if err != nil {
		return fmt.Errorf("abandoned_carts query error: %w", err)
	}
	defer rows.Close()

	count := 0
	failures := 0

	for rows.Next() {
		var orderID int
		var customerID string

		if err := rows.Scan(&orderID, &customerID); err != nil {
			log.Printf("abandoned_carts scan error: %v", err)
			failures++
			continue
		}

		result, err := db.Exec(
			`UPDATE orders
			 SET status = 'cancelled', updated_at = NOW()
			 WHERE id = $1 AND status = 'pending'`,
			orderID,
		)
		if err != nil {
			log.Printf(
				"abandoned_carts update error for order %d: %v",
				orderID,
				err,
			)
			failures++
			continue
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf(
				"abandoned_carts rows affected error for order %d: %v",
				orderID,
				err,
			)
			failures++
			continue
		}

		if rowsAffected != 1 {
			log.Printf(
				"abandoned_carts order %d was not cancelled because its state changed",
				orderID,
			)
			continue
		}

		log.Printf(
			"Cancelled abandoned order #%d (customer: %s)",
			orderID,
			customerID,
		)

		count++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("abandoned_carts iteration error: %w", err)
	}

	if count > 0 {
		log.Printf("Cancelled %d abandoned orders", count)
	}

	if failures > 0 {
		return fmt.Errorf(
			"abandoned_carts completed with %d failed operation(s)",
			failures,
		)
	}

	return nil
}

// retryFailedPayments attempts to re-process failed payments
func retryFailedPayments() error {
	cutoff := time.Now().Add(-1 * time.Hour)

	var count int

	err := db.QueryRow(
		`SELECT COUNT(*) FROM payments
		 WHERE status = 'failed' AND created_at > $1`,
		cutoff,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("retry_failed_payments query error: %w", err)
	}

	if count > 0 {
		log.Printf("Found %d failed payments eligible for retry", count)

		// In production: re-queue payment attempts via SQS
		// POST to payment-service/charge for each
	}

	return nil
}

// generateDigest creates summary stats
func generateDigest() error {
	var totalOrders, pendingOrders, completedOrders int
	var totalRevenue float64

	today := time.Now().UTC().Truncate(24 * time.Hour)

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM orders WHERE created_at >= $1",
		today,
	).Scan(&totalOrders); err != nil {
		return fmt.Errorf("daily_digest total orders query error: %w", err)
	}

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM orders WHERE status = 'pending'",
	).Scan(&pendingOrders); err != nil {
		return fmt.Errorf("daily_digest pending orders query error: %w", err)
	}

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM orders WHERE status = 'delivered' AND updated_at >= $1",
		today,
	).Scan(&completedOrders); err != nil {
		return fmt.Errorf("daily_digest completed orders query error: %w", err)
	}

	if err := db.QueryRow(
		`SELECT COALESCE(SUM(amount), 0)
		 FROM payments
		 WHERE status = 'completed' AND created_at >= $1`,
		today,
	).Scan(&totalRevenue); err != nil {
		return fmt.Errorf("daily_digest revenue query error: %w", err)
	}

	log.Printf(
		"Daily digest - Orders today: %d, Pending: %d, Completed: %d, Revenue: %.2f",
		totalOrders,
		pendingOrders,
		completedOrders,
		totalRevenue,
	)

	// In production: send digest via notification-service

	return nil
}

// cleanupOldEvents removes old tracking/audit data
func cleanupOldEvents() error {
	cutoff := time.Now().AddDate(0, 0, -90)

	result, err := db.Exec(
		"DELETE FROM order_events WHERE created_at < $1",
		cutoff,
	)
	if err != nil {
		return fmt.Errorf("cleanup error: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("cleanup rows affected error: %w", err)
	}

	if rows > 0 {
		log.Printf("Cleaned up %d old order events", rows)
	}

	return nil
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
