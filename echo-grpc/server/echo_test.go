package server

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/jsr-probitas/dockerfiles/echo-grpc/proto"
)

func setupTestServer(t *testing.T) (pb.EchoClient, func()) {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	pb.RegisterEchoServer(s, NewEchoServer())

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("server exited: %v", err)
		}
	}()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		s.Stop()
	}

	return pb.NewEchoClient(conn), cleanup
}

func TestEcho_ReturnsSameMessage(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.Echo(context.Background(), &pb.EchoRequest{
		Message: "hello",
	})

	if err != nil {
		t.Fatalf("Echo failed: %v", err)
	}
	if resp.Message != "hello" {
		t.Errorf("expected message %q, got %q", "hello", resp.Message)
	}
}

func TestEcho_IncludesMetadata(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"x-custom-header", "custom-value",
	))

	resp, err := client.Echo(ctx, &pb.EchoRequest{
		Message: "hello",
	})

	if err != nil {
		t.Fatalf("Echo failed: %v", err)
	}
	if resp.Metadata["x-custom-header"] != "custom-value" {
		t.Errorf("expected metadata x-custom-header=%q, got %q", "custom-value", resp.Metadata["x-custom-header"])
	}
}

func TestEchoWithDelay_ReturnsAfterDelay(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	delayMs := int32(10)
	start := time.Now()

	resp, err := client.EchoWithDelay(context.Background(), &pb.EchoWithDelayRequest{
		Message: "delayed",
		DelayMs: delayMs,
	})

	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("EchoWithDelay failed: %v", err)
	}
	if resp.Message != "delayed" {
		t.Errorf("expected message %q, got %q", "delayed", resp.Message)
	}
	if elapsed < time.Duration(delayMs)*time.Millisecond {
		t.Errorf("expected delay of at least %dms, got %v", delayMs, elapsed)
	}
}

func TestEchoError_ReturnsCorrectStatusCode(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name     string
		code     int32
		details  string
		wantCode codes.Code
	}{
		{
			name:     "InvalidArgument",
			code:     int32(codes.InvalidArgument),
			details:  "invalid input",
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "NotFound",
			code:     int32(codes.NotFound),
			details:  "resource not found",
			wantCode: codes.NotFound,
		},
		{
			name:     "PermissionDenied",
			code:     int32(codes.PermissionDenied),
			details:  "access denied",
			wantCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.EchoError(context.Background(), &pb.EchoErrorRequest{
				Message: "error test",
				Code:    tt.code,
				Details: tt.details,
			})

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %v", err)
			}
			if st.Code() != tt.wantCode {
				t.Errorf("expected code %v, got %v", tt.wantCode, st.Code())
			}
			if st.Message() != tt.details {
				t.Errorf("expected message %q, got %q", tt.details, st.Message())
			}
		})
	}
}

func TestServerStream_ReturnsCorrectCount(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.ServerStream(context.Background(), &pb.ServerStreamRequest{
		Message:    "stream",
		Count:      5,
		IntervalMs: 0,
	})
	if err != nil {
		t.Fatalf("ServerStream failed: %v", err)
	}

	count := 0
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 messages, got %d", count)
	}
}

func TestServerStream_MessagesContainCorrectContent(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.ServerStream(context.Background(), &pb.ServerStreamRequest{
		Message:    "hello",
		Count:      3,
		IntervalMs: 0,
	})
	if err != nil {
		t.Fatalf("ServerStream failed: %v", err)
	}

	expected := []string{
		"hello [1/3]",
		"hello [2/3]",
		"hello [3/3]",
	}

	for i, want := range expected {
		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv failed at message %d: %v", i, err)
		}
		if resp.Message != want {
			t.Errorf("message %d: expected %q, got %q", i, want, resp.Message)
		}
	}
}

func TestClientStream_AggregatesMessages(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.ClientStream(context.Background())
	if err != nil {
		t.Fatalf("ClientStream failed: %v", err)
	}

	messages := []string{"one", "two", "three"}
	for _, msg := range messages {
		if err := stream.Send(&pb.EchoRequest{Message: msg}); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv failed: %v", err)
	}

	want := "one, two, three"
	if resp.Message != want {
		t.Errorf("expected %q, got %q", want, resp.Message)
	}
}

func TestBidirectionalStream_EchoesEachMessage(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	stream, err := client.BidirectionalStream(context.Background())
	if err != nil {
		t.Fatalf("BidirectionalStream failed: %v", err)
	}

	messages := []string{"first", "second", "third"}

	for _, msg := range messages {
		if err := stream.Send(&pb.EchoRequest{Message: msg}); err != nil {
			t.Fatalf("Send failed: %v", err)
		}

		resp, err := stream.Recv()
		if err != nil {
			t.Fatalf("Recv failed: %v", err)
		}

		if resp.Message != msg {
			t.Errorf("expected %q, got %q", msg, resp.Message)
		}
	}

	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend failed: %v", err)
	}
}
