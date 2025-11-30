//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/echo.proto

package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/jsr-probitas/dockerfiles/echo-grpc/proto"
	"github.com/jsr-probitas/dockerfiles/echo-grpc/server"
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

	// Enable server reflection (v1 and v1alpha)
	reflection.Register(s)

	log.Printf("Starting server on %s", cfg.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
