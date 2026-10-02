package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

type Product struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	SKU       string  `json:"sku"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	Reserved  int     `json:"reserved"`
	Available int     `json:"available"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type Reservation struct {
	ID        int    `json:"id"`
	OrderID   int    `json:"order_id"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

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

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
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
	mux.HandleFunc("/products", handleProducts)
	mux.HandleFunc("/products/", handleProduct)
	mux.HandleFunc("/reserve", handleReserve)
	mux.HandleFunc("/release", handleRelease)
	mux.HandleFunc("/low-stock", handleLowStock)

	port := getEnv("PORT", "8082")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Inventory service listening on :%s", port)
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
		`CREATE TABLE IF NOT EXISTS products (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			sku VARCHAR(100) UNIQUE NOT NULL,
			price DECIMAL(10,2) NOT NULL,
			stock INTEGER NOT NULL DEFAULT 0,
			reserved INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_products_sku ON products(sku)`,
		`CREATE TABLE IF NOT EXISTS reservations (
			id SERIAL PRIMARY KEY,
			order_id INTEGER NOT NULL,
			product_id VARCHAR(50) NOT NULL REFERENCES products(id),
			quantity INTEGER NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'active',
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reservations_order ON reservations(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reservations_status ON reservations(status)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("Inventory service migrations complete")
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"

	if err := db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  status,
			"service": "inventory-service",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  status,
		"service": "inventory-service",
	})
}

func handleProducts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows, err := db.QueryContext(
			r.Context(),
			"SELECT id, name, sku, price, stock, reserved, created_at, updated_at FROM products ORDER BY name LIMIT 100",
		)
		if err != nil {
			log.Printf("product query failed: %v", err)
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		products := []Product{}

		for rows.Next() {
			var p Product

			if err := rows.Scan(
				&p.ID,
				&p.Name,
				&p.SKU,
				&p.Price,
				&p.Stock,
				&p.Reserved,
				&p.CreatedAt,
				&p.UpdatedAt,
			); err != nil {
				log.Printf("product scan failed: %v", err)
				httpError(w, "failed to read products", http.StatusInternalServerError)
				return
			}

			p.Available = p.Stock - p.Reserved
			products = append(products, p)
		}

		if err := rows.Err(); err != nil {
			log.Printf("product iteration failed: %v", err)
			httpError(w, "failed to read products", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(products)

	case http.MethodPost:
		var p struct {
			ID    string  `json:"id"`
			Name  string  `json:"name"`
			SKU   string  `json:"sku"`
			Price float64 `json:"price"`
			Stock int     `json:"stock"`
		}

		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}

		p.ID = strings.TrimSpace(p.ID)
		p.Name = strings.TrimSpace(p.Name)
		p.SKU = strings.TrimSpace(p.SKU)

		switch {
		case p.ID == "":
			httpError(w, "product id is required", http.StatusBadRequest)
			return
		case p.Name == "":
			httpError(w, "product name is required", http.StatusBadRequest)
			return
		case p.SKU == "":
			httpError(w, "product sku is required", http.StatusBadRequest)
			return
		case p.Price <= 0:
			httpError(w, "product price must be greater than zero", http.StatusBadRequest)
			return
		case p.Stock < 0:
			httpError(w, "product stock cannot be negative", http.StatusBadRequest)
			return
		}

		_, err := db.ExecContext(
			r.Context(),
			"INSERT INTO products (id, name, sku, price, stock) VALUES ($1, $2, $3, $4, $5)",
			p.ID,
			p.Name,
			p.SKU,
			p.Price,
			p.Stock,
		)
		if err != nil {
			log.Printf("failed to create product: %v", err)
			httpError(w, "failed to create product", http.StatusConflict)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":     p.ID,
			"status": "created",
		})

	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/products/"))
	if id == "" {
		httpError(w, "product id required", http.StatusBadRequest)
		return
	}

	var p Product

	err := db.QueryRowContext(
		r.Context(),
		"SELECT id, name, sku, price, stock, reserved, created_at, updated_at FROM products WHERE id = $1",
		id,
	).Scan(
		&p.ID,
		&p.Name,
		&p.SKU,
		&p.Price,
		&p.Stock,
		&p.Reserved,
		&p.CreatedAt,
		&p.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		httpError(w, "product not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("product lookup failed: %v", err)
		httpError(w, "failed to retrieve product", http.StatusInternalServerError)
		return
	}

	p.Available = p.Stock - p.Reserved

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(p)
}

func handleReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int `json:"order_id"`
		Items   []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrderID <= 0 {
		httpError(w, "order_id must be greater than zero", http.StatusBadRequest)
		return
	}

	if len(req.Items) == 0 {
		httpError(w, "at least one item is required", http.StatusBadRequest)
		return
	}

	for i := range req.Items {
		req.Items[i].ProductID = strings.TrimSpace(req.Items[i].ProductID)

		if req.Items[i].ProductID == "" {
			httpError(w, "product_id is required", http.StatusBadRequest)
			return
		}

		if req.Items[i].Quantity <= 0 {
			httpError(
				w,
				fmt.Sprintf("quantity for product %s must be greater than zero", req.Items[i].ProductID),
				http.StatusBadRequest,
			)
			return
		}
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("failed to begin reservation transaction: %v", err)
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	expiresAt := time.Now().Add(15 * time.Minute)

	for _, item := range req.Items {
		var stock, reserved int

		err := tx.QueryRowContext(
			r.Context(),
			"SELECT stock, reserved FROM products WHERE id = $1 FOR UPDATE",
			item.ProductID,
		).Scan(&stock, &reserved)

		if errors.Is(err, sql.ErrNoRows) {
			httpError(
				w,
				fmt.Sprintf("product %s not found", item.ProductID),
				http.StatusNotFound,
			)
			return
		}
		if err != nil {
			log.Printf("failed to lock product %s: %v", item.ProductID, err)
			httpError(w, "failed to retrieve product", http.StatusInternalServerError)
			return
		}

		if stock < 0 || reserved < 0 || reserved > stock {
			log.Printf(
				"invalid inventory state for product %s: stock=%d reserved=%d",
				item.ProductID,
				stock,
				reserved,
			)
			httpError(
				w,
				fmt.Sprintf("invalid inventory state for product %s", item.ProductID),
				http.StatusConflict,
			)
			return
		}

		available := stock - reserved

		if available < item.Quantity {
			httpError(
				w,
				fmt.Sprintf(
					"insufficient stock for %s: available %d, requested %d",
					item.ProductID,
					available,
					item.Quantity,
				),
				http.StatusConflict,
			)
			return
		}

		result, err := tx.ExecContext(
			r.Context(),
			`UPDATE products
			 SET reserved = reserved + $1, updated_at = NOW()
			 WHERE id = $2
			   AND reserved + $1 <= stock`,
			item.Quantity,
			item.ProductID,
		)
		if err != nil {
			log.Printf("reservation update failed for product %s: %v", item.ProductID, err)
			httpError(w, "reservation update failed", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("failed to verify reservation update for product %s: %v", item.ProductID, err)
			httpError(w, "reservation update failed", http.StatusInternalServerError)
			return
		}
		if rowsAffected != 1 {
			httpError(
				w,
				fmt.Sprintf("unable to reserve stock for %s", item.ProductID),
				http.StatusConflict,
			)
			return
		}

		if _, err := tx.ExecContext(
			r.Context(),
			`INSERT INTO reservations
			 (order_id, product_id, quantity, status, expires_at)
			 VALUES ($1, $2, $3, 'active', $4)`,
			req.OrderID,
			item.ProductID,
			item.Quantity,
			expiresAt,
		); err != nil {
			log.Printf("reservation insert failed for product %s: %v", item.ProductID, err)
			httpError(w, "reservation insert failed", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("reservation commit failed: %v", err)
		httpError(w, "reservation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":   req.OrderID,
		"status":     "reserved",
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func handleRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID int `json:"order_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.OrderID <= 0 {
		httpError(w, "order_id must be greater than zero", http.StatusBadRequest)
		return
	}

	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("failed to begin release transaction: %v", err)
		httpError(w, "transaction failed", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.QueryContext(
		r.Context(),
		`SELECT product_id, quantity
		 FROM reservations
		 WHERE order_id = $1 AND status = 'active'
		 ORDER BY product_id`,
		req.OrderID,
	)
	if err != nil {
		log.Printf("failed to query reservations for order %d: %v", req.OrderID, err)
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	type item struct {
		productID string
		quantity  int
	}

	var items []item

	for rows.Next() {
		var it item

		if err := rows.Scan(&it.productID, &it.quantity); err != nil {
			rows.Close()
			log.Printf("failed to scan reservation for order %d: %v", req.OrderID, err)
			httpError(w, "scan failed", http.StatusInternalServerError)
			return
		}

		if it.quantity <= 0 {
			rows.Close()
			log.Printf(
				"invalid reservation quantity for order %d product %s: %d",
				req.OrderID,
				it.productID,
				it.quantity,
			)
			httpError(w, "invalid reservation state", http.StatusConflict)
			return
		}

		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		log.Printf("reservation iteration failed for order %d: %v", req.OrderID, err)
		httpError(w, "iteration failed", http.StatusInternalServerError)
		return
	}

	rows.Close()

	for _, it := range items {
		var reserved int

		err := tx.QueryRowContext(
			r.Context(),
			"SELECT reserved FROM products WHERE id = $1 FOR UPDATE",
			it.productID,
		).Scan(&reserved)

		if errors.Is(err, sql.ErrNoRows) {
			httpError(
				w,
				fmt.Sprintf("product %s not found", it.productID),
				http.StatusNotFound,
			)
			return
		}
		if err != nil {
			log.Printf("failed to lock product %s during release: %v", it.productID, err)
			httpError(w, "release failed", http.StatusInternalServerError)
			return
		}

		if reserved < it.quantity {
			log.Printf(
				"invalid release state for product %s: reserved=%d release=%d",
				it.productID,
				reserved,
				it.quantity,
			)
			httpError(
				w,
				fmt.Sprintf("invalid reserved stock for product %s", it.productID),
				http.StatusConflict,
			)
			return
		}

		result, err := tx.ExecContext(
			r.Context(),
			`UPDATE products
			 SET reserved = reserved - $1, updated_at = NOW()
			 WHERE id = $2
			   AND reserved >= $1`,
			it.quantity,
			it.productID,
		)
		if err != nil {
			log.Printf("release update failed for product %s: %v", it.productID, err)
			httpError(w, "release update failed", http.StatusInternalServerError)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			log.Printf("failed to verify release update for product %s: %v", it.productID, err)
			httpError(w, "release update failed", http.StatusInternalServerError)
			return
		}
		if rowsAffected != 1 {
			httpError(
				w,
				fmt.Sprintf("unable to release stock for product %s", it.productID),
				http.StatusConflict,
			)
			return
		}
	}

	if _, err := tx.ExecContext(
		r.Context(),
		"UPDATE reservations SET status = 'released' WHERE order_id = $1 AND status = 'active'",
		req.OrderID,
	); err != nil {
		log.Printf("failed to update reservations for order %d: %v", req.OrderID, err)
		httpError(w, "release update failed", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("release commit failed for order %d: %v", req.OrderID, err)
		httpError(w, "commit failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id": req.OrderID,
		"released": len(items),
	})
}

func handleLowStock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	threshold := 10

	rows, err := db.QueryContext(
		r.Context(),
		`SELECT id, name, sku, stock, reserved
		 FROM products
		 WHERE (stock - reserved) < $1
		 ORDER BY (stock - reserved) ASC`,
		threshold,
	)
	if err != nil {
		log.Printf("low-stock query failed: %v", err)
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type LowStockItem struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SKU       string `json:"sku"`
		Stock     int    `json:"stock"`
		Reserved  int    `json:"reserved"`
		Available int    `json:"available"`
	}

	items := []LowStockItem{}

	for rows.Next() {
		var i LowStockItem

		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.SKU,
			&i.Stock,
			&i.Reserved,
		); err != nil {
			log.Printf("low-stock scan failed: %v", err)
			httpError(w, "failed to read low-stock products", http.StatusInternalServerError)
			return
		}

		i.Available = i.Stock - i.Reserved
		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		log.Printf("low-stock iteration failed: %v", err)
		httpError(w, "failed to read low-stock products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
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
