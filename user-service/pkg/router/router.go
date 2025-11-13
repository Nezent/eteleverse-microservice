package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter() *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.Recoverer)

	// Request size limiting (prevent large payloads)
	router.Use(middleware.RequestSize(1024 * 1024)) // 1MB limit

	// Compression middleware (before security headers)
	router.Use(middleware.Compress(5)) // gzip compression

	// Timeout middleware
	router.Use(middleware.Timeout(30 * time.Second))

	// Content type middleware for API responses
	router.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Remove trailing slashes
	router.Use(middleware.RedirectSlashes)

	// Liveness probe endpoint for Kubernetes
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"alive"}`))
		if err != nil {
			http.Error(w, `{"status":"error"}`, http.StatusInternalServerError)
		}
	})
	return router
}
