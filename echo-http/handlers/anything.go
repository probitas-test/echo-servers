package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type AnythingResponse struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Args    map[string]string `json:"args"`
	Headers map[string]string `json:"headers"`
	Origin  string            `json:"origin"`
	Data    string            `json:"data,omitempty"`
	JSON    any               `json:"json,omitempty"`
	Form    map[string]string `json:"form,omitempty"`
	Files   map[string]string `json:"files,omitempty"`
}

// AnythingHandler echoes any request information.
// ANY /anything - Echo any request (method, headers, body, etc.)
// ANY /anything/{path} - Echo any request with path
func AnythingHandler(w http.ResponseWriter, r *http.Request) {
	response := AnythingResponse{
		Method:  r.Method,
		URL:     r.URL.RequestURI(),
		Args:    make(map[string]string),
		Headers: make(map[string]string),
		Origin:  getClientIP(r),
	}

	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			response.Args[key] = values[0]
		}
	}

	for key, values := range r.Header {
		if len(values) > 0 {
			response.Headers[key] = values[0]
		}
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodDelete {
		body, err := io.ReadAll(r.Body)
		if err == nil && len(body) > 0 {
			response.Data = string(body)

			contentType := r.Header.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				var jsonData any
				if err := json.Unmarshal(body, &jsonData); err == nil {
					response.JSON = jsonData
				}
			} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
				r.Body = io.NopCloser(strings.NewReader(string(body)))
				if err := r.ParseForm(); err == nil {
					formData := make(map[string]string)
					for key, values := range r.PostForm {
						if len(values) > 0 {
							formData[key] = values[0]
						}
					}
					response.Form = formData
				}
			} else if strings.Contains(contentType, "multipart/form-data") {
				r.Body = io.NopCloser(strings.NewReader(string(body)))
				if err := r.ParseMultipartForm(32 << 20); err == nil {
					if r.MultipartForm != nil {
						formData := make(map[string]string)
						for key, values := range r.MultipartForm.Value {
							if len(values) > 0 {
								formData[key] = values[0]
							}
						}
						response.Form = formData

						files := make(map[string]string)
						for key, fileHeaders := range r.MultipartForm.File {
							if len(fileHeaders) > 0 {
								files[key] = fileHeaders[0].Filename
							}
						}
						response.Files = files
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
