package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/types/known/durationpb"

	pb "github.com/jsr-probitas/echo-servers/echo-connectrpc/proto"
	"github.com/jsr-probitas/echo-servers/echo-connectrpc/proto/protoconnect"
)

const (
	// MaxPayloadSize is the maximum allowed payload size (10MB)
	MaxPayloadSize = 10 * 1024 * 1024
)

type EchoServer struct {
	protoconnect.UnimplementedEchoHandler
}

func NewEchoServer() *EchoServer {
	return &EchoServer{}
}

func (s *EchoServer) Echo(ctx context.Context, req *connect.Request[pb.EchoRequest]) (*connect.Response[pb.EchoResponse], error) {
	resp := &pb.EchoResponse{
		Message:  req.Msg.Message,
		Metadata: make(map[string]string),
	}

	// Echo back request headers
	for key, values := range req.Header() {
		if len(values) > 0 {
			resp.Metadata[key] = values[0]
		}
	}

	response := connect.NewResponse(resp)

	// Set response trailers
	for key, values := range req.Header() {
		if len(values) > 0 {
			response.Trailer().Set(key, values[0])
		}
	}

	return response, nil
}

func (s *EchoServer) EchoWithDelay(ctx context.Context, req *connect.Request[pb.EchoWithDelayRequest]) (*connect.Response[pb.EchoResponse], error) {
	if req.Msg.DelayMs > 0 {
		select {
		case <-time.After(time.Duration(req.Msg.DelayMs) * time.Millisecond):
		case <-ctx.Done():
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("context deadline exceeded"))
		}
	}

	resp := &pb.EchoResponse{
		Message:  req.Msg.Message,
		Metadata: make(map[string]string),
	}

	// Echo back request headers
	for key, values := range req.Header() {
		if len(values) > 0 {
			resp.Metadata[key] = values[0]
		}
	}

	response := connect.NewResponse(resp)

	// Set response trailers
	for key, values := range req.Header() {
		if len(values) > 0 {
			response.Trailer().Set(key, values[0])
		}
	}

	return response, nil
}

func (s *EchoServer) EchoError(_ context.Context, req *connect.Request[pb.EchoErrorRequest]) (*connect.Response[pb.EchoResponse], error) {
	code := connect.Code(req.Msg.Code)
	if code > 16 {
		code = connect.CodeUnknown
	}

	details := req.Msg.Details
	if details == "" {
		details = fmt.Sprintf("error with code %d: %s", req.Msg.Code, req.Msg.Message)
	}

	return nil, connect.NewError(code, fmt.Errorf("%s", details))
}

func (s *EchoServer) EchoRequestMetadata(ctx context.Context, req *connect.Request[pb.EchoRequestMetadataRequest]) (*connect.Response[pb.EchoRequestMetadataResponse], error) {
	resp := &pb.EchoRequestMetadataResponse{
		Metadata: make(map[string]*pb.MetadataValues),
	}

	headers := req.Header()

	// If specific keys requested, filter to those
	if len(req.Msg.Keys) > 0 {
		for _, key := range req.Msg.Keys {
			if values := headers.Values(key); len(values) > 0 {
				resp.Metadata[key] = &pb.MetadataValues{Values: values}
			}
		}
	} else {
		// Return all metadata
		for key, values := range headers {
			resp.Metadata[key] = &pb.MetadataValues{Values: values}
		}
	}

	return connect.NewResponse(resp), nil
}

func (s *EchoServer) EchoWithTrailers(ctx context.Context, req *connect.Request[pb.EchoWithTrailersRequest]) (*connect.Response[pb.EchoResponse], error) {
	resp := &pb.EchoResponse{
		Message:  req.Msg.Message,
		Metadata: make(map[string]string),
	}

	// Echo back request headers in response body
	for key, values := range req.Header() {
		if len(values) > 0 {
			resp.Metadata[key] = values[0]
		}
	}

	response := connect.NewResponse(resp)

	// Set specified trailers
	for k, v := range req.Msg.Trailers {
		response.Trailer().Set(k, v)
	}

	return response, nil
}

func (s *EchoServer) EchoLargePayload(_ context.Context, req *connect.Request[pb.EchoLargePayloadRequest]) (*connect.Response[pb.EchoLargePayloadResponse], error) {
	size := int(req.Msg.SizeBytes)
	if size <= 0 {
		size = 1
	}
	if size > MaxPayloadSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("requested size %d exceeds maximum %d bytes", size, MaxPayloadSize))
	}

	pattern := req.Msg.Pattern
	if pattern == "" {
		pattern = "X"
	}

	// Generate payload by repeating pattern
	patternBytes := []byte(pattern)
	payload := bytes.Repeat(patternBytes, (size/len(patternBytes))+1)
	payload = payload[:size]

	resp := &pb.EchoLargePayloadResponse{
		Payload:    payload,
		ActualSize: int32(len(payload)),
	}

	return connect.NewResponse(resp), nil
}

