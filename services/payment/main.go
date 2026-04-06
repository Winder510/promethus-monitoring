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

type paymentRequest struct {
	OrderID string  `json:"order_id"`
	SKU     string  `json:"sku"`
	Qty     int     `json:"qty"`
	Amount  float64 `json:"amount"`
}

type paymentResponse struct {
	Service      string         `json:"service"`
	OrderID      string         `json:"order_id"`
	Status       string         `json:"status"`
	Authorized   bool           `json:"authorized"`
	Stock        map[string]any `json:"stock"`
	ChargedAtUTC string         `json:"charged_at_utc"`
}

var (
	requestCount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_http_requests_total",
		Help: "Total HTTP requests handled by the payment service.",
	}, []string{"path", "status"})
	requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "payment_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"path"})
)

func main() {
	prometheus.MustRegister(requestCount, requestDuration)

	ctx := context.Background()
	shutdown, err := common.InitTelemetry(ctx, "payment")
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
	stockURL := envOrDefault("STOCK_URL", "http://localhost:8082")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{
			"service": "payment",
			"message": "payment service is running",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("/metrics", promhttp.Handler().ServeHTTP)
	mux.HandleFunc("/pay", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		start := time.Now()
		status := http.StatusOK
		defer func() {
			requestCount.WithLabelValues("/pay", strconv.Itoa(status)).Inc()
			requestDuration.WithLabelValues("/pay").Observe(time.Since(start).Seconds())
		}()

		var req paymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			status = http.StatusBadRequest
			respondError(w, status, "invalid JSON payload")
			return
		}
		if req.OrderID == "" {
			status = http.StatusBadRequest
			respondError(w, status, "order_id is required")
			return
		}
		if req.SKU == "" {
			status = http.StatusBadRequest
			respondError(w, status, "sku is required")
			return
		}
		if req.Qty <= 0 {
			req.Qty = 1
		}

		var stockResp map[string]any
		stockReq := map[string]any{
			"order_id": req.OrderID,
			"sku":      req.SKU,
			"qty":      req.Qty,
		}
		if err := common.PostJSON(r.Context(), client, stockURL+"/reserve", stockReq, &stockResp); err != nil {
			status = http.StatusBadGateway
			respondError(w, status, err.Error())
			return
		}

		resp := paymentResponse{
			Service:      "payment",
			OrderID:      req.OrderID,
			Status:       "charged",
			Authorized:   true,
			Stock:        stockResp,
			ChargedAtUTC: time.Now().UTC().Format(time.RFC3339),
		}
		respondJSON(w, status, resp)
	})

	port := envOrDefault("PORT", "8081")
	server := &http.Server{
		Addr:              ":" + port,
		Handler:           common.WrapHandler(mux, "payment-server"),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("payment listening on %s", server.Addr)
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
