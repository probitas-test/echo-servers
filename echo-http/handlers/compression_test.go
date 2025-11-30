package handlers

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/go-chi/chi/v5"
)

func TestGzipHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/gzip", GzipHandler)

	req := httptest.NewRequest(http.MethodGet, "/gzip", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentEncoding := rec.Header().Get("Content-Encoding")
	if contentEncoding != "gzip" {
		t.Errorf("expected Content-Encoding gzip, got %s", contentEncoding)
	}

	// Decompress and verify
	gz, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer func() { _ = gz.Close() }()

	decompressed, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	var response CompressionResponse
	if err := json.Unmarshal(decompressed, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Compressed {
		t.Error("expected compressed to be true")
	}
	if response.Method != "gzip" {
		t.Errorf("expected method gzip, got %s", response.Method)
	}
	if response.Origin != "192.168.1.100" {
		t.Errorf("expected origin 192.168.1.100, got %s", response.Origin)
	}
}

func TestDeflateHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/deflate", DeflateHandler)

	req := httptest.NewRequest(http.MethodGet, "/deflate", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentEncoding := rec.Header().Get("Content-Encoding")
	if contentEncoding != "deflate" {
		t.Errorf("expected Content-Encoding deflate, got %s", contentEncoding)
	}

	// Decompress and verify
	fr := flate.NewReader(bytes.NewReader(rec.Body.Bytes()))
	defer func() { _ = fr.Close() }()

	decompressed, err := io.ReadAll(fr)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	var response CompressionResponse
	if err := json.Unmarshal(decompressed, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Compressed {
		t.Error("expected compressed to be true")
	}
	if response.Method != "deflate" {
		t.Errorf("expected method deflate, got %s", response.Method)
	}
}

func TestBrotliHandler(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/brotli", BrotliHandler)

	req := httptest.NewRequest(http.MethodGet, "/brotli", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	contentEncoding := rec.Header().Get("Content-Encoding")
	if contentEncoding != "br" {
		t.Errorf("expected Content-Encoding br, got %s", contentEncoding)
	}

	// Decompress and verify
	br := brotli.NewReader(bytes.NewReader(rec.Body.Bytes()))

	decompressed, err := io.ReadAll(br)
	if err != nil {
		t.Fatalf("failed to decompress: %v", err)
	}

	var response CompressionResponse
	if err := json.Unmarshal(decompressed, &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !response.Compressed {
		t.Error("expected compressed to be true")
	}
	if response.Method != "br" {
		t.Errorf("expected method br, got %s", response.Method)
	}
}
