package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type CookiesResponse struct {
	Cookies map[string]string `json:"cookies"`
}

// CookiesHandler returns all cookies sent with the request.
// GET /cookies - Echo request cookies
func CookiesHandler(w http.ResponseWriter, r *http.Request) {
	cookies := make(map[string]string)
	for _, cookie := range r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}

	response := CookiesResponse{
		Cookies: cookies,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// CookiesSetHandler sets cookies from query parameters and redirects to /cookies.
// GET /cookies/set?{name}={value}&... - Set cookies and redirect to /cookies
func CookiesSetHandler(w http.ResponseWriter, r *http.Request) {
	for name, values := range r.URL.Query() {
		if len(values) > 0 {
			cookie := &http.Cookie{
				Name:     name,
				Value:    values[0],
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			}
			http.SetCookie(w, cookie)
		}
	}

	http.Redirect(w, r, "/cookies", http.StatusFound)
}

// CookiesDeleteHandler deletes cookies specified in query parameters.
// GET /cookies/delete?{name}&... - Delete specified cookies and redirect to /cookies
func CookiesDeleteHandler(w http.ResponseWriter, r *http.Request) {
	for name := range r.URL.Query() {
		cookie := &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
			HttpOnly: true,
		}
		http.SetCookie(w, cookie)
	}

	http.Redirect(w, r, "/cookies", http.StatusFound)
}
