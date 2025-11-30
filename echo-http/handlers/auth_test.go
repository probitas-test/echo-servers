package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBasicAuthHandler(t *testing.T) {
	tests := []struct {
		name           string
		user           string
		pass           string
		authUser       string
		authPass       string
		setAuth        bool
		expectedStatus int
		authenticated  bool
	}{
		{
			name:           "correct credentials",
			user:           "testuser",
			pass:           "testpass",
			authUser:       "testuser",
			authPass:       "testpass",
			setAuth:        true,
			expectedStatus: http.StatusOK,
			authenticated:  true,
		},
		{
			name:           "wrong password",
			user:           "testuser",
			pass:           "testpass",
			authUser:       "testuser",
			authPass:       "wrongpass",
			setAuth:        true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "wrong username",
			user:           "testuser",
			pass:           "testpass",
			authUser:       "wronguser",
			authPass:       "testpass",
			setAuth:        true,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "no auth header",
			user:           "testuser",
			pass:           "testpass",
			setAuth:        false,
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/basic-auth/{user}/{pass}", BasicAuthHandler)

			req := httptest.NewRequest(http.MethodGet, "/basic-auth/"+tt.user+"/"+tt.pass, nil)
			if tt.setAuth {
				req.SetBasicAuth(tt.authUser, tt.authPass)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.authenticated {
				var response AuthResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !response.Authenticated {
					t.Error("expected authenticated to be true")
				}
				if response.User != tt.authUser {
					t.Errorf("expected user %q, got %q", tt.authUser, response.User)
				}
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				wwwAuth := rec.Header().Get("WWW-Authenticate")
				if wwwAuth == "" {
					t.Error("expected WWW-Authenticate header")
				}
			}
		})
	}
}

func TestHiddenBasicAuthHandler(t *testing.T) {
	tests := []struct {
		name           string
		user           string
		pass           string
		authUser       string
		authPass       string
		setAuth        bool
		expectedStatus int
	}{
		{
			name:           "correct credentials",
			user:           "testuser",
			pass:           "testpass",
			authUser:       "testuser",
			authPass:       "testpass",
			setAuth:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "wrong credentials returns 404",
			user:           "testuser",
			pass:           "testpass",
			authUser:       "testuser",
			authPass:       "wrongpass",
			setAuth:        true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "no auth returns 404",
			user:           "testuser",
			pass:           "testpass",
			setAuth:        false,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/hidden-basic-auth/{user}/{pass}", HiddenBasicAuthHandler)

			req := httptest.NewRequest(http.MethodGet, "/hidden-basic-auth/"+tt.user+"/"+tt.pass, nil)
			if tt.setAuth {
				req.SetBasicAuth(tt.authUser, tt.authPass)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestBearerHandler(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		authenticated  bool
		expectedToken  string
	}{
		{
			name:           "valid bearer token",
			authHeader:     "Bearer my-secret-token",
			expectedStatus: http.StatusOK,
			authenticated:  true,
			expectedToken:  "my-secret-token",
		},
		{
			name:           "case insensitive bearer",
			authHeader:     "bearer my-token",
			expectedStatus: http.StatusOK,
			authenticated:  true,
			expectedToken:  "my-token",
		},
		{
			name:           "no auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "wrong auth type",
			authHeader:     "Basic dXNlcjpwYXNz",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "malformed header",
			authHeader:     "Bearer",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/bearer", BearerHandler)

			req := httptest.NewRequest(http.MethodGet, "/bearer", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.authenticated {
				var response AuthResponse
				if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !response.Authenticated {
					t.Error("expected authenticated to be true")
				}
				if response.Token != tt.expectedToken {
					t.Errorf("expected token %q, got %q", tt.expectedToken, response.Token)
				}
			}

			if tt.expectedStatus == http.StatusUnauthorized {
				wwwAuth := rec.Header().Get("WWW-Authenticate")
				if wwwAuth == "" {
					t.Error("expected WWW-Authenticate header")
				}
			}
		})
	}
}
