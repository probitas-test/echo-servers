package handlers

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
)

type IPResponse struct {
	Origin string `json:"origin"`
}

type UserAgentResponse struct {
	UserAgent string `json:"user-agent"`
}

// IPHandler returns the client's IP address.
// GET /ip - Return client IP address
func IPHandler(w http.ResponseWriter, r *http.Request) {
	response := IPResponse{
		Origin: getClientIP(r),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// UserAgentHandler returns the User-Agent header.
// GET /user-agent - Return User-Agent header
func UserAgentHandler(w http.ResponseWriter, r *http.Request) {
	response := UserAgentResponse{
		UserAgent: r.Header.Get("User-Agent"),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// getClientIP extracts the client IP address from the request.
// It checks X-Forwarded-For, X-Real-IP headers before falling back to RemoteAddr.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
