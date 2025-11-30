package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestCookiesHandler(t *testing.T) {
	tests := []struct {
		name            string
		cookies         []*http.Cookie
		expectedCookies map[string]string
	}{
		{
			name:            "no cookies",
			cookies:         nil,
			expectedCookies: map[string]string{},
		},
		{
			name: "single cookie",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
			},
			expectedCookies: map[string]string{"session": "abc123"},
		},
		{
			name: "multiple cookies",
			cookies: []*http.Cookie{
				{Name: "session", Value: "abc123"},
				{Name: "user", Value: "testuser"},
			},
			expectedCookies: map[string]string{"session": "abc123", "user": "testuser"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/cookies", CookiesHandler)

			req := httptest.NewRequest(http.MethodGet, "/cookies", nil)
			for _, cookie := range tt.cookies {
				req.AddCookie(cookie)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var response CookiesResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(response.Cookies) != len(tt.expectedCookies) {
				t.Errorf("expected %d cookies, got %d", len(tt.expectedCookies), len(response.Cookies))
			}

			for name, value := range tt.expectedCookies {
				if response.Cookies[name] != value {
					t.Errorf("expected cookie %s=%s, got %s", name, value, response.Cookies[name])
				}
			}
		})
	}
}

func TestCookiesSetHandler(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		expectedCookies []string
	}{
		{
			name:            "set single cookie",
			query:           "?session=abc123",
			expectedCookies: []string{"session"},
		},
		{
			name:            "set multiple cookies",
			query:           "?session=abc123&user=testuser",
			expectedCookies: []string{"session", "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/cookies/set", CookiesSetHandler)

			req := httptest.NewRequest(http.MethodGet, "/cookies/set"+tt.query, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
			}

			location := rec.Header().Get("Location")
			if location != "/cookies" {
				t.Errorf("expected redirect to /cookies, got %s", location)
			}

			cookies := rec.Result().Cookies()
			cookieNames := make(map[string]bool)
			for _, cookie := range cookies {
				cookieNames[cookie.Name] = true
			}

			for _, expected := range tt.expectedCookies {
				if !cookieNames[expected] {
					t.Errorf("expected cookie %s to be set", expected)
				}
			}
		})
	}
}

func TestCookiesDeleteHandler(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		expectedDeleted []string
	}{
		{
			name:            "delete single cookie",
			query:           "?session",
			expectedDeleted: []string{"session"},
		},
		{
			name:            "delete multiple cookies",
			query:           "?session&user",
			expectedDeleted: []string{"session", "user"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/cookies/delete", CookiesDeleteHandler)

			req := httptest.NewRequest(http.MethodGet, "/cookies/delete"+tt.query, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusFound {
				t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
			}

			location := rec.Header().Get("Location")
			if location != "/cookies" {
				t.Errorf("expected redirect to /cookies, got %s", location)
			}

			cookies := rec.Result().Cookies()
			for _, cookie := range cookies {
				found := false
				for _, expected := range tt.expectedDeleted {
					if cookie.Name == expected {
						found = true
						if cookie.MaxAge != -1 {
							t.Errorf("expected cookie %s to have MaxAge -1, got %d", cookie.Name, cookie.MaxAge)
						}
						break
					}
				}
				if !found {
					t.Errorf("unexpected cookie in response: %s", cookie.Name)
				}
			}
		})
	}
}
