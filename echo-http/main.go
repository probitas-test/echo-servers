package main

import (
	_ "embed"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/probitas-test/echo-servers/echo-http/handlers"
)

//go:embed docs/api.md
var apiDocs string

func main() {
	cfg := LoadConfig()

	// Set API docs content for handler
	handlers.SetAPIDocs(apiDocs)

	// Set OAuth2/OIDC config for handlers
	handlers.SetConfig(&handlers.Config{
		AuthAllowedClientID:         cfg.AuthAllowedClientID,
		AuthAllowedClientSecret:     cfg.AuthAllowedClientSecret,
		AuthSupportedScopes:         cfg.AuthSupportedScopes,
		AuthTokenExpiry:             cfg.AuthTokenExpiry,
		AuthAllowedGrantTypes:       cfg.AuthAllowedGrantTypes,
		AuthAllowedUsername:         cfg.AuthAllowedUsername,
		AuthAllowedPassword:         cfg.AuthAllowedPassword,
		AuthCodeRequirePKCE:         cfg.AuthCodeRequirePKCE,
		AuthCodeSessionTTL:          cfg.AuthCodeSessionTTL,
		AuthCodeValidateRedirectURI: cfg.AuthCodeValidateRedirectURI,
		AuthCodeAllowedRedirectURIs: cfg.AuthCodeAllowedRedirectURIs,
		OIDCEnableJWTSigning:        cfg.OIDCEnableJWTSigning,
	})

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
	r.Get("/response-header", handlers.ResponseHeaderHandler)
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

	// OAuth2/OIDC endpoints (environment-based auth)
	r.Get("/.well-known/oauth-authorization-server", handlers.OAuth2MetadataHandler)
	r.Get("/.well-known/openid-configuration", handlers.OIDCDiscoveryRootHandler)
	r.Get("/.well-known/jwks.json", handlers.OAuth2JWKSHandler)
	r.Get("/oauth2/authorize", handlers.OAuth2AuthorizeHandler)
	r.Post("/oauth2/authorize", handlers.OAuth2AuthorizeHandler)
	r.Get("/oauth2/callback", handlers.OAuth2CallbackHandler)
	r.Post("/oauth2/token", handlers.OAuth2TokenHandler)
	r.Get("/oauth2/userinfo", handlers.OAuth2UserInfoHandler)
	r.Get("/oauth2/demo", handlers.OAuth2DemoHandler)

	// Basic Auth (environment-based)
	r.Get("/basic-auth", handlers.BasicAuthEnvHandler)

	// Bearer Token Auth (environment-based)
	r.Get("/bearer-auth", handlers.BearerAuthEnvHandler)

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

	// API documentation endpoint
	r.Get("/", handlers.APIDocsHandler)

	log.Printf("Starting server on %s", cfg.Addr())
	if err := http.ListenAndServe(cfg.Addr(), r); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
