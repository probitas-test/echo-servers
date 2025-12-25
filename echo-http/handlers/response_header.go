package handlers

import (
	"encoding/json"
	"net/http"
)

type ResponseHeaderResponse struct {
	Headers map[string]string `json:"headers"`
}

// ResponseHeaderHandler sets response headers based on query parameters.
// Each query parameter key-value pair is set as a response header.
// Example: GET /response-header?X-Custom-Header=value&Content-Language=en
func ResponseHeaderHandler(w http.ResponseWriter, r *http.Request) {
	response := ResponseHeaderResponse{
		Headers: make(map[string]string),
	}

	// Set each query parameter as a response header
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			w.Header().Set(key, values[0])
			response.Headers[key] = values[0]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
