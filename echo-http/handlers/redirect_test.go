package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRedirectHandler(t *testing.T) {
	tests := []struct {
		name             string
		n                string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:           "0 redirects returns success",
			n:              "0",
			expectedStatus: http.StatusOK,
		},
		{
			name:             "1 redirect",
			n:                "1",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/redirect/0",
		},
		{
			name:             "5 redirects",
			n:                "5",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/redirect/4",
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
			r.Get("/redirect/{n}", RedirectHandler)

			req := httptest.NewRequest(http.MethodGet, "/redirect/"+tt.n, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedLocation != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected location %q, got %q", tt.expectedLocation, location)
				}
			}
		})
	}
}

func TestRedirectToHandler(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:             "redirect to URL with default status",
			query:            "?url=https://example.com",
			expectedStatus:   http.StatusFound,
			expectedLocation: "https://example.com",
		},
		{
			name:             "redirect with 301",
			query:            "?url=https://example.com&status_code=301",
			expectedStatus:   http.StatusMovedPermanently,
			expectedLocation: "https://example.com",
		},
		{
			name:             "redirect with 307",
			query:            "?url=https://example.com&status_code=307",
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedLocation: "https://example.com",
		},
		{
			name:             "redirect with 308",
			query:            "?url=https://example.com&status_code=308",
			expectedStatus:   http.StatusPermanentRedirect,
			expectedLocation: "https://example.com",
		},
		{
			name:           "missing url returns 400",
			query:          "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid status_code returns 400",
			query:          "?url=https://example.com&status_code=abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "non-redirect status_code returns 400",
			query:          "?url=https://example.com&status_code=200",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/redirect-to", RedirectToHandler)

			req := httptest.NewRequest(http.MethodGet, "/redirect-to"+tt.query, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedLocation != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected location %q, got %q", tt.expectedLocation, location)
				}
			}
		})
	}
}

func TestAbsoluteRedirectHandler(t *testing.T) {
	tests := []struct {
		name             string
		n                string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:           "0 redirects returns success",
			n:              "0",
			expectedStatus: http.StatusOK,
		},
		{
			name:             "1 redirect with absolute URL",
			n:                "1",
			expectedStatus:   http.StatusFound,
			expectedLocation: "http://example.com/absolute-redirect/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/absolute-redirect/{n}", AbsoluteRedirectHandler)

			req := httptest.NewRequest(http.MethodGet, "/absolute-redirect/"+tt.n, nil)
			req.Host = "example.com"
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedLocation != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected location %q, got %q", tt.expectedLocation, location)
				}
			}
		})
	}
}

func TestRelativeRedirectHandler(t *testing.T) {
	tests := []struct {
		name             string
		n                string
		expectedStatus   int
		expectedLocation string
	}{
		{
			name:           "0 redirects returns success",
			n:              "0",
			expectedStatus: http.StatusOK,
		},
		{
			name:             "1 redirect with relative URL",
			n:                "1",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/relative-redirect/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/relative-redirect/{n}", RelativeRedirectHandler)

			req := httptest.NewRequest(http.MethodGet, "/relative-redirect/"+tt.n, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedLocation != "" {
				location := rec.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected location %q, got %q", tt.expectedLocation, location)
				}
			}
		})
	}
}
