package server

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	pb "github.com/jsr-probitas/echo-servers/echo-grpc/proto"
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

func TestEchoWithTrailers_SetsTrailers(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	var trailer metadata.MD
	resp, err := client.EchoWithTrailers(context.Background(),
		&pb.EchoWithTrailersRequest{
			Message: "hello",
			Trailers: map[string]string{
				"x-custom-trailer": "trailer-value",
				"x-another":        "another-value",
			},
		},
		grpc.Trailer(&trailer),
	)

	if err != nil {
		t.Fatalf("EchoWithTrailers failed: %v", err)
	}
	if resp.Message != "hello" {
		t.Errorf("expected message %q, got %q", "hello", resp.Message)
	}

	// Check trailers were set
	if vals := trailer.Get("x-custom-trailer"); len(vals) == 0 || vals[0] != "trailer-value" {
		t.Errorf("expected trailer x-custom-trailer=trailer-value, got %v", vals)
	}
	if vals := trailer.Get("x-another"); len(vals) == 0 || vals[0] != "another-value" {
		t.Errorf("expected trailer x-another=another-value, got %v", vals)
	}
}

func TestEchoWithTrailers_NoTrailers(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.EchoWithTrailers(context.Background(),
		&pb.EchoWithTrailersRequest{
			Message: "hello",
		},
	)

	if err != nil {
		t.Fatalf("EchoWithTrailers failed: %v", err)
	}
	if resp.Message != "hello" {
		t.Errorf("expected message %q, got %q", "hello", resp.Message)
	}
}

func TestEchoRequestMetadata_ReturnsAllMetadata(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"x-auth-token", "bearer-123",
		"x-request-id", "req-456",
	))

	resp, err := client.EchoRequestMetadata(ctx, &pb.EchoRequestMetadataRequest{})
	if err != nil {
		t.Fatalf("EchoRequestMetadata failed: %v", err)
	}

	if resp.Metadata["x-auth-token"] == nil || resp.Metadata["x-auth-token"].Values[0] != "bearer-123" {
		t.Errorf("expected x-auth-token=bearer-123, got %v", resp.Metadata["x-auth-token"])
	}
	if resp.Metadata["x-request-id"] == nil || resp.Metadata["x-request-id"].Values[0] != "req-456" {
		t.Errorf("expected x-request-id=req-456, got %v", resp.Metadata["x-request-id"])
	}
}

func TestEchoRequestMetadata_FiltersToSpecificKeys(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"x-auth-token", "bearer-123",
		"x-request-id", "req-456",
		"x-other", "other-value",
	))

	resp, err := client.EchoRequestMetadata(ctx, &pb.EchoRequestMetadataRequest{
		Keys: []string{"x-auth-token"},
	})
	if err != nil {
		t.Fatalf("EchoRequestMetadata failed: %v", err)
	}

	if resp.Metadata["x-auth-token"] == nil {
		t.Error("expected x-auth-token to be present")
	}
	if resp.Metadata["x-request-id"] != nil {
		t.Error("expected x-request-id to be absent (filtered)")
	}
}

func TestEchoLargePayload_ReturnsCorrectSize(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name    string
		size    int32
		pattern string
	}{
		{"small payload", 100, ""},
		{"medium payload", 1024, "ABC"},
		{"custom pattern", 50, "XY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.EchoLargePayload(context.Background(), &pb.EchoLargePayloadRequest{
				SizeBytes: tt.size,
				Pattern:   tt.pattern,
			})
			if err != nil {
				t.Fatalf("EchoLargePayload failed: %v", err)
			}

			if resp.ActualSize != tt.size {
				t.Errorf("expected size %d, got %d", tt.size, resp.ActualSize)
			}
			if len(resp.Payload) != int(tt.size) {
				t.Errorf("expected payload length %d, got %d", tt.size, len(resp.Payload))
			}
		})
	}
}

func TestEchoLargePayload_RejectsOversizedRequest(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.EchoLargePayload(context.Background(), &pb.EchoLargePayloadRequest{
		SizeBytes: MaxPayloadSize + 1,
	})

	if err == nil {
		t.Fatal("expected error for oversized request")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}
}

func TestEchoDeadline_WithDeadline(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.EchoDeadline(ctx, &pb.EchoDeadlineRequest{
		Message: "deadline test",
	})
	if err != nil {
		t.Fatalf("EchoDeadline failed: %v", err)
	}

	if resp.Message != "deadline test" {
		t.Errorf("expected message %q, got %q", "deadline test", resp.Message)
	}
	if !resp.HasDeadline {
		t.Error("expected HasDeadline=true")
	}
	if resp.DeadlineRemainingMs <= 0 {
		t.Errorf("expected positive deadline remaining, got %d", resp.DeadlineRemainingMs)
	}
}

