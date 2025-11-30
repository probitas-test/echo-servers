package handlers

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestStreamHandler(t *testing.T) {
	tests := []struct {
		name           string
		n              string
		expectedStatus int
		expectedLines  int
	}{
		{
			name:           "0 lines",
			n:              "0",
			expectedStatus: http.StatusOK,
			expectedLines:  0,
		},
		{
			name:           "5 lines",
			n:              "5",
			expectedStatus: http.StatusOK,
			expectedLines:  5,
		},
		{
			name:           "negative number returns 400",
			n:              "-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "over max returns 400",
			n:              "101",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-numeric returns 400",
			n:              "abc",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/stream/{n}", StreamHandler)

			req := httptest.NewRequest(http.MethodGet, "/stream/"+tt.n, nil)
			req.RemoteAddr = "192.168.1.100:12345"
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.expectedLines > 0 {
				scanner := bufio.NewScanner(rec.Body)
				lineCount := 0
				for scanner.Scan() {
					line := scanner.Text()
					if line == "" {
						continue
					}

					var streamLine StreamLine
					if err := json.Unmarshal([]byte(line), &streamLine); err != nil {
						t.Errorf("failed to parse line %d: %v", lineCount, err)
						continue
					}

					if streamLine.ID != lineCount {
						t.Errorf("expected ID %d, got %d", lineCount, streamLine.ID)
					}

					lineCount++
				}

				if lineCount != tt.expectedLines {
					t.Errorf("expected %d lines, got %d", tt.expectedLines, lineCount)
				}
			}
		})
	}
}

func TestStreamHandlerContent(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/stream/{n}", StreamHandler)

	req := httptest.NewRequest(http.MethodGet, "/stream/1?foo=bar", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var streamLine StreamLine
	if err := json.Unmarshal([]byte(strings.TrimSpace(rec.Body.String())), &streamLine); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if streamLine.Args["foo"] != "bar" {
		t.Errorf("expected foo=bar, got %s", streamLine.Args["foo"])
	}

	if streamLine.Headers["X-Custom-Header"] != "test-value" {
		t.Errorf("expected X-Custom-Header=test-value, got %s", streamLine.Headers["X-Custom-Header"])
	}

	if streamLine.Origin != "192.168.1.100" {
		t.Errorf("expected origin 192.168.1.100, got %s", streamLine.Origin)
	}
}

func TestDripHandler(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		expectedStatus int
		expectedBytes  int
	}{
		{
			name:           "default values",
			query:          "",
			expectedStatus: http.StatusOK,
			expectedBytes:  10,
		},
		{
			name:           "custom numbytes",
			query:          "?numbytes=5",
			expectedStatus: http.StatusOK,
			expectedBytes:  5,
		},
		{
			name:           "zero bytes",
			query:          "?numbytes=0",
			expectedStatus: http.StatusOK,
			expectedBytes:  0,
		},
		{
			name:           "zero duration",
			query:          "?duration=0&numbytes=5",
			expectedStatus: http.StatusOK,
			expectedBytes:  5,
		},
		{
			name:           "invalid duration",
			query:          "?duration=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid numbytes",
			query:          "?numbytes=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative duration",
			query:          "?duration=-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "over max numbytes",
			query:          "?numbytes=10241",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "over max duration",
			query:          "?duration=61",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/drip", DripHandler)

			req := httptest.NewRequest(http.MethodGet, "/drip"+tt.query, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				if len(rec.Body.Bytes()) != tt.expectedBytes {
					t.Errorf("expected %d bytes, got %d", tt.expectedBytes, len(rec.Body.Bytes()))
				}
			}
		})
	}
}
