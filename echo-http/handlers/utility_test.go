package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestIPHandler(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedOrigin string
	}{
		{
			name:           "from RemoteAddr",
			remoteAddr:     "192.168.1.100:12345",
			expectedOrigin: "192.168.1.100",
		},
		{
			name:           "from X-Forwarded-For",
			remoteAddr:     "10.0.0.1:12345",
			xForwardedFor:  "203.0.113.50",
			expectedOrigin: "203.0.113.50",
		},
		{
			name:           "from X-Forwarded-For with multiple IPs",
			remoteAddr:     "10.0.0.1:12345",
			xForwardedFor:  "203.0.113.50, 70.41.3.18, 150.172.238.178",
			expectedOrigin: "203.0.113.50",
		},
		{
			name:           "from X-Real-IP",
			remoteAddr:     "10.0.0.1:12345",
			xRealIP:        "203.0.113.100",
			expectedOrigin: "203.0.113.100",
		},
		{
			name:           "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr:     "10.0.0.1:12345",
			xForwardedFor:  "203.0.113.50",
			xRealIP:        "203.0.113.100",
			expectedOrigin: "203.0.113.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/ip", IPHandler)

			req := httptest.NewRequest(http.MethodGet, "/ip", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var response IPResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Origin != tt.expectedOrigin {
				t.Errorf("expected origin %q, got %q", tt.expectedOrigin, response.Origin)
			}
		})
	}
}

func TestUserAgentHandler(t *testing.T) {
	tests := []struct {
		name              string
		userAgent         string
		expectedUserAgent string
	}{
		{
			name:              "standard user agent",
			userAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedUserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		},
		{
			name:              "curl user agent",
			userAgent:         "curl/7.79.1",
			expectedUserAgent: "curl/7.79.1",
		},
		{
			name:              "empty user agent",
			userAgent:         "",
			expectedUserAgent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/user-agent", UserAgentHandler)

			req := httptest.NewRequest(http.MethodGet, "/user-agent", nil)
			if tt.userAgent != "" {
				req.Header.Set("User-Agent", tt.userAgent)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var response UserAgentResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.UserAgent != tt.expectedUserAgent {
				t.Errorf("expected user-agent %q, got %q", tt.expectedUserAgent, response.UserAgent)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "with port",
			remoteAddr: "192.168.1.1:8080",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "without port",
			remoteAddr: "192.168.1.1",
			expectedIP: "192.168.1.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[::1]:8080",
			expectedIP: "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("expected IP %q, got %q", tt.expectedIP, ip)
			}
		})
	}
}
