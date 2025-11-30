package server

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/jsr-probitas/dockerfiles/echo-grpc/proto"
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
