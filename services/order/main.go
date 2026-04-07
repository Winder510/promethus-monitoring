package main

import (
	"context"
	"encoding/json"
	"fmt"
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

type orderRequest struct {
	OrderID string  `json:"order_id"`
	SKU     string  `json:"sku"`
	Amount  float64 `json:"amount"`
	Qty     int     `json:"qty"`
}

type orderResponse struct {
	Service   string         `json:"service"`
	OrderID   string         `json:"order_id"`
	Status    string         `json:"status"`
	Payment   map[string]any `json:"payment"`
	Timestamp string         `json:"timestamp"`
}

var (
	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "order_http_requests_total",
		Help: "Total HTTP requests handled by the order service.",
	}, []string{"path", "status"})
	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "order_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path"})
)

func main() {
	prometheus.MustRegister(requestCount, requestDuration)

	ctx := context.Background()
	shutdown, err := common.InitTelemetry(ctx, "order")
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
	client := common.NewHTTPClient()
	paymentURL := envOrDefault("PAYMENT_URL", "http://localhost:8081")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"service": "order",
			"message": "order service is running",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		start := time.Now()
		status := http.StatusOK
		defer func() {
			requestCount.WithLabelValues("/order", strconv.Itoa(status)).Inc()
			requestDuration.WithLabelValues("/order").Observe(time.Since(start).Seconds())
		}()

		var req orderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			status = http.StatusBadRequest
			respondError(w, status, "invalid JSON payload")
			return
		}
		if req.OrderID == "" {
			req.OrderID = fmt.Sprintf("ord-%d", time.Now().UnixNano())
		}
		if req.Qty <= 0 {
			req.Qty = 1
		}
		if req.SKU == "" {
			status = http.StatusBadRequest
			respondError(w, status, "sku is required")
			return
		}

		var paymentResp map[string]any
		paymentReq := map[string]any{
			"order_id": req.OrderID,
			"sku":      req.SKU,
			"qty":      req.Qty,
			"amount":   req.Amount,
		}
		if err := common.PostJSON(r.Context(), client, paymentURL+"/pay", paymentReq, &paymentResp); err != nil {
			status = http.StatusBadGateway
			respondError(w, status, err.Error())
			return
		}

		log.Print("Creat order success")
		respondJSON(w, status, orderResponse{
			Service:   "order",
			OrderID:   req.OrderID,
			Status:    "completed",
			Payment:   paymentResp,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})
	})

	port := envOrDefault("PORT", "8080")
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           common.WrapHandler(mux, "order-server"),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("order listening on %s", server.Addr)
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