func (s *EchoServer) EchoDeadline(ctx context.Context, req *connect.Request[pb.EchoDeadlineRequest]) (*connect.Response[pb.EchoDeadlineResponse], error) {
	resp := &pb.EchoDeadlineResponse{
		Message:     req.Msg.Message,
		HasDeadline: false,
	}

	deadline, ok := ctx.Deadline()
	if ok {
		resp.HasDeadline = true
		remaining := time.Until(deadline)
		resp.DeadlineRemainingMs = remaining.Milliseconds()
	} else {
		resp.DeadlineRemainingMs = -1
	}

	return connect.NewResponse(resp), nil
}

func (s *EchoServer) EchoErrorWithDetails(_ context.Context, req *connect.Request[pb.EchoErrorWithDetailsRequest]) (*connect.Response[pb.EchoResponse], error) {
	code := connect.Code(req.Msg.Code)
	if code > 16 {
		code = connect.CodeUnknown
	}

	message := req.Msg.Message
	if message == "" {
		message = fmt.Sprintf("error with code %d", req.Msg.Code)
	}

	err := connect.NewError(code, fmt.Errorf("%s", message))

	// Add rich error details
	for _, detail := range req.Msg.Details {
		switch detail.Type {
		case "bad_request":
			br := &errdetails.BadRequest{}
			for _, fv := range detail.FieldViolations {
				br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fv.Field,
					Description: fv.Description,
				})
			}
			if d, detailErr := connect.NewErrorDetail(br); detailErr == nil {
				err.AddDetail(d)
			}
		case "retry_info":
			ri := &errdetails.RetryInfo{
				RetryDelay: durationpb.New(time.Duration(detail.RetryDelayMs) * time.Millisecond),
			}
			if d, detailErr := connect.NewErrorDetail(ri); detailErr == nil {
				err.AddDetail(d)
			}
		case "debug_info":
			di := &errdetails.DebugInfo{
				StackEntries: detail.StackEntries,
				Detail:       detail.DebugDetail,
			}
			if d, detailErr := connect.NewErrorDetail(di); detailErr == nil {
				err.AddDetail(d)
			}
		case "quota_failure":
			qf := &errdetails.QuotaFailure{}
			for _, qv := range detail.QuotaViolations {
				qf.Violations = append(qf.Violations, &errdetails.QuotaFailure_Violation{
					Subject:     qv.Subject,
					Description: qv.Description,
				})
			}
			if d, detailErr := connect.NewErrorDetail(qf); detailErr == nil {
				err.AddDetail(d)
			}
		}
	}

	return nil, err
}

func (s *EchoServer) ServerStream(ctx context.Context, req *connect.Request[pb.ServerStreamRequest], stream *connect.ServerStream[pb.EchoResponse]) error {
	md := make(map[string]string)

	// Collect request headers
	for key, values := range req.Header() {
		if len(values) > 0 {
			md[key] = values[0]
		}
	}

	count := req.Msg.Count
	if count <= 0 {
		count = 1
	}

	interval := time.Duration(req.Msg.IntervalMs) * time.Millisecond

	for i := int32(0); i < count; i++ {
		select {
		case <-ctx.Done():
			return connect.NewError(connect.CodeCanceled, fmt.Errorf("stream canceled"))
		default:
		}

		resp := &pb.EchoResponse{
			Message:  fmt.Sprintf("%s [%d/%d]", req.Msg.Message, i+1, count),
			Metadata: md,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		if i < count-1 && interval > 0 {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return connect.NewError(connect.CodeCanceled, fmt.Errorf("stream canceled"))
			}
		}
	}

	return nil
}

func (s *EchoServer) ClientStream(ctx context.Context, stream *connect.ClientStream[pb.EchoRequest]) (*connect.Response[pb.EchoResponse], error) {
	md := make(map[string]string)

	// Collect request headers
	for key, values := range stream.RequestHeader() {
		if len(values) > 0 {
			md[key] = values[0]
		}
	}

	var messages []string

	for stream.Receive() {
		req := stream.Msg()
		messages = append(messages, req.Message)
	}

	if err := stream.Err(); err != nil {
		return nil, err
	}

	resp := &pb.EchoResponse{
		Message:  strings.Join(messages, ", "),
		Metadata: md,
	}

	return connect.NewResponse(resp), nil
}

func (s *EchoServer) BidirectionalStream(ctx context.Context, stream *connect.BidiStream[pb.EchoRequest, pb.EchoResponse]) error {
	md := make(map[string]string)

	// Collect request headers
	for key, values := range stream.RequestHeader() {
		if len(values) > 0 {
			md[key] = values[0]
		}
	}

	for {
		select {
		case <-ctx.Done():
			return connect.NewError(connect.CodeCanceled, fmt.Errorf("stream canceled"))
		default:
		}

		req, err := stream.Receive()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		resp := &pb.EchoResponse{
			Message:  req.Message,
			Metadata: md,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}
