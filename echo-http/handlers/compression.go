package handlers

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"net/http"

	"github.com/andybalholm/brotli"
)

type CompressionResponse struct {
	Compressed bool              `json:"compressed"`
	Method     string            `json:"method"`
	Origin     string            `json:"origin"`
	Headers    map[string]string `json:"headers"`
}

// GzipHandler returns a gzip-compressed response.
// GET /gzip - Return gzip-compressed response
func GzipHandler(w http.ResponseWriter, r *http.Request) {
	response := CompressionResponse{
		Compressed: true,
		Method:     "gzip",
		Origin:     getClientIP(r),
		Headers:    make(map[string]string),
	}

	for key, values := range r.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(jsonData); err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}
	if err := gz.Close(); err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	_, _ = w.Write(buf.Bytes())
}

// DeflateHandler returns a deflate-compressed response.
// GET /deflate - Return deflate-compressed response
func DeflateHandler(w http.ResponseWriter, r *http.Request) {
	response := CompressionResponse{
		Compressed: true,
		Method:     "deflate",
		Origin:     getClientIP(r),
		Headers:    make(map[string]string),
	}

	for key, values := range r.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	fw, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}
	if _, err := fw.Write(jsonData); err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}
	if err := fw.Close(); err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "deflate")
	_, _ = w.Write(buf.Bytes())
}

// BrotliHandler returns a brotli-compressed response.
// GET /brotli - Return brotli-compressed response
func BrotliHandler(w http.ResponseWriter, r *http.Request) {
	response := CompressionResponse{
		Compressed: true,
		Method:     "br",
		Origin:     getClientIP(r),
		Headers:    make(map[string]string),
	}

	for key, values := range r.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	bw := brotli.NewWriter(&buf)
	if _, err := bw.Write(jsonData); err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}
	if err := bw.Close(); err != nil {
		http.Error(w, "Failed to compress response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "br")
	_, _ = w.Write(buf.Bytes())
}
