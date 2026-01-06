package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestWriteOIDCError(t *testing.T) {
	tests := []struct {
		name               string
		statusCode         int
		errorCode          string
		description        string
		expectedStatusCode int
		expectedError      string
		expectedDesc       string
	}{
		{
			name:               "invalid_request error",
			statusCode:         http.StatusBadRequest,
			errorCode:          "invalid_request",
			description:        "client_id parameter is required",
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid_request",
			expectedDesc:       "client_id parameter is required",
		},
		{
			name:               "invalid_client error",
			statusCode:         http.StatusUnauthorized,
			errorCode:          "invalid_client",
			description:        "unknown client_id",
			expectedStatusCode: http.StatusUnauthorized,
			expectedError:      "invalid_client",
			expectedDesc:       "unknown client_id",
		},
		{
			name:               "error without description",
			statusCode:         http.StatusBadRequest,
			errorCode:          "invalid_scope",
			description:        "",
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid_scope",
			expectedDesc:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			writeOIDCError(rec, tt.statusCode, tt.errorCode, tt.description)

			// Verify status code
			if rec.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, rec.Code)
			}

			// Verify Content-Type
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// Verify response body
			var errResp struct {
				Error            string `json:"error"`
				ErrorDescription string `json:"error_description,omitempty"`
			}

			if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
				t.Fatalf("failed to parse response body: %v", err)
			}

			if errResp.Error != tt.expectedError {
				t.Errorf("expected error %q, got %q", tt.expectedError, errResp.Error)
			}

			if errResp.ErrorDescription != tt.expectedDesc {
				t.Errorf("expected error_description %q, got %q", tt.expectedDesc, errResp.ErrorDescription)
			}
		})
	}
}

func TestWriteAuthorizationError(t *testing.T) {
	tests := []struct {
		name               string
		errorCode          string
		description        string
		state              string
		redirectURI        string
		expectedStatusCode int
		expectRedirect     bool
		expectJSONError    bool
	}{
		{
			name:               "redirect with error and state",
			errorCode:          "unauthorized_client",
			description:        "unknown client_id",
			state:              "xyz123",
			redirectURI:        "http://localhost/callback",
			expectedStatusCode: http.StatusFound,
			expectRedirect:     true,
			expectJSONError:    false,
		},
		{
			name:               "redirect with error without state",
			errorCode:          "invalid_scope",
			description:        "unsupported scope: admin",
			state:              "",
			redirectURI:        "http://localhost/callback",
			expectedStatusCode: http.StatusFound,
			expectRedirect:     true,
			expectJSONError:    false,
		},
		{
			name:               "no redirect_uri returns JSON error",
			errorCode:          "invalid_request",
			description:        "redirect_uri parameter is required",
			state:              "",
			redirectURI:        "",
			expectedStatusCode: http.StatusBadRequest,
			expectRedirect:     false,
			expectJSONError:    true,
		},
		{
			name:               "invalid redirect_uri returns JSON error",
			errorCode:          "invalid_request",
			description:        "some error",
			state:              "",
			redirectURI:        "://invalid-url",
			expectedStatusCode: http.StatusBadRequest,
			expectRedirect:     false,
			expectJSONError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/authorize", nil)
			rec := httptest.NewRecorder()

			writeAuthorizationError(rec, req, tt.errorCode, tt.description, tt.state, tt.redirectURI)

			// Verify status code
			if rec.Code != tt.expectedStatusCode {
				t.Errorf("expected status code %d, got %d", tt.expectedStatusCode, rec.Code)
			}

			if tt.expectRedirect {
				// Verify redirect
				location := rec.Header().Get("Location")
				if location == "" {
					t.Fatal("expected Location header, got none")
				}

				redirectURL, err := url.Parse(location)
				if err != nil {
					t.Fatalf("failed to parse redirect URL: %v", err)
				}

				query := redirectURL.Query()

				// Verify error parameter
				if query.Get("error") != tt.errorCode {
					t.Errorf("expected error=%q, got %q", tt.errorCode, query.Get("error"))
				}

				// Verify error_description parameter
				if tt.description != "" && query.Get("error_description") != tt.description {
					t.Errorf("expected error_description=%q, got %q", tt.description, query.Get("error_description"))
				}

				// Verify state parameter
				if tt.state != "" && query.Get("state") != tt.state {
					t.Errorf("expected state=%q, got %q", tt.state, query.Get("state"))
				}
			}

			if tt.expectJSONError {
				// Verify JSON error response
				contentType := rec.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", contentType)
				}

				var errResp struct {
					Error            string `json:"error"`
					ErrorDescription string `json:"error_description,omitempty"`
				}

				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to parse response body: %v", err)
				}

				// For invalid redirect_uri case, error code should be "invalid_request"
				if tt.redirectURI == "://invalid-url" {
					if errResp.Error != "invalid_request" {
						t.Errorf("expected error=invalid_request for invalid redirect_uri, got %q", errResp.Error)
					}
				} else {
					if errResp.Error != tt.errorCode {
						t.Errorf("expected error %q, got %q", tt.errorCode, errResp.Error)
					}
				}
			}
		})
	}
}