func TestEchoDeadline_WithoutDeadline(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	resp, err := client.EchoDeadline(context.Background(), &pb.EchoDeadlineRequest{
		Message: "no deadline",
	})
	if err != nil {
		t.Fatalf("EchoDeadline failed: %v", err)
	}

	if resp.HasDeadline {
		t.Error("expected HasDeadline=false")
	}
	if resp.DeadlineRemainingMs != -1 {
		t.Errorf("expected DeadlineRemainingMs=-1, got %d", resp.DeadlineRemainingMs)
	}
}

func TestEchoErrorWithDetails_BadRequest(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.EchoErrorWithDetails(context.Background(), &pb.EchoErrorWithDetailsRequest{
		Code:    int32(codes.InvalidArgument),
		Message: "validation failed",
		Details: []*pb.ErrorDetail{
			{
				Type: "bad_request",
				FieldViolations: []*pb.FieldViolation{
					{Field: "email", Description: "invalid email format"},
					{Field: "age", Description: "must be positive"},
				},
			},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}

	details := st.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	br, ok := details[0].(*errdetails.BadRequest)
	if !ok {
		t.Fatalf("expected BadRequest detail, got %T", details[0])
	}

	if len(br.FieldViolations) != 2 {
		t.Errorf("expected 2 field violations, got %d", len(br.FieldViolations))
	}
}

func TestEchoErrorWithDetails_RetryInfo(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.EchoErrorWithDetails(context.Background(), &pb.EchoErrorWithDetailsRequest{
		Code:    int32(codes.Unavailable),
		Message: "service unavailable",
		Details: []*pb.ErrorDetail{
			{
				Type:         "retry_info",
				RetryDelayMs: 5000,
			},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	details := st.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	ri, ok := details[0].(*errdetails.RetryInfo)
	if !ok {
		t.Fatalf("expected RetryInfo detail, got %T", details[0])
	}

	if ri.RetryDelay.Seconds != 5 {
		t.Errorf("expected 5 second retry delay, got %v", ri.RetryDelay)
	}
}

func TestEchoErrorWithDetails_DebugInfo(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.EchoErrorWithDetails(context.Background(), &pb.EchoErrorWithDetailsRequest{
		Code:    int32(codes.Internal),
		Message: "internal error",
		Details: []*pb.ErrorDetail{
			{
				Type:         "debug_info",
				StackEntries: []string{"main.go:42", "handler.go:15"},
				DebugDetail:  "null pointer exception",
			},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	if st.Code() != codes.Internal {
		t.Errorf("expected Internal, got %v", st.Code())
	}

	details := st.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	di, ok := details[0].(*errdetails.DebugInfo)
	if !ok {
		t.Fatalf("expected DebugInfo detail, got %T", details[0])
	}

	if len(di.StackEntries) != 2 {
		t.Errorf("expected 2 stack entries, got %d", len(di.StackEntries))
	}
	if di.Detail != "null pointer exception" {
		t.Errorf("expected detail %q, got %q", "null pointer exception", di.Detail)
	}
}

func TestEchoErrorWithDetails_QuotaFailure(t *testing.T) {
	client, cleanup := setupTestServer(t)
	defer cleanup()

	_, err := client.EchoErrorWithDetails(context.Background(), &pb.EchoErrorWithDetailsRequest{
		Code:    int32(codes.ResourceExhausted),
		Message: "quota exceeded",
		Details: []*pb.ErrorDetail{
			{
				Type: "quota_failure",
				QuotaViolations: []*pb.QuotaViolation{
					{Subject: "user:123", Description: "API calls per minute exceeded"},
					{Subject: "project:abc", Description: "Storage limit reached"},
				},
			},
		},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %v", err)
	}

	if st.Code() != codes.ResourceExhausted {
		t.Errorf("expected ResourceExhausted, got %v", st.Code())
	}

	details := st.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	qf, ok := details[0].(*errdetails.QuotaFailure)
	if !ok {
		t.Fatalf("expected QuotaFailure detail, got %T", details[0])
	}

	if len(qf.Violations) != 2 {
		t.Errorf("expected 2 quota violations, got %d", len(qf.Violations))
	}
	if qf.Violations[0].Subject != "user:123" {
		t.Errorf("expected subject %q, got %q", "user:123", qf.Violations[0].Subject)
	}
}
