package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	pb "github.com/jsr-probitas/echo-servers/echo-grpc/proto"
)

const (
	// MaxPayloadSize is the maximum allowed payload size (10MB)
	MaxPayloadSize = 10 * 1024 * 1024
)

type EchoServer struct {
	pb.UnimplementedEchoServer
}

func NewEchoServer() *EchoServer {
	return &EchoServer{}
}

func (s *EchoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	resp := &pb.EchoResponse{
		Message:  req.Message,
		Metadata: make(map[string]string),
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range md {
			if len(v) > 0 {
				resp.Metadata[k] = v[0]
			}
		}
		_ = grpc.SetTrailer(ctx, md)
	}

	return resp, nil
}

func (s *EchoServer) EchoWithDelay(ctx context.Context, req *pb.EchoWithDelayRequest) (*pb.EchoResponse, error) {
	if req.DelayMs > 0 {
		select {
		case <-time.After(time.Duration(req.DelayMs) * time.Millisecond):
		case <-ctx.Done():
			return nil, status.Error(codes.DeadlineExceeded, "context deadline exceeded")
		}
	}

	resp := &pb.EchoResponse{
		Message:  req.Message,
		Metadata: make(map[string]string),
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range md {
			if len(v) > 0 {
				resp.Metadata[k] = v[0]
			}
		}
		_ = grpc.SetTrailer(ctx, md)
	}

	return resp, nil
}

func (s *EchoServer) EchoError(_ context.Context, req *pb.EchoErrorRequest) (*pb.EchoResponse, error) {
	code := codes.Code(req.Code)
	if code > 16 {
		code = codes.Unknown
	}

	details := req.Details
	if details == "" {
		details = fmt.Sprintf("error with code %d: %s", req.Code, req.Message)
	}

	return nil, status.Error(code, details)
}

func (s *EchoServer) EchoRequestMetadata(ctx context.Context, req *pb.EchoRequestMetadataRequest) (*pb.EchoRequestMetadataResponse, error) {
	resp := &pb.EchoRequestMetadataResponse{
		Metadata: make(map[string]*pb.MetadataValues),
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return resp, nil
	}

	// If specific keys requested, filter to those
	if len(req.Keys) > 0 {
		for _, key := range req.Keys {
			if values, exists := md[key]; exists {
				resp.Metadata[key] = &pb.MetadataValues{Values: values}
			}
		}
	} else {
		// Return all metadata
		for k, v := range md {
			resp.Metadata[k] = &pb.MetadataValues{Values: v}
		}
	}

	return resp, nil
}

func (s *EchoServer) EchoWithTrailers(ctx context.Context, req *pb.EchoWithTrailersRequest) (*pb.EchoResponse, error) {
	resp := &pb.EchoResponse{
		Message:  req.Message,
		Metadata: make(map[string]string),
	}

	// Echo back request metadata in response body
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range md {
			if len(v) > 0 {
				resp.Metadata[k] = v[0]
			}
		}
	}

	// Set specified trailers
	if len(req.Trailers) > 0 {
		trailerMD := metadata.New(nil)
		for k, v := range req.Trailers {
			trailerMD.Set(k, v)
		}
		_ = grpc.SetTrailer(ctx, trailerMD)
	}

	return resp, nil
}

func (s *EchoServer) EchoLargePayload(_ context.Context, req *pb.EchoLargePayloadRequest) (*pb.EchoLargePayloadResponse, error) {
	size := int(req.SizeBytes)
	if size <= 0 {
		size = 1
	}
	if size > MaxPayloadSize {
		return nil, status.Errorf(codes.InvalidArgument, "requested size %d exceeds maximum %d bytes", size, MaxPayloadSize)
	}

	pattern := req.Pattern
	if pattern == "" {
		pattern = "X"
	}

	// Generate payload by repeating pattern
	patternBytes := []byte(pattern)
	payload := bytes.Repeat(patternBytes, (size/len(patternBytes))+1)
	payload = payload[:size]

	return &pb.EchoLargePayloadResponse{
		Payload:    payload,
		ActualSize: int32(len(payload)),
	}, nil
}

