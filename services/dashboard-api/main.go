package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

//go:embed static
var staticFiles embed.FS

var db *sql.DB
var apiGatewayProxy *httputil.ReverseProxy

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

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	db.SetConnMaxLifetime(5 * time.Minute)

	waitForDB()

	apiGatewayURL := getEnv("API_GATEWAY_URL", "http://api-gateway:8080")
	apiGatewayTarget, err := url.ParseRequestURI(apiGatewayURL)
	if err != nil || apiGatewayTarget.Scheme == "" || apiGatewayTarget.Host == "" {
		log.Fatalf("invalid API_GATEWAY_URL: %q", apiGatewayURL)
	}
	apiGatewayProxy = httputil.NewSingleHostReverseProxy(apiGatewayTarget)
	apiGatewayProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("API gateway proxy error: %v", err)
		httpError(w, "service unavailable", http.StatusBadGateway)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/healthz", handleHealth)
	mux.HandleFunc("/dashboard/healthz", handleHealth)
	mux.HandleFunc("/dashboard/summary", handleSummary)
	mux.HandleFunc("/dashboard/orders/stats", handleOrderStats)
	mux.HandleFunc("/dashboard/revenue", handleRevenue)
	mux.HandleFunc("/dashboard/inventory/alerts", handleInventoryAlerts)
	mux.HandleFunc("/dashboard/shipping/overview", handleShippingOverview)
	mux.Handle("/api/", apiGatewayProxy)
	mux.Handle("/auth/", apiGatewayProxy)

	// Serve frontend UI
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	port := getEnv("PORT", "8086")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Dashboard API listening on :%s", port)
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "ok"

	if err := db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  status,
		"service": "dashboard-api",
	})
}

func handleSummary(w http.ResponseWriter, r *http.Request) {
	today := time.Now().UTC().Truncate(24 * time.Hour)

	summary := map[string]interface{}{}

	// Order counts
	var totalOrders, ordersToday int

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM orders",
	).Scan(&totalOrders); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM orders WHERE created_at >= $1",
		today,
	).Scan(&ordersToday); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Orders by status
	statusCounts := map[string]int{}

	rows, err := db.Query(
		"SELECT status, COUNT(*) FROM orders GROUP BY status",
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int

		if err := rows.Scan(&status, &count); err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}

		statusCounts[status] = count
	}

	if err := rows.Err(); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Revenue (charges minus refunds)
	var totalCharges, totalRefunds, todayCharges, todayRefunds float64

	if err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status IN ('completed','partially_refunded','refunded') AND (method IS NULL OR method != 'refund')",
	).Scan(&totalCharges); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	if err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'completed' AND method = 'refund'",
	).Scan(&totalRefunds); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	if err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status IN ('completed','partially_refunded','refunded') AND (method IS NULL OR method != 'refund') AND created_at >= $1",
		today,
	).Scan(&todayCharges); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	if err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'completed' AND method = 'refund' AND created_at >= $1",
		today,
	).Scan(&todayRefunds); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	totalRevenue := totalCharges - totalRefunds
	revenueToday := todayCharges - todayRefunds

	// Product count
	var totalProducts int

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM products",
	).Scan(&totalProducts); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Low stock count
	var lowStockCount int

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM products WHERE (stock - reserved) < 10",
	).Scan(&lowStockCount); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Active shipments
	var activeShipments int

	if err := db.QueryRow(
		"SELECT COUNT(*) FROM shipments WHERE status NOT IN ('delivered', 'cancelled')",
	).Scan(&activeShipments); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	summary["orders"] = map[string]interface{}{
		"total":     totalOrders,
		"today":     ordersToday,
		"by_status": statusCounts,
	}

	summary["revenue"] = map[string]interface{}{
		"total":    totalRevenue,
		"today":    revenueToday,
		"currency": "GBP",
	}

	summary["inventory"] = map[string]interface{}{
		"total_products": totalProducts,
		"low_stock":      lowStockCount,
	}

	summary["shipping"] = map[string]interface{}{
		"active_shipments": activeShipments,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func handleOrderStats(w http.ResponseWriter, r *http.Request) {
	// Orders per day for last 30 days
	rows, err := db.Query(
		`SELECT DATE(created_at) as day, COUNT(*), COALESCE(SUM(total), 0)
		 FROM orders
		 WHERE created_at >= NOW() - INTERVAL '30 days'
		 GROUP BY DATE(created_at)
		 ORDER BY day DESC`,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type DayStat struct {
		Date    string  `json:"date"`
		Orders  int     `json:"orders"`
		Revenue float64 `json:"revenue"`
	}

	stats := []DayStat{}

	for rows.Next() {
		var s DayStat

		if err := rows.Scan(&s.Date, &s.Orders, &s.Revenue); err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}

		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleRevenue(w http.ResponseWriter, r *http.Request) {
	// Revenue breakdown
	var total, refunded, net float64

	if err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status IN ('completed', 'partially_refunded', 'refunded') AND (method IS NULL OR method != 'refund')",
	).Scan(&total); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	if err := db.QueryRow(
		"SELECT COALESCE(SUM(amount), 0) FROM payments WHERE status = 'completed' AND method = 'refund'",
	).Scan(&refunded); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	net = total - refunded

	// Revenue by day (last 7 days)
	rows, err := db.Query(
		`SELECT DATE(created_at) as day, COALESCE(SUM(amount), 0)
		 FROM payments
		 WHERE status IN ('completed', 'partially_refunded', 'refunded') AND (method IS NULL OR method != 'refund')
		 AND created_at >= NOW() - INTERVAL '7 days'
		 GROUP BY DATE(created_at)
		 ORDER BY day DESC`,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type DayRevenue struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
	}

	daily := []DayRevenue{}

	for rows.Next() {
		var d DayRevenue

		if err := rows.Scan(&d.Date, &d.Revenue); err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}

		daily = append(daily, d)
	}

	if err := rows.Err(); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":    total,
		"refunded": refunded,
		"net":      net,
		"currency": "GBP",
		"daily":    daily,
	})
}

