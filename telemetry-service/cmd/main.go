package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Nezent/microservice-template/telemetry-service/internal/handler"
	"github.com/Nezent/microservice-template/telemetry-service/internal/logger"
	"github.com/Nezent/microservice-template/telemetry-service/internal/metrics"
	"github.com/gorilla/mux"
)

func main() {
	// Initialize logger
	if err := logger.InitLogger("info", "json"); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	zapLogger := logger.GetLogger()
	zapLogger.Info("Starting Telemetry Service...")

	// Initialize metrics
	metrics.InitMetrics()
	zapLogger.Info("Metrics initialized")

	// Create handler
	h := handler.NewHandler()

	// Setup router
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", h.HealthCheck).Methods("GET")
	api.HandleFunc("/logs", h.LogHandler).Methods("POST")
	api.HandleFunc("/logs/batch", h.LogBatchHandler).Methods("POST")
	api.HandleFunc("/metrics", h.MetricsHandler).Methods("POST")

	// Prometheus metrics endpoint
	router.Handle("/metrics", h.PrometheusMetricsHandler())

	// Root health check
	router.HandleFunc("/", h.HealthCheck).Methods("GET")

	// Server configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		zapLogger.Info(fmt.Sprintf("Server starting on port %s", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Fatal(fmt.Sprintf("Server failed to start: %v", err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Fatal(fmt.Sprintf("Server forced to shutdown: %v", err))
	}

	zapLogger.Info("Server stopped")
}