func (s *EchoServer) EchoDeadline(ctx context.Context, req *pb.EchoDeadlineRequest) (*pb.EchoDeadlineResponse, error) {
	resp := &pb.EchoDeadlineResponse{
		Message:     req.Message,
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

	return resp, nil
}

func (s *EchoServer) EchoErrorWithDetails(_ context.Context, req *pb.EchoErrorWithDetailsRequest) (*pb.EchoResponse, error) {
	code := codes.Code(req.Code)
	if code > 16 {
		code = codes.Unknown
	}

	message := req.Message
	if message == "" {
		message = fmt.Sprintf("error with code %d", req.Code)
	}

	st := status.New(code, message)

	// Add rich error details
	for _, detail := range req.Details {
		var err error
		switch detail.Type {
		case "bad_request":
			br := &errdetails.BadRequest{}
			for _, fv := range detail.FieldViolations {
				br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
					Field:       fv.Field,
					Description: fv.Description,
				})
			}
			st, err = st.WithDetails(br)
		case "retry_info":
			ri := &errdetails.RetryInfo{
				RetryDelay: durationpb.New(time.Duration(detail.RetryDelayMs) * time.Millisecond),
			}
			st, err = st.WithDetails(ri)
		case "debug_info":
			di := &errdetails.DebugInfo{
				StackEntries: detail.StackEntries,
				Detail:       detail.DebugDetail,
			}
			st, err = st.WithDetails(di)
		case "quota_failure":
			qf := &errdetails.QuotaFailure{}
			for _, qv := range detail.QuotaViolations {
				qf.Violations = append(qf.Violations, &errdetails.QuotaFailure_Violation{
					Subject:     qv.Subject,
					Description: qv.Description,
				})
			}
			st, err = st.WithDetails(qf)
		}
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to attach error details: %v", err)
		}
	}

	return nil, st.Err()
}

func (s *EchoServer) ServerStream(req *pb.ServerStreamRequest, stream grpc.ServerStreamingServer[pb.EchoResponse]) error {
	ctx := stream.Context()
	md := make(map[string]string)

	if inMd, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range inMd {
			if len(v) > 0 {
				md[k] = v[0]
			}
		}
	}

	count := req.Count
	if count <= 0 {
		count = 1
	}

	interval := time.Duration(req.IntervalMs) * time.Millisecond

	for i := int32(0); i < count; i++ {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "stream canceled")
		default:
		}

		resp := &pb.EchoResponse{
			Message:  fmt.Sprintf("%s [%d/%d]", req.Message, i+1, count),
			Metadata: md,
		}

		if err := stream.Send(resp); err != nil {
			return err
		}

		if i < count-1 && interval > 0 {
			select {
			case <-time.After(interval):
			case <-ctx.Done():
				return status.Error(codes.Canceled, "stream canceled")
			}
		}
	}

	return nil
}

func (s *EchoServer) ClientStream(stream grpc.ClientStreamingServer[pb.EchoRequest, pb.EchoResponse]) error {
	ctx := stream.Context()
	md := make(map[string]string)

	if inMd, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range inMd {
			if len(v) > 0 {
				md[k] = v[0]
			}
		}
	}

	var messages []string

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		messages = append(messages, req.Message)
	}

	resp := &pb.EchoResponse{
		Message:  strings.Join(messages, ", "),
		Metadata: md,
	}

	return stream.SendAndClose(resp)
}

func (s *EchoServer) BidirectionalStream(stream grpc.BidiStreamingServer[pb.EchoRequest, pb.EchoResponse]) error {
	ctx := stream.Context()
	md := make(map[string]string)

	if inMd, ok := metadata.FromIncomingContext(ctx); ok {
		for k, v := range inMd {
			if len(v) > 0 {
				md[k] = v[0]
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return status.Error(codes.Canceled, "stream canceled")
		default:
		}

		req, err := stream.Recv()
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