func handleInventoryAlerts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(
		`SELECT id, name, sku, stock, reserved, (stock - reserved) as available
		 FROM products
		 WHERE (stock - reserved) < 10
		 ORDER BY (stock - reserved) ASC`,
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Alert struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		SKU       string `json:"sku"`
		Stock     int    `json:"stock"`
		Reserved  int    `json:"reserved"`
		Available int    `json:"available"`
	}

	alerts := []Alert{}

	for rows.Next() {
		var a Alert

		if err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.SKU,
			&a.Stock,
			&a.Reserved,
			&a.Available,
		); err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}

		alerts = append(alerts, a)
	}

	if err := rows.Err(); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

func handleShippingOverview(w http.ResponseWriter, r *http.Request) {
	// Shipments by status
	statusCounts := map[string]int{}

	rows, err := db.Query(
		"SELECT status, COUNT(*) FROM shipments GROUP BY status",
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int

		if err := rows.Scan(&status, &count); err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}

		statusCounts[status] = count
	}

	if err := rows.Err(); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Carrier breakdown
	carrierCounts := map[string]int{}

	rows2, err := db.Query(
		"SELECT carrier, COUNT(*) FROM shipments GROUP BY carrier",
	)
	if err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer rows2.Close()

	for rows2.Next() {
		var carrier string
		var count int

		if err := rows2.Scan(&carrier, &count); err != nil {
			httpError(w, "query failed", http.StatusInternalServerError)
			return
		}

		carrierCounts[carrier] = count
	}

	if err := rows2.Err(); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Average delivery time
	var avgDeliveryHours float64

	if err := db.QueryRow(
		`SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (delivered_at - shipped_at)) / 3600), 0)
		 FROM shipments WHERE delivered_at IS NOT NULL AND shipped_at IS NOT NULL`,
	).Scan(&avgDeliveryHours); err != nil {
		httpError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"by_status":          statusCounts,
		"by_carrier":         carrierCounts,
		"avg_delivery_hours": avgDeliveryHours,
	})
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
