package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/jsr-probitas/echo-servers/echo-connectrpc/proto/protoconnect"
	"github.com/jsr-probitas/echo-servers/echo-connectrpc/server"
)

func main() {
	cfg := LoadConfig()

	// Validate that at least one protocol is enabled
	if cfg.DisableConnectRPC && cfg.DisableGRPC && cfg.DisableGRPCWeb {
		log.Fatal("At least one protocol must be enabled (ConnectRPC, gRPC, or gRPC-Web)")
	}

	mux := http.NewServeMux()

	// Prepare handler options for protocol control
	var handlerOpts []connect.HandlerOption

	// Determine which protocols to support
	protocols := []string{}
	if !cfg.DisableConnectRPC {
		protocols = append(protocols, connect.ProtocolConnect)
	}
	if !cfg.DisableGRPC {
		protocols = append(protocols, connect.ProtocolGRPC)
	}
	if !cfg.DisableGRPCWeb {
		protocols = append(protocols, connect.ProtocolGRPCWeb)
	}

	// Log enabled protocols
	log.Printf("Enabled protocols: %v", protocols)

	// Register echo service
	echoServer := server.NewEchoServer()
	path, handler := protoconnect.NewEchoHandler(echoServer, handlerOpts...)
	mux.Handle(path, protocolFilterMiddleware(cfg, handler))

	// Register health check service
	checker := grpchealth.NewStaticChecker(
		protoconnect.EchoName,
	)
	healthPath, healthHandler := grpchealth.NewHandler(checker, handlerOpts...)
	mux.Handle(healthPath, protocolFilterMiddleware(cfg, healthHandler))

	// Register reflection service
	if !cfg.ReflectionIncludeDeps {
		// By default, grpcreflect includes dependencies
		// We need to use custom options if we want to exclude them
		// For now, we'll document this limitation
		log.Printf("Note: REFLECTION_INCLUDE_DEPENDENCIES is set to %v", cfg.ReflectionIncludeDeps)
	}

	// Build list of services for reflection
	reflectionServices := []string{
		protoconnect.EchoName,
		grpchealth.HealthV1ServiceName,
	}

	if !cfg.DisableReflectionV1 {
		reflectionServices = append(reflectionServices, grpcreflect.ReflectV1ServiceName)
	}
	if !cfg.DisableReflectionV1Alpha {
		reflectionServices = append(reflectionServices, grpcreflect.ReflectV1AlphaServiceName)
	}

	// Create reflector
	reflector := grpcreflect.NewStaticReflector(reflectionServices...)

	if !cfg.DisableReflectionV1 {
		v1Path, v1Handler := grpcreflect.NewHandlerV1(reflector, handlerOpts...)
		mux.Handle(v1Path, protocolFilterMiddleware(cfg, v1Handler))
		log.Printf("Registered reflection v1")
	} else {
		log.Printf("Reflection v1 disabled")
	}

	if !cfg.DisableReflectionV1Alpha {
		v1AlphaPath, v1AlphaHandler := grpcreflect.NewHandlerV1Alpha(reflector, handlerOpts...)
		mux.Handle(v1AlphaPath, protocolFilterMiddleware(cfg, v1AlphaHandler))
		log.Printf("Registered reflection v1alpha")
	} else {
		log.Printf("Reflection v1alpha disabled")
	}

	// Create server with h2c support (HTTP/2 without TLS)
	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Starting Connect RPC server on %s", cfg.Addr())
	log.Printf("Protocol configuration: ConnectRPC=%v, gRPC=%v, gRPC-Web=%v",
		!cfg.DisableConnectRPC, !cfg.DisableGRPC, !cfg.DisableGRPCWeb)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Failed to serve: %v", err)
	}

	log.Println("Server stopped")
}

// protocolFilterMiddleware filters requests based on the Connect protocol header
func protocolFilterMiddleware(cfg *Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		// Determine protocol from content type and headers
		// gRPC-Web has specific content type
		isGRPCWeb := contains(contentType, "application/grpc-web")
		// gRPC has application/grpc but not grpc-web
		isGRPC := contains(contentType, "application/grpc") && !isGRPCWeb
		// Connect RPC uses application/connect+, application/json, or application/proto
		isConnectRPC := contains(contentType, "application/connect+") ||
			contentType == "application/json" ||
			contentType == "application/proto" ||
			contains(contentType, "application/json;") ||
			contains(contentType, "application/proto;")

		// If it's a recognized protocol, check if it's disabled
		if isGRPC && cfg.DisableGRPC {
			http.Error(w, "gRPC protocol is disabled", http.StatusNotImplemented)
			return
		}
		if isGRPCWeb && cfg.DisableGRPCWeb {
			http.Error(w, "gRPC-Web protocol is disabled", http.StatusNotImplemented)
			return
		}
		if isConnectRPC && cfg.DisableConnectRPC {
			http.Error(w, "Connect RPC protocol is disabled", http.StatusNotImplemented)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
