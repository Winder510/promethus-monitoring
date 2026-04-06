package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"mircrosvc-app/services/common"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type reserveRequest struct {
	OrderID string `json:"order_id"`
	SKU     string `json:"sku"`
	Qty     int    `json:"qty"`
}

type reserveResponse struct {
	Service    string `json:"service"`
	OrderID    string `json:"order_id"`
	SKU        string `json:"sku"`
	Qty        int    `json:"qty"`
	Reserved   bool   `json:"reserved"`
	Warehouse  string `json:"warehouse"`
	ReservedAt string `json:"reserved_at_utc"`
}

var (
	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "stock_http_requests_total",
		Help: "Total HTTP requests handled by the stock service.",
	}, []string{"path", "status"})
	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "stock_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path"})
)

func main() {
	prometheus.MustRegister(requestCount, requestDuration)

	ctx := context.Background()
	shutdown, err := common.InitTelemetry(ctx, "stock")
	if err != nil {
		log.Printf("telemetry disabled: %v", err)
	}
	if shutdown != nil {
		defer func() {
			if err := shutdown(context.Background()); err != nil {
				log.Printf("shutdown telemetry: %v", err)
			}
		}()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"service": "stock",
			"message": "stock service is running",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("/reserve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		start := time.Now()
		status := http.StatusOK
		defer func() {
			requestCount.WithLabelValues("/reserve", strconv.Itoa(status)).Inc()
			requestDuration.WithLabelValues("/reserve").Observe(time.Since(start).Seconds())
		}()

		var req reserveRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			status = http.StatusBadRequest
			respondError(w, status, "invalid JSON payload")
			return
		}
		if req.OrderID == "" || req.SKU == "" {
			status = http.StatusBadRequest
			respondError(w, status, "order_id and sku are required")
			return
		}
		if req.Qty <= 0 {
			req.Qty = 1
		}

		respondJSON(w, status, reserveResponse{
			Service:    "stock",
			OrderID:    req.OrderID,
			SKU:        req.SKU,
			Qty:        req.Qty,
			Reserved:   true,
			Warehouse:  "warehouse-a",
			ReservedAt: time.Now().UTC().Format(time.RFC3339),
		})
	})

	port := envOrDefault("PORT", "8082")
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           common.WrapHandler(mux, "stock-server"),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("stock listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]any{"error": message})
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
