package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseHeaderHandler(t *testing.T) {
	t.Run("sets response headers from query parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/response-header?X-Custom-Header=custom-value&X-Request-Id=12345", nil)
		rec := httptest.NewRecorder()

		ResponseHeaderHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		// Check response headers
		if rec.Header().Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected X-Custom-Header=custom-value, got %s", rec.Header().Get("X-Custom-Header"))
		}

		if rec.Header().Get("X-Request-Id") != "12345" {
			t.Errorf("expected X-Request-Id=12345, got %s", rec.Header().Get("X-Request-Id"))
		}

		// Check response body
		var resp ResponseHeaderResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Headers["X-Custom-Header"] != "custom-value" {
			t.Errorf("expected body X-Custom-Header=custom-value, got %s", resp.Headers["X-Custom-Header"])
		}

		if resp.Headers["X-Request-Id"] != "12345" {
			t.Errorf("expected body X-Request-Id=12345, got %s", resp.Headers["X-Request-Id"])
		}
	})

	t.Run("handles standard HTTP headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/response-header?Cache-Control=no-cache&Content-Language=en-US", nil)
		rec := httptest.NewRecorder()

		ResponseHeaderHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("expected Cache-Control=no-cache, got %s", rec.Header().Get("Cache-Control"))
		}

		if rec.Header().Get("Content-Language") != "en-US" {
			t.Errorf("expected Content-Language=en-US, got %s", rec.Header().Get("Content-Language"))
		}
	})

	t.Run("handles empty query parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/response-header", nil)
		rec := httptest.NewRecorder()

		ResponseHeaderHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp ResponseHeaderResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Headers) != 0 {
			t.Errorf("expected empty headers map, got %d headers", len(resp.Headers))
		}
	})

	t.Run("uses first value when multiple values exist", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/response-header?X-Multi=first&X-Multi=second", nil)
		rec := httptest.NewRecorder()

		ResponseHeaderHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		headerValue := rec.Header().Get("X-Multi")
		if headerValue != "first" {
			t.Errorf("expected X-Multi=first, got %s", headerValue)
		}
	})
}
