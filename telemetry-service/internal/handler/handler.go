package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Nezent/microservice-template/telemetry-service/internal/logger"
	"github.com/Nezent/microservice-template/telemetry-service/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	metrics *metrics.Metrics
}

// NewHandler creates a new handler instance
func NewHandler() *Handler {
	return &Handler{
		metrics: metrics.GetMetrics(),
	}
}

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "telemetry-service",
	})
}

// LogHandler handles incoming log requests
func (h *Handler) LogHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.HTTPRequestsInFlight.Inc()
	defer h.metrics.HTTPRequestsInFlight.Dec()

	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs", http.StatusMethodNotAllowed, start)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, "Failed to read request body", http.StatusBadRequest)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs", http.StatusBadRequest, start)
		return
	}
	defer r.Body.Close()

	entry, err := logger.ParseLogEntry(body)
	if err != nil {
		h.respondError(w, "Failed to parse log entry: "+err.Error(), http.StatusBadRequest)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs", http.StatusBadRequest, start)
		h.metrics.LogsErrors.WithLabelValues(entry.ServiceName, "parse_error").Inc()
		return
	}

	// Record metrics
	h.metrics.LogsReceived.WithLabelValues(entry.ServiceName, entry.Level).Inc()

	// Log the entry
	if err := logger.LogFromService(*entry); err != nil {
		h.respondError(w, "Failed to process log entry: "+err.Error(), http.StatusInternalServerError)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs", http.StatusInternalServerError, start)
		h.metrics.LogsErrors.WithLabelValues(entry.ServiceName, "processing_error").Inc()
		return
	}

	h.metrics.LogsProcessed.WithLabelValues(entry.ServiceName, entry.Level).Inc()

	h.respondSuccess(w, map[string]interface{}{
		"status":  "success",
		"message": "Log entry processed successfully",
	})
	h.recordHTTPMetrics(r.Method, "/api/v1/logs", http.StatusOK, start)
}

// LogBatchHandler handles batch log requests
func (h *Handler) LogBatchHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.HTTPRequestsInFlight.Inc()
	defer h.metrics.HTTPRequestsInFlight.Dec()

	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs/batch", http.StatusMethodNotAllowed, start)
		return
	}

	var entries []logger.LogEntry
	if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
		h.respondError(w, "Failed to parse log entries: "+err.Error(), http.StatusBadRequest)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs/batch", http.StatusBadRequest, start)
		return
	}
	defer r.Body.Close()

	h.metrics.LogBatchSize.Observe(float64(len(entries)))

	if err := logger.LogBatch(entries); err != nil {
		h.respondError(w, "Failed to process log batch: "+err.Error(), http.StatusInternalServerError)
		h.recordHTTPMetrics(r.Method, "/api/v1/logs/batch", http.StatusInternalServerError, start)
		return
	}

	for _, entry := range entries {
		h.metrics.LogsReceived.WithLabelValues(entry.ServiceName, entry.Level).Inc()
		h.metrics.LogsProcessed.WithLabelValues(entry.ServiceName, entry.Level).Inc()
	}

	h.respondSuccess(w, map[string]interface{}{
		"status":  "success",
		"message": "Log batch processed successfully",
		"count":   len(entries),
	})
	h.recordHTTPMetrics(r.Method, "/api/v1/logs/batch", http.StatusOK, start)
}

// MetricsHandler handles incoming metrics from external services
func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.metrics.HTTPRequestsInFlight.Inc()
	defer h.metrics.HTTPRequestsInFlight.Dec()

	if r.Method != http.MethodPost {
		h.respondError(w, "Method not allowed", http.StatusMethodNotAllowed)
		h.recordHTTPMetrics(r.Method, "/api/v1/metrics", http.StatusMethodNotAllowed, start)
		return
	}

	var entry metrics.MetricEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		h.respondError(w, "Failed to parse metric entry: "+err.Error(), http.StatusBadRequest)
		h.recordHTTPMetrics(r.Method, "/api/v1/metrics", http.StatusBadRequest, start)
		return
	}
	defer r.Body.Close()

	metrics.RecordMetric(entry)

	h.respondSuccess(w, map[string]interface{}{
		"status":  "success",
		"message": "Metric recorded successfully",
	})
	h.recordHTTPMetrics(r.Method, "/api/v1/metrics", http.StatusOK, start)
}

// PrometheusMetricsHandler exposes metrics for Prometheus scraping
func (h *Handler) PrometheusMetricsHandler() http.Handler {
	return promhttp.HandlerFor(
		metrics.GetRegistry(),
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}

// respondError sends an error response
func (h *Handler) respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "error",
		"error":  message,
	})
}

// respondSuccess sends a success response
func (h *Handler) respondSuccess(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// recordHTTPMetrics records HTTP request metrics
func (h *Handler) recordHTTPMetrics(method, endpoint string, status int, start time.Time) {
	duration := time.Since(start).Seconds()
	h.metrics.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	h.metrics.HTTPRequestsTotal.WithLabelValues(method, endpoint, http.StatusText(status)).Inc()
}
