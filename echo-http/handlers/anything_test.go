package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestAnythingHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		expectedMethod string
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			path:           "/anything?foo=bar",
			expectedMethod: "GET",
		},
		{
			name:           "POST request with JSON",
			method:         http.MethodPost,
			path:           "/anything",
			body:           `{"key":"value"}`,
			contentType:    "application/json",
			expectedMethod: "POST",
		},
		{
			name:           "PUT request",
			method:         http.MethodPut,
			path:           "/anything",
			body:           `{"key":"value"}`,
			contentType:    "application/json",
			expectedMethod: "PUT",
		},
		{
			name:           "PATCH request",
			method:         http.MethodPatch,
			path:           "/anything",
			body:           `{"key":"value"}`,
			contentType:    "application/json",
			expectedMethod: "PATCH",
		},
		{
			name:           "DELETE request",
			method:         http.MethodDelete,
			path:           "/anything",
			expectedMethod: "DELETE",
		},
		{
			name:           "POST with form data",
			method:         http.MethodPost,
			path:           "/anything",
			body:           "username=test&password=secret",
			contentType:    "application/x-www-form-urlencoded",
			expectedMethod: "POST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.HandleFunc("/anything", AnythingHandler)
			r.HandleFunc("/anything/*", AnythingHandler)

			var body *strings.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			} else {
				body = strings.NewReader("")
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			req.RemoteAddr = "192.168.1.100:12345"
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var response AnythingResponse
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response.Method != tt.expectedMethod {
				t.Errorf("expected method %s, got %s", tt.expectedMethod, response.Method)
			}

			if response.Origin != "192.168.1.100" {
				t.Errorf("expected origin 192.168.1.100, got %s", response.Origin)
			}
		})
	}
}

func TestAnythingHandlerQueryParams(t *testing.T) {
	r := chi.NewRouter()
	r.HandleFunc("/anything", AnythingHandler)

	req := httptest.NewRequest(http.MethodGet, "/anything?foo=bar&baz=qux", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var response AnythingResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Args["foo"] != "bar" {
		t.Errorf("expected foo=bar, got foo=%s", response.Args["foo"])
	}
	if response.Args["baz"] != "qux" {
		t.Errorf("expected baz=qux, got baz=%s", response.Args["baz"])
	}
}

func TestAnythingHandlerHeaders(t *testing.T) {
	r := chi.NewRouter()
	r.HandleFunc("/anything", AnythingHandler)

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	req.Header.Set("X-Custom-Header", "custom-value")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var response AnythingResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("expected X-Custom-Header=custom-value, got %s", response.Headers["X-Custom-Header"])
	}
}

func TestAnythingHandlerJSONBody(t *testing.T) {
	r := chi.NewRouter()
	r.HandleFunc("/anything", AnythingHandler)

	req := httptest.NewRequest(http.MethodPost, "/anything", strings.NewReader(`{"name":"test","count":42}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var response AnythingResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Data != `{"name":"test","count":42}` {
		t.Errorf("unexpected data: %s", response.Data)
	}

	if response.JSON == nil {
		t.Error("expected JSON field to be populated")
	}
}

func TestAnythingHandlerFormBody(t *testing.T) {
	r := chi.NewRouter()
	r.HandleFunc("/anything", AnythingHandler)

	req := httptest.NewRequest(http.MethodPost, "/anything", strings.NewReader("username=test&password=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	var response AnythingResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Form["username"] != "test" {
		t.Errorf("expected username=test, got %s", response.Form["username"])
	}
	if response.Form["password"] != "secret" {
		t.Errorf("expected password=secret, got %s", response.Form["password"])
	}
}
