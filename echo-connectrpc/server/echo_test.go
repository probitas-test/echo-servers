package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	pb "github.com/jsr-probitas/echo-servers/echo-connectrpc/proto"
	"github.com/jsr-probitas/echo-servers/echo-connectrpc/proto/protoconnect"
)

func setupTestServer(t *testing.T) (protoconnect.EchoClient, *httptest.Server) {
	t.Helper()

	mux := http.NewServeMux()
	echoServer := NewEchoServer()
	path, handler := protoconnect.NewEchoHandler(echoServer)
	mux.Handle(path, handler)

	server := httptest.NewServer(mux)
	client := protoconnect.NewEchoClient(http.DefaultClient, server.URL)

	return client, server
}

func TestEcho_ReturnsSameMessage(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	resp, err := client.Echo(context.Background(), connect.NewRequest(&pb.EchoRequest{
		Message: "hello",
	}))

	if err != nil {
		t.Fatalf("Echo failed: %v", err)
	}
	if resp.Msg.Message != "hello" {
		t.Errorf("expected message %q, got %q", "hello", resp.Msg.Message)
	}
}

func TestEcho_IncludesMetadata(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	req := connect.NewRequest(&pb.EchoRequest{
		Message: "hello",
	})
	req.Header().Set("X-Custom-Header", "custom-value")

	resp, err := client.Echo(context.Background(), req)

	if err != nil {
		t.Fatalf("Echo failed: %v", err)
	}
	if resp.Msg.Metadata["X-Custom-Header"] != "custom-value" {
		t.Errorf("expected metadata X-Custom-Header=%q, got %q", "custom-value", resp.Msg.Metadata["X-Custom-Header"])
	}
}

func TestEchoWithDelay_ReturnsAfterDelay(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	delayMs := int32(50)
	start := time.Now()

	resp, err := client.EchoWithDelay(context.Background(), connect.NewRequest(&pb.EchoWithDelayRequest{
		Message: "delayed",
		DelayMs: delayMs,
	}))

	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("EchoWithDelay failed: %v", err)
	}
	if resp.Msg.Message != "delayed" {
		t.Errorf("expected message %q, got %q", "delayed", resp.Msg.Message)
	}
	if elapsed < time.Duration(delayMs)*time.Millisecond {
		t.Errorf("expected delay of at least %dms, got %v", delayMs, elapsed)
	}
}

func TestEchoError_ReturnsCorrectStatusCode(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	tests := []struct {
		name     string
		code     int32
		details  string
		wantCode connect.Code
	}{
		{
			name:     "InvalidArgument",
			code:     int32(connect.CodeInvalidArgument),
			details:  "invalid input",
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name:     "NotFound",
			code:     int32(connect.CodeNotFound),
			details:  "resource not found",
			wantCode: connect.CodeNotFound,
		},
		{
			name:     "PermissionDenied",
			code:     int32(connect.CodePermissionDenied),
			details:  "access denied",
			wantCode: connect.CodePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.EchoError(context.Background(), connect.NewRequest(&pb.EchoErrorRequest{
				Message: "error test",
				Code:    tt.code,
				Details: tt.details,
			}))

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			connectErr, ok := err.(*connect.Error)
			if !ok {
				t.Fatalf("expected connect.Error, got %T", err)
			}
			if connectErr.Code() != tt.wantCode {
				t.Errorf("expected code %v, got %v", tt.wantCode, connectErr.Code())
			}
			if connectErr.Message() != tt.details {
				t.Errorf("expected message %q, got %q", tt.details, connectErr.Message())
			}
		})
	}
}

