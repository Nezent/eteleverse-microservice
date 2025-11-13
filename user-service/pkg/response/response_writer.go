package response

import (
	"encoding/json"
	"net/http"
)

// APIResponse is a standard response structure.
type APIResponse struct {
	Success    bool   `json:"success"`
	StatusCode int    `json:"status_code"`
	Data       any    `json:"data,omitempty"`
	Error      string `json:"error,omitempty"`
}

// WriteSuccess writes a successful response with data.
func WriteSuccess(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(APIResponse{
		Success:    true,
		StatusCode: statusCode,
		Data:       data,
	})
	if err != nil {
		http.Error(w, `{"success":false,"status_code":500,"error":"Internal Server Error"}`, http.StatusInternalServerError)
	}
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, errMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(APIResponse{
		Success:    false,
		StatusCode: statusCode,
		Error:      errMsg,
	})
	if err != nil {
		http.Error(w, `{"success":false,"status_code":500,"error":"Internal Server Error"}`, http.StatusInternalServerError)
	}
}
