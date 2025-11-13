package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	once sync.Once
	reg  *prometheus.Registry
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge

	// Log metrics
	LogsReceived  *prometheus.CounterVec
	LogsProcessed *prometheus.CounterVec
	LogsErrors    *prometheus.CounterVec
	LogBatchSize  prometheus.Histogram

	// Custom metrics from external services
	CustomCounters   map[string]*prometheus.CounterVec
	CustomGauges     map[string]*prometheus.GaugeVec
	CustomHistograms map[string]*prometheus.HistogramVec

	mu sync.RWMutex
}

var metrics *Metrics

// InitMetrics initializes Prometheus metrics
func InitMetrics() *Metrics {
	once.Do(func() {
		reg = prometheus.NewRegistry()

		factory := promauto.With(reg)

		metrics = &Metrics{
			HTTPRequestsTotal: factory.NewCounterVec(
				prometheus.CounterOpts{
					Name: "telemetry_http_requests_total",
					Help: "Total number of HTTP requests",
				},
				[]string{"method", "endpoint", "status"},
			),
			HTTPRequestDuration: factory.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "telemetry_http_request_duration_seconds",
					Help:    "HTTP request duration in seconds",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"method", "endpoint"},
			),
			HTTPRequestsInFlight: factory.NewGauge(
				prometheus.GaugeOpts{
					Name: "telemetry_http_requests_in_flight",
					Help: "Number of HTTP requests currently being processed",
				},
			),
			LogsReceived: factory.NewCounterVec(
				prometheus.CounterOpts{
					Name: "telemetry_logs_received_total",
					Help: "Total number of logs received",
				},
				[]string{"service_name", "level"},
			),
			LogsProcessed: factory.NewCounterVec(
				prometheus.CounterOpts{
					Name: "telemetry_logs_processed_total",
					Help: "Total number of logs processed successfully",
				},
				[]string{"service_name", "level"},
			),
			LogsErrors: factory.NewCounterVec(
				prometheus.CounterOpts{
					Name: "telemetry_logs_errors_total",
					Help: "Total number of log processing errors",
				},
				[]string{"service_name", "error_type"},
			),
			LogBatchSize: factory.NewHistogram(
				prometheus.HistogramOpts{
					Name:    "telemetry_log_batch_size",
					Help:    "Size of log batches received",
					Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
				},
			),
			CustomCounters:   make(map[string]*prometheus.CounterVec),
			CustomGauges:     make(map[string]*prometheus.GaugeVec),
			CustomHistograms: make(map[string]*prometheus.HistogramVec),
		}
	})
	return metrics
}

// GetMetrics returns the initialized metrics instance
func GetMetrics() *Metrics {
	if metrics == nil {
		return InitMetrics()
	}
	return metrics
}

// GetRegistry returns the Prometheus registry
func GetRegistry() *prometheus.Registry {
	if reg == nil {
		InitMetrics()
	}
	return reg
}

// RecordCustomCounter records a custom counter metric from external services
func (m *Metrics) RecordCustomCounter(name string, labels map[string]string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	counter, exists := m.CustomCounters[name]
	if !exists {
		labelNames := make([]string, 0, len(labels)+1)
		labelNames = append(labelNames, "service_name")
		for key := range labels {
			labelNames = append(labelNames, key)
		}

		counter = promauto.With(reg).NewCounterVec(
			prometheus.CounterOpts{
				Name: "custom_" + name,
				Help: "Custom counter metric from external service",
			},
			labelNames,
		)
		m.CustomCounters[name] = counter
	}

	labelValues := make([]string, 0, len(labels)+1)
	labelValues = append(labelValues, labels["service_name"])
	delete(labels, "service_name")
	for _, value := range labels {
		labelValues = append(labelValues, value)
	}

	counter.WithLabelValues(labelValues...).Add(value)
}

// RecordCustomGauge records a custom gauge metric from external services
func (m *Metrics) RecordCustomGauge(name string, labels map[string]string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gauge, exists := m.CustomGauges[name]
	if !exists {
		labelNames := make([]string, 0, len(labels)+1)
		labelNames = append(labelNames, "service_name")
		for key := range labels {
			labelNames = append(labelNames, key)
		}

		gauge = promauto.With(reg).NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "custom_" + name,
				Help: "Custom gauge metric from external service",
			},
			labelNames,
		)
		m.CustomGauges[name] = gauge
	}

	labelValues := make([]string, 0, len(labels)+1)
	labelValues = append(labelValues, labels["service_name"])
	delete(labels, "service_name")
	for _, value := range labels {
		labelValues = append(labelValues, value)
	}

	gauge.WithLabelValues(labelValues...).Set(value)
}

// RecordCustomHistogram records a custom histogram metric from external services
func (m *Metrics) RecordCustomHistogram(name string, labels map[string]string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	histogram, exists := m.CustomHistograms[name]
	if !exists {
		labelNames := make([]string, 0, len(labels)+1)
		labelNames = append(labelNames, "service_name")
		for key := range labels {
			labelNames = append(labelNames, key)
		}

		histogram = promauto.With(reg).NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "custom_" + name,
				Help:    "Custom histogram metric from external service",
				Buckets: prometheus.DefBuckets,
			},
			labelNames,
		)
		m.CustomHistograms[name] = histogram
	}

	labelValues := make([]string, 0, len(labels)+1)
	labelValues = append(labelValues, labels["service_name"])
	delete(labels, "service_name")
	for _, value := range labels {
		labelValues = append(labelValues, value)
	}

	histogram.WithLabelValues(labelValues...).Observe(value)
}

// MetricEntry represents a metric entry from external services
type MetricEntry struct {
	ServiceName string            `json:"service_name"`
	MetricName  string            `json:"metric_name"`
	MetricType  string            `json:"metric_type"` // counter, gauge, histogram
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// RecordMetric processes and records metrics from external services
func RecordMetric(entry MetricEntry) {
	m := GetMetrics()

	if entry.Labels == nil {
		entry.Labels = make(map[string]string)
	}
	entry.Labels["service_name"] = entry.ServiceName

	switch entry.MetricType {
	case "counter":
		m.RecordCustomCounter(entry.MetricName, entry.Labels, entry.Value)
	case "gauge":
		m.RecordCustomGauge(entry.MetricName, entry.Labels, entry.Value)
	case "histogram":
		m.RecordCustomHistogram(entry.MetricName, entry.Labels, entry.Value)
	}
}