func TestServerStream_ReturnsCorrectCount(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	stream, err := client.ServerStream(context.Background(), connect.NewRequest(&pb.ServerStreamRequest{
		Message:    "stream",
		Count:      5,
		IntervalMs: 0,
	}))
	if err != nil {
		t.Fatalf("ServerStream failed: %v", err)
	}

	count := 0
	for stream.Receive() {
		count++
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	if count != 5 {
		t.Errorf("expected 5 messages, got %d", count)
	}
}

func TestServerStream_MessagesContainCorrectContent(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	stream, err := client.ServerStream(context.Background(), connect.NewRequest(&pb.ServerStreamRequest{
		Message:    "hello",
		Count:      3,
		IntervalMs: 0,
	}))
	if err != nil {
		t.Fatalf("ServerStream failed: %v", err)
	}

	expected := []string{
		"hello [1/3]",
		"hello [2/3]",
		"hello [3/3]",
	}

	i := 0
	for stream.Receive() {
		if i >= len(expected) {
			t.Fatalf("received more messages than expected")
		}
		msg := stream.Msg()
		if msg.Message != expected[i] {
			t.Errorf("message %d: expected %q, got %q", i, expected[i], msg.Message)
		}
		i++
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("Stream error: %v", err)
	}
}

func TestClientStream_AggregatesMessages(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	stream := client.ClientStream(context.Background())

	messages := []string{"one", "two", "three"}
	for _, msg := range messages {
		if err := stream.Send(&pb.EchoRequest{Message: msg}); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	resp, err := stream.CloseAndReceive()
	if err != nil {
		t.Fatalf("CloseAndReceive failed: %v", err)
	}

	want := "one, two, three"
	if resp.Msg.Message != want {
		t.Errorf("expected %q, got %q", want, resp.Msg.Message)
	}
}

func TestBidirectionalStream_EchoesEachMessage(t *testing.T) {
	t.Skip("Bidirectional streaming requires HTTP/2, httptest.Server only supports HTTP/1.1")
	// Note: This functionality is tested via integration tests with actual server
}

func TestEchoWithTrailers_SetsTrailers(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	resp, err := client.EchoWithTrailers(context.Background(), connect.NewRequest(&pb.EchoWithTrailersRequest{
		Message: "hello",
		Trailers: map[string]string{
			"x-custom-trailer": "trailer-value",
			"x-another":        "another-value",
		},
	}))

	if err != nil {
		t.Fatalf("EchoWithTrailers failed: %v", err)
	}
	if resp.Msg.Message != "hello" {
		t.Errorf("expected message %q, got %q", "hello", resp.Msg.Message)
	}

	// Check trailers were set
	if val := resp.Trailer().Get("x-custom-trailer"); val != "trailer-value" {
		t.Errorf("expected trailer x-custom-trailer=trailer-value, got %q", val)
	}
	if val := resp.Trailer().Get("x-another"); val != "another-value" {
		t.Errorf("expected trailer x-another=another-value, got %q", val)
	}
}

func TestEchoWithTrailers_NoTrailers(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	resp, err := client.EchoWithTrailers(context.Background(), connect.NewRequest(&pb.EchoWithTrailersRequest{
		Message: "hello",
	}))

	if err != nil {
		t.Fatalf("EchoWithTrailers failed: %v", err)
	}
	if resp.Msg.Message != "hello" {
		t.Errorf("expected message %q, got %q", "hello", resp.Msg.Message)
	}
}

func TestEchoRequestMetadata_ReturnsAllMetadata(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	req := connect.NewRequest(&pb.EchoRequestMetadataRequest{})
	req.Header().Set("X-Auth-Token", "bearer-123")
	req.Header().Set("X-Request-Id", "req-456")

	resp, err := client.EchoRequestMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("EchoRequestMetadata failed: %v", err)
	}

	if resp.Msg.Metadata["X-Auth-Token"] == nil || resp.Msg.Metadata["X-Auth-Token"].Values[0] != "bearer-123" {
		t.Errorf("expected X-Auth-Token=bearer-123, got %v", resp.Msg.Metadata["X-Auth-Token"])
	}
	if resp.Msg.Metadata["X-Request-Id"] == nil || resp.Msg.Metadata["X-Request-Id"].Values[0] != "req-456" {
		t.Errorf("expected X-Request-Id=req-456, got %v", resp.Msg.Metadata["X-Request-Id"])
	}
}

func TestEchoRequestMetadata_FiltersToSpecificKeys(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	req := connect.NewRequest(&pb.EchoRequestMetadataRequest{
		Keys: []string{"X-Auth-Token"},
	})
	req.Header().Set("X-Auth-Token", "bearer-123")
	req.Header().Set("X-Request-Id", "req-456")
	req.Header().Set("X-Other", "other-value")

	resp, err := client.EchoRequestMetadata(context.Background(), req)
	if err != nil {
		t.Fatalf("EchoRequestMetadata failed: %v", err)
	}

	if resp.Msg.Metadata["X-Auth-Token"] == nil {
		t.Error("expected X-Auth-Token to be present")
	}
	if resp.Msg.Metadata["X-Request-Id"] != nil {
		t.Error("expected X-Request-Id to be absent (filtered)")
	}
}

func TestEchoLargePayload_ReturnsCorrectSize(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

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
			resp, err := client.EchoLargePayload(context.Background(), connect.NewRequest(&pb.EchoLargePayloadRequest{
				SizeBytes: tt.size,
				Pattern:   tt.pattern,
			}))
			if err != nil {
				t.Fatalf("EchoLargePayload failed: %v", err)
			}

			if resp.Msg.ActualSize != tt.size {
				t.Errorf("expected size %d, got %d", tt.size, resp.Msg.ActualSize)
			}
			if len(resp.Msg.Payload) != int(tt.size) {
				t.Errorf("expected payload length %d, got %d", tt.size, len(resp.Msg.Payload))
			}
		})
	}
}

