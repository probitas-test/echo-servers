package handlers

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const (
	maxBytesSize = 100 * 1024 // 100KB
)

// BytesHandler returns n random bytes.
// GET /bytes/{n} - Return n random bytes
func BytesHandler(w http.ResponseWriter, r *http.Request) {
	nStr := chi.URLParam(r, "n")
	n, err := strconv.Atoi(nStr)
	if err != nil || n < 0 || n > maxBytesSize {
		http.Error(w, fmt.Sprintf("Invalid byte count (must be 0-%d)", maxBytesSize), http.StatusBadRequest)
		return
	}

	if n == 0 {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "0")
		return
	}

	data := make([]byte, n)
	if _, err := rand.Read(data); err != nil {
		http.Error(w, "Failed to generate random bytes", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(n))
	_, _ = w.Write(data)
}
