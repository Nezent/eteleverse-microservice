package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// LogEntry represents a log entry to send to telemetry service
type LogEntry struct {
	ServiceName string         `json:"service_name"`
	Level       string         `json:"level"`
	Message     string         `json:"message"`
	Timestamp   time.Time      `json:"timestamp"`
	Fields      map[string]any `json:"fields,omitempty"`
}

// sendLog sends a log entry to the telemetry service
func sendLog(level, message string, fields map[string]any) {
	entry := LogEntry{
		ServiceName: "order-service",
		Level:       level,
		Message:     message,
		Timestamp:   time.Now().UTC(),
		Fields:      fields,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	// Send to telemetry service via API Gateway
	resp, err := http.Post(
		"http://api-gateway/api/v1/logs",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		log.Printf("Failed to send log to telemetry: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Telemetry service returned status: %d", resp.StatusCode)
	}
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	sendLog("info", "Health check endpoint called", map[string]interface{}{
		"endpoint": "/health",
		"method":   r.Method,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "order-service",
		"timestamp": time.Now().UTC(),
	})
}

// Create order endpoint (demo)
func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendLog("warn", "Invalid method for create order", map[string]interface{}{
			"method":   r.Method,
			"expected": "POST",
		})
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var order map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		sendLog("error", "Failed to decode order request", map[string]interface{}{
			"error": err.Error(),
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sendLog("info", "New order created", map[string]interface{}{
		"order_data": order,
		"order_id":   time.Now().Unix(),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"message":  "Order created successfully",
		"order_id": time.Now().Unix(),
		"data":     order,
	})
}

// List orders endpoint (demo)
func listOrdersHandler(w http.ResponseWriter, r *http.Request) {
	sendLog("info", "List orders endpoint called", map[string]interface{}{
		"endpoint": "/api/v1/orders",
	})

	orders := []map[string]interface{}{
		{
			"id":         1,
			"product":    "Widget Pro",
			"quantity":   5,
			"price":      99.99,
			"status":     "completed",
			"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		},
		{
			"id":         2,
			"product":    "Gadget Max",
			"quantity":   3,
			"price":      149.99,
			"status":     "pending",
			"created_at": time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"count":  len(orders),
		"orders": orders,
	})
}

func main() {
	fmt.Println("Order Service is starting...")

	// Send startup log
	sendLog("info", "Order Service starting up", map[string]interface{}{
		"version": "1.0.0",
		"port":    "8080",
	})

	// Setup HTTP routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createOrderHandler(w, r)
		} else if r.Method == http.MethodGet {
			listOrdersHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Root endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "order-service",
			"status":  "running",
			"version": "1.0.0",
		})
	})

	// Start server
	port := "8080"
	sendLog("info", "Order Service started successfully", map[string]interface{}{
		"port": port,
	})

	fmt.Printf("Order Service listening on port %s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		sendLog("error", "Server failed to start", map[string]interface{}{
			"error": err.Error(),
		})
		log.Fatal(err)
	}
}