func TestEchoLargePayload_RejectsOversizedRequest(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	_, err := client.EchoLargePayload(context.Background(), connect.NewRequest(&pb.EchoLargePayloadRequest{
		SizeBytes: MaxPayloadSize + 1,
	}))

	if err == nil {
		t.Fatal("expected error for oversized request")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}
	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connectErr.Code())
	}
}

func TestEchoDeadline_WithDeadline(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.EchoDeadline(ctx, connect.NewRequest(&pb.EchoDeadlineRequest{
		Message: "deadline test",
	}))
	if err != nil {
		t.Fatalf("EchoDeadline failed: %v", err)
	}

	if resp.Msg.Message != "deadline test" {
		t.Errorf("expected message %q, got %q", "deadline test", resp.Msg.Message)
	}
	if !resp.Msg.HasDeadline {
		t.Error("expected HasDeadline=true")
	}
	if resp.Msg.DeadlineRemainingMs <= 0 {
		t.Errorf("expected positive deadline remaining, got %d", resp.Msg.DeadlineRemainingMs)
	}
}

func TestEchoDeadline_WithoutDeadline(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	resp, err := client.EchoDeadline(context.Background(), connect.NewRequest(&pb.EchoDeadlineRequest{
		Message: "no deadline",
	}))
	if err != nil {
		t.Fatalf("EchoDeadline failed: %v", err)
	}

	if resp.Msg.HasDeadline {
		t.Error("expected HasDeadline=false")
	}
	if resp.Msg.DeadlineRemainingMs != -1 {
		t.Errorf("expected DeadlineRemainingMs=-1, got %d", resp.Msg.DeadlineRemainingMs)
	}
}

func TestEchoErrorWithDetails_BadRequest(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	_, err := client.EchoErrorWithDetails(context.Background(), connect.NewRequest(&pb.EchoErrorWithDetailsRequest{
		Code:    int32(connect.CodeInvalidArgument),
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
	}))

	if err == nil {
		t.Fatal("expected error")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeInvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", connectErr.Code())
	}

	details := connectErr.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	br, err := details[0].Value()
	if err != nil {
		t.Fatalf("failed to get detail value: %v", err)
	}

	badReq, ok := br.(*errdetails.BadRequest)
	if !ok {
		t.Fatalf("expected *errdetails.BadRequest, got %T", br)
	}

	if len(badReq.FieldViolations) != 2 {
		t.Errorf("expected 2 field violations, got %d", len(badReq.FieldViolations))
	}
}

