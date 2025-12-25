package handlers

import (
	"net/http"
)

var apiDocs string

func SetAPIDocs(content string) {
	apiDocs = content
}

func APIDocsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	_, _ = w.Write([]byte(apiDocs))
}
