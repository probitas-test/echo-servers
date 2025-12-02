package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jsr-probitas/echo-servers/echo-http/handlers"
)

func main() {
	cfg := LoadConfig()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Echo endpoints
	r.Get("/get", handlers.EchoHandler)
	r.Post("/post", handlers.EchoHandler)
	r.Put("/put", handlers.EchoHandler)
	r.Patch("/patch", handlers.EchoHandler)
	r.Delete("/delete", handlers.EchoHandler)

	// Anything endpoint - echoes any request
	r.HandleFunc("/anything", handlers.AnythingHandler)
	r.HandleFunc("/anything/*", handlers.AnythingHandler)

	// Utility endpoints
	r.Get("/headers", handlers.HeadersHandler)
	r.Get("/ip", handlers.IPHandler)
	r.Get("/user-agent", handlers.UserAgentHandler)

	// Status endpoint - support all HTTP methods
	r.HandleFunc("/status/{code}", handlers.StatusHandler)

	// Delay endpoint
	r.Get("/delay/{seconds}", handlers.DelayHandler)

	// Redirect endpoints
	r.Get("/redirect/{n}", handlers.RedirectHandler)
	r.Get("/redirect-to", handlers.RedirectToHandler)
	r.Get("/absolute-redirect/{n}", handlers.AbsoluteRedirectHandler)
	r.Get("/relative-redirect/{n}", handlers.RelativeRedirectHandler)

	// Authentication endpoints
	r.Get("/basic-auth/{user}/{pass}", handlers.BasicAuthHandler)
	r.Get("/hidden-basic-auth/{user}/{pass}", handlers.HiddenBasicAuthHandler)
	r.Get("/bearer", handlers.BearerHandler)

	// Cookie endpoints
	r.Get("/cookies", handlers.CookiesHandler)
	r.Get("/cookies/set", handlers.CookiesSetHandler)
	r.Get("/cookies/delete", handlers.CookiesDeleteHandler)

	// Binary data endpoints
	r.Get("/bytes/{n}", handlers.BytesHandler)

	// Streaming endpoints
	r.Get("/stream/{n}", handlers.StreamHandler)
	r.Get("/drip", handlers.DripHandler)

	// Compression endpoints
	r.Get("/gzip", handlers.GzipHandler)
	r.Get("/deflate", handlers.DeflateHandler)
	r.Get("/brotli", handlers.BrotliHandler)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	log.Printf("Starting server on %s", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), r); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
