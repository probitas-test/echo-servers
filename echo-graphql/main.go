package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"

	"github.com/jsr-probitas/echo-servers/echo-graphql/graph"
	"github.com/jsr-probitas/echo-servers/echo-graphql/graph/model"
)

// requestContextMiddleware injects the http.Request into context for header access
func requestContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), model.RequestKey, r)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	cfg := LoadConfig()

	resolver := graph.NewResolver()
	srv := handler.New(graph.NewExecutableSchema(graph.Config{
		Resolvers: resolver,
	}))

	// HTTP transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	// WebSocket transport for subscriptions
	srv.AddTransport(transport.Websocket{
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		KeepAlivePingInterval: 10 * time.Second,
	})

	// Enable introspection
	srv.Use(extension.Introspection{})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// GraphQL playground
	http.Handle("/", playground.Handler("GraphQL Playground", "/graphql"))

	// GraphQL endpoint (with request context middleware for header access)
	http.Handle("/graphql", requestContextMiddleware(srv))

	log.Printf("Starting server on %s", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), nil); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
