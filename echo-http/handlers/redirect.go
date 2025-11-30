package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

const (
	maxRedirectCount = 100
)

// RedirectHandler redirects n times before returning a final response.
// GET /redirect/{n} - Redirect n times, then return 200 OK
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	nStr := chi.URLParam(r, "n")
	n, err := strconv.Atoi(nStr)
	if err != nil || n < 0 || n > maxRedirectCount {
		http.Error(w, fmt.Sprintf("Invalid redirect count (must be 0-%d)", maxRedirectCount), http.StatusBadRequest)
		return
	}

	if n == 0 {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"redirected":true}`))
		return
	}

	location := fmt.Sprintf("/redirect/%d", n-1)
	http.Redirect(w, r, location, http.StatusFound)
}

// RedirectToHandler redirects to a specified URL.
// GET /redirect-to?url={url}&status_code={code}
func RedirectToHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := r.URL.Query().Get("url")
	if targetURL == "" {
		http.Error(w, "Missing required parameter: url", http.StatusBadRequest)
		return
	}

	statusCode := http.StatusFound // default 302
	if codeStr := r.URL.Query().Get("status_code"); codeStr != "" {
		code, err := strconv.Atoi(codeStr)
		if err != nil {
			http.Error(w, "Invalid status_code", http.StatusBadRequest)
			return
		}
		// Only allow redirect status codes
		if code != http.StatusMovedPermanently && // 301
			code != http.StatusFound && // 302
			code != http.StatusSeeOther && // 303
			code != http.StatusTemporaryRedirect && // 307
			code != http.StatusPermanentRedirect { // 308
			http.Error(w, "status_code must be a redirect code (301, 302, 303, 307, 308)", http.StatusBadRequest)
			return
		}
		statusCode = code
	}

	http.Redirect(w, r, targetURL, statusCode)
}

// AbsoluteRedirectHandler redirects n times using absolute URLs.
// GET /absolute-redirect/{n} - Redirect n times with absolute URLs
func AbsoluteRedirectHandler(w http.ResponseWriter, r *http.Request) {
	nStr := chi.URLParam(r, "n")
	n, err := strconv.Atoi(nStr)
	if err != nil || n < 0 || n > maxRedirectCount {
		http.Error(w, fmt.Sprintf("Invalid redirect count (must be 0-%d)", maxRedirectCount), http.StatusBadRequest)
		return
	}

	if n == 0 {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"redirected":true}`))
		return
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwdProto := r.Header.Get("X-Forwarded-Proto"); fwdProto != "" {
		scheme = fwdProto
	}

	host := r.Host
	location := fmt.Sprintf("%s://%s/absolute-redirect/%d", scheme, host, n-1)
	http.Redirect(w, r, location, http.StatusFound)
}

// RelativeRedirectHandler redirects n times using relative URLs.
// GET /relative-redirect/{n} - Redirect n times with relative URLs
func RelativeRedirectHandler(w http.ResponseWriter, r *http.Request) {
	nStr := chi.URLParam(r, "n")
	n, err := strconv.Atoi(nStr)
	if err != nil || n < 0 || n > maxRedirectCount {
		http.Error(w, fmt.Sprintf("Invalid redirect count (must be 0-%d)", maxRedirectCount), http.StatusBadRequest)
		return
	}

	if n == 0 {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"redirected":true}`))
		return
	}

	location := fmt.Sprintf("/relative-redirect/%d", n-1)
	http.Redirect(w, r, location, http.StatusFound)
}
