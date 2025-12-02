package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	pb "github.com/jsr-probitas/echo-servers/echo-grpc/proto"
	"github.com/jsr-probitas/echo-servers/echo-grpc/server"
)

func main() {
	cfg := LoadConfig()

	lis, err := net.Listen("tcp", cfg.Addr())
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// Register echo service
	echoServer := server.NewEchoServer()
	pb.RegisterEchoServer(s, echoServer)

	// Register health service (grpc.health.v1)
	healthServer := server.NewHealthServer()
	healthpb.RegisterHealthServer(s, healthServer)

	// Enable server reflection (v1 and v1alpha)
	server.RegisterReflection(s, cfg.ReflectionIncludeDeps, cfg.DisableReflectionV1, cfg.DisableReflectionV1Alpha)

	log.Printf("Starting server on %s", cfg.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