func TestEchoErrorWithDetails_RetryInfo(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	_, err := client.EchoErrorWithDetails(context.Background(), connect.NewRequest(&pb.EchoErrorWithDetailsRequest{
		Code:    int32(connect.CodeUnavailable),
		Message: "service unavailable",
		Details: []*pb.ErrorDetail{
			{
				Type:         "retry_info",
				RetryDelayMs: 5000,
			},
		},
	}))

	if err == nil {
		t.Fatal("expected error")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	details := connectErr.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	ri, err := details[0].Value()
	if err != nil {
		t.Fatalf("failed to get detail value: %v", err)
	}

	retryInfo, ok := ri.(*errdetails.RetryInfo)
	if !ok {
		t.Fatalf("expected *errdetails.RetryInfo, got %T", ri)
	}

	if retryInfo.RetryDelay.Seconds != 5 {
		t.Errorf("expected 5 second retry delay, got %v", retryInfo.RetryDelay)
	}
}

func TestEchoErrorWithDetails_DebugInfo(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	_, err := client.EchoErrorWithDetails(context.Background(), connect.NewRequest(&pb.EchoErrorWithDetailsRequest{
		Code:    int32(connect.CodeInternal),
		Message: "internal error",
		Details: []*pb.ErrorDetail{
			{
				Type:         "debug_info",
				StackEntries: []string{"main.go:42", "handler.go:15"},
				DebugDetail:  "null pointer exception",
			},
		},
	}))

	if err == nil {
		t.Fatal("expected error")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeInternal {
		t.Errorf("expected Internal, got %v", connectErr.Code())
	}

	details := connectErr.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	di, err := details[0].Value()
	if err != nil {
		t.Fatalf("failed to get detail value: %v", err)
	}

	debugInfo, ok := di.(*errdetails.DebugInfo)
	if !ok {
		t.Fatalf("expected *errdetails.DebugInfo, got %T", di)
	}

	if len(debugInfo.StackEntries) != 2 {
		t.Errorf("expected 2 stack entries, got %d", len(debugInfo.StackEntries))
	}
	if debugInfo.Detail != "null pointer exception" {
		t.Errorf("expected detail %q, got %q", "null pointer exception", debugInfo.Detail)
	}
}

func TestEchoErrorWithDetails_QuotaFailure(t *testing.T) {
	client, server := setupTestServer(t)
	defer server.Close()

	_, err := client.EchoErrorWithDetails(context.Background(), connect.NewRequest(&pb.EchoErrorWithDetailsRequest{
		Code:    int32(connect.CodeResourceExhausted),
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
	}))

	if err == nil {
		t.Fatal("expected error")
	}

	connectErr, ok := err.(*connect.Error)
	if !ok {
		t.Fatalf("expected connect.Error, got %T", err)
	}

	if connectErr.Code() != connect.CodeResourceExhausted {
		t.Errorf("expected ResourceExhausted, got %v", connectErr.Code())
	}

	details := connectErr.Details()
	if len(details) == 0 {
		t.Fatal("expected error details")
	}

	qf, err := details[0].Value()
	if err != nil {
		t.Fatalf("failed to get detail value: %v", err)
	}

	quotaFailure, ok := qf.(*errdetails.QuotaFailure)
	if !ok {
		t.Fatalf("expected *errdetails.QuotaFailure, got %T", qf)
	}

	if len(quotaFailure.Violations) != 2 {
		t.Errorf("expected 2 quota violations, got %d", len(quotaFailure.Violations))
	}
	if quotaFailure.Violations[0].Subject != "user:123" {
		t.Errorf("expected subject %q, got %q", "user:123", quotaFailure.Violations[0].Subject)
	}
}
