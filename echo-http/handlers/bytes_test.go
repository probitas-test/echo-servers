package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBytesHandler(t *testing.T) {
	tests := []struct {
		name           string
		n              string
		expectedStatus int
		expectedLength int
	}{
		{
			name:           "0 bytes",
			n:              "0",
			expectedStatus: http.StatusOK,
			expectedLength: 0,
		},
		{
			name:           "10 bytes",
			n:              "10",
			expectedStatus: http.StatusOK,
			expectedLength: 10,
		},
		{
			name:           "1024 bytes",
			n:              "1024",
			expectedStatus: http.StatusOK,
			expectedLength: 1024,
		},
		{
			name:           "negative number returns 400",
			n:              "-1",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "over max returns 400",
			n:              "102401",
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
			r.Get("/bytes/{n}", BytesHandler)

			req := httptest.NewRequest(http.MethodGet, "/bytes/"+tt.n, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				if len(rec.Body.Bytes()) != tt.expectedLength {
					t.Errorf("expected %d bytes, got %d", tt.expectedLength, len(rec.Body.Bytes()))
				}

				contentType := rec.Header().Get("Content-Type")
				if contentType != "application/octet-stream" {
					t.Errorf("expected Content-Type application/octet-stream, got %s", contentType)
				}
			}
		})
	}
}

func TestBytesHandlerRandomness(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/bytes/{n}", BytesHandler)

	// Generate two responses and ensure they're different (with high probability)
	req1 := httptest.NewRequest(http.MethodGet, "/bytes/100", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodGet, "/bytes/100", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	// Compare the two responses
	if rec1.Body.String() == rec2.Body.String() {
		t.Error("expected different random bytes, got identical responses")
	}
}
