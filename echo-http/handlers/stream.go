package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	maxStreamLines  = 100
	maxDripBytes    = 10 * 1024 // 10KB
	maxDripDuration = 60        // 60 seconds
)

type StreamLine struct {
	ID      int               `json:"id"`
	URL     string            `json:"url"`
	Args    map[string]string `json:"args"`
	Headers map[string]string `json:"headers"`
	Origin  string            `json:"origin"`
}

// StreamHandler streams n lines of JSON data.
// GET /stream/{n} - Stream n lines of data (chunked transfer)
func StreamHandler(w http.ResponseWriter, r *http.Request) {
	nStr := chi.URLParam(r, "n")
	n, err := strconv.Atoi(nStr)
	if err != nil || n < 0 || n > maxStreamLines {
		http.Error(w, fmt.Sprintf("Invalid line count (must be 0-%d)", maxStreamLines), http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Transfer-Encoding", "chunked")

	args := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			args[key] = values[0]
		}
	}

	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	origin := getClientIP(r)

	for i := range n {
		line := StreamLine{
			ID:      i,
			URL:     r.URL.RequestURI(),
			Args:    args,
			Headers: headers,
			Origin:  origin,
		}

		data, _ := json.Marshal(line)
		_, _ = w.Write(data)
		_, _ = w.Write([]byte("\n"))
		flusher.Flush()
	}
}

// DripHandler drips data over a specified duration.
// GET /drip?duration={s}&numbytes={n}&delay={s} - Drip data over duration
func DripHandler(w http.ResponseWriter, r *http.Request) {
	duration := 2.0 // default 2 seconds
	if d := r.URL.Query().Get("duration"); d != "" {
		parsed, err := strconv.ParseFloat(d, 64)
		if err != nil || parsed < 0 || parsed > float64(maxDripDuration) {
			http.Error(w, fmt.Sprintf("Invalid duration (must be 0-%d seconds)", maxDripDuration), http.StatusBadRequest)
			return
		}
		duration = parsed
	}

	numBytes := 10 // default 10 bytes
	if n := r.URL.Query().Get("numbytes"); n != "" {
		parsed, err := strconv.Atoi(n)
		if err != nil || parsed < 0 || parsed > maxDripBytes {
			http.Error(w, fmt.Sprintf("Invalid numbytes (must be 0-%d)", maxDripBytes), http.StatusBadRequest)
			return
		}
		numBytes = parsed
	}

	delay := 0.0 // default no initial delay
	if d := r.URL.Query().Get("delay"); d != "" {
		parsed, err := strconv.ParseFloat(d, 64)
		if err != nil || parsed < 0 || parsed > float64(maxDripDuration) {
			http.Error(w, fmt.Sprintf("Invalid delay (must be 0-%d seconds)", maxDripDuration), http.StatusBadRequest)
			return
		}
		delay = parsed
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Initial delay
	if delay > 0 {
		time.Sleep(time.Duration(delay * float64(time.Second)))
	}

	w.Header().Set("Content-Type", "application/octet-stream")

	if numBytes == 0 {
		return
	}

	// Calculate interval between bytes
	interval := time.Duration(duration * float64(time.Second) / float64(numBytes))

	for range numBytes {
		_, _ = w.Write([]byte("*"))
		flusher.Flush()
		if interval > 0 {
			time.Sleep(interval)
		}
	}
}
