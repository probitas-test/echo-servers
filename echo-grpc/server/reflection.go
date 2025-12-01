package server

import (
	"context"
	"io"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// RegisterReflection registers the reflection service. When includeDeps is
// false (default), the reflection response will omit transitive dependencies,
// forcing clients to resolve imports themselves. When true, it falls back to
// the standard gRPC reflection implementation.
func RegisterReflection(s *grpc.Server, includeDeps bool) {
	if includeDeps {
		reflection.Register(s)
		return
	}

	svr := newReflectionServer(s, includeDeps)
	reflectionv1.RegisterServerReflectionServer(s, svr)
	reflectionv1alpha.RegisterServerReflectionServer(s, &v1AlphaAdapter{svr: svr})
}

type reflectionServer struct {
	reflectionv1.UnimplementedServerReflectionServer
	includeDeps bool
	services    map[string]grpc.ServiceInfo
	desc        protodesc.Resolver
	ext         extensionResolver
}

type extensionResolver interface {
	protoregistry.ExtensionTypeResolver
	RangeExtensionsByMessage(message protoreflect.FullName, f func(protoreflect.ExtensionType) bool)
}

func newReflectionServer(s *grpc.Server, includeDeps bool) *reflectionServer {
	return &reflectionServer{
		includeDeps: includeDeps,
		services:    s.GetServiceInfo(),
		desc:        protoregistry.GlobalFiles,
		ext:         protoregistry.GlobalTypes,
	}
}

func (s *reflectionServer) ServerReflectionInfo(stream reflectionv1.ServerReflection_ServerReflectionInfoServer) error {
	sent := make(map[string]bool)

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		out := &reflectionv1.ServerReflectionResponse{
			ValidHost:       in.Host,
			OriginalRequest: in,
		}

		switch req := in.MessageRequest.(type) {
		case *reflectionv1.ServerReflectionRequest_FileByFilename:
			var b [][]byte
			fd, err := s.desc.FindFileByPath(req.FileByFilename)
			if err == nil {
				b, err = s.fileDescWithDependencies(fd, sent)
			}
			s.writeFileDescriptorResponse(out, b, err)
		case *reflectionv1.ServerReflectionRequest_FileContainingSymbol:
			b, err := s.fileDescEncodingContainingSymbol(req.FileContainingSymbol, sent)
			s.writeFileDescriptorResponse(out, b, err)
		case *reflectionv1.ServerReflectionRequest_FileContainingExtension:
			typeName := req.FileContainingExtension.ContainingType
			extNum := req.FileContainingExtension.ExtensionNumber
			b, err := s.fileDescEncodingContainingExtension(typeName, extNum, sent)
			s.writeFileDescriptorResponse(out, b, err)
		case *reflectionv1.ServerReflectionRequest_AllExtensionNumbersOfType:
			extNums, err := s.allExtensionNumbersForTypeName(req.AllExtensionNumbersOfType)
			if err != nil {
				out.MessageResponse = &reflectionv1.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &reflectionv1.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &reflectionv1.ServerReflectionResponse_AllExtensionNumbersResponse{
					AllExtensionNumbersResponse: &reflectionv1.ExtensionNumberResponse{
						BaseTypeName:    req.AllExtensionNumbersOfType,
						ExtensionNumber: extNums,
					},
				}
			}
		case *reflectionv1.ServerReflectionRequest_ListServices:
			out.MessageResponse = &reflectionv1.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &reflectionv1.ListServiceResponse{
					Service: s.listServices(),
				},
			}
		default:
			return status.Errorf(codes.InvalidArgument, "invalid MessageRequest: %v", in.MessageRequest)
		}

		if err := stream.Send(out); err != nil {
			return err
		}
	}
}

func (s *reflectionServer) writeFileDescriptorResponse(out *reflectionv1.ServerReflectionResponse, b [][]byte, err error) {
	if err != nil {
		out.MessageResponse = &reflectionv1.ServerReflectionResponse_ErrorResponse{
			ErrorResponse: &reflectionv1.ErrorResponse{
				ErrorCode:    int32(codes.NotFound),
				ErrorMessage: err.Error(),
			},
		}
		return
	}

	out.MessageResponse = &reflectionv1.ServerReflectionResponse_FileDescriptorResponse{
		FileDescriptorResponse: &reflectionv1.FileDescriptorResponse{
			FileDescriptorProto: b,
		},
	}
}

func (s *reflectionServer) fileDescWithDependencies(fd protoreflect.FileDescriptor, sent map[string]bool) ([][]byte, error) {
	if fd.IsPlaceholder() {
		return nil, protoregistry.NotFound
	}

	var result [][]byte
	queue := []protoreflect.FileDescriptor{fd}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current.IsPlaceholder() {
			continue
		}

		if sent[current.Path()] {
			continue
		}

		sent[current.Path()] = true

		fdProto := protodesc.ToFileDescriptorProto(current)
		encoded, err := proto.Marshal(fdProto)
		if err != nil {
			return nil, err
		}
		result = append(result, encoded)

		if s.includeDeps {
			for i := 0; i < current.Imports().Len(); i++ {
				queue = append(queue, current.Imports().Get(i))
			}
		}
	}

	return result, nil
}

func (s *reflectionServer) fileDescEncodingContainingSymbol(name string, sent map[string]bool) ([][]byte, error) {
	d, err := s.desc.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	return s.fileDescWithDependencies(d.ParentFile(), sent)
}

func (s *reflectionServer) fileDescEncodingContainingExtension(typeName string, extNum int32, sent map[string]bool) ([][]byte, error) {
	xt, err := s.ext.FindExtensionByNumber(protoreflect.FullName(typeName), protoreflect.FieldNumber(extNum))
	if err != nil {
		return nil, err
	}
	return s.fileDescWithDependencies(xt.TypeDescriptor().ParentFile(), sent)
}

func (s *reflectionServer) allExtensionNumbersForTypeName(name string) ([]int32, error) {
	var numbers []int32
	s.ext.RangeExtensionsByMessage(protoreflect.FullName(name), func(xt protoreflect.ExtensionType) bool {
		numbers = append(numbers, int32(xt.TypeDescriptor().Number()))
		return true
	})
	sort.Slice(numbers, func(i, j int) bool {
		return numbers[i] < numbers[j]
	})
	if len(numbers) == 0 {
		if _, err := s.desc.FindDescriptorByName(protoreflect.FullName(name)); err != nil {
			return nil, err
		}
	}
	return numbers, nil
}

func (s *reflectionServer) listServices() []*reflectionv1.ServiceResponse {
	resp := make([]*reflectionv1.ServiceResponse, 0, len(s.services))
	for name := range s.services {
		resp = append(resp, &reflectionv1.ServiceResponse{Name: name})
	}
	sort.Slice(resp, func(i, j int) bool {
		return resp[i].Name < resp[j].Name
	})
	return resp
}

type v1AlphaAdapter struct {
	svr reflectionv1.ServerReflectionServer
}

func (s *v1AlphaAdapter) ServerReflectionInfo(stream reflectionv1alpha.ServerReflection_ServerReflectionInfoServer) error {
	return s.svr.ServerReflectionInfo(&v1AlphaStreamAdapter{stream: stream})
}

type v1AlphaStreamAdapter struct {
	stream reflectionv1alpha.ServerReflection_ServerReflectionInfoServer
}

func (s *v1AlphaStreamAdapter) Send(resp *reflectionv1.ServerReflectionResponse) error {
	return s.stream.Send(toV1AlphaResponse(resp))
}

func (s *v1AlphaStreamAdapter) Recv() (*reflectionv1.ServerReflectionRequest, error) {
	resp, err := s.stream.Recv()
	if err != nil {
		return nil, err
	}
	return toV1Request(resp), nil
}

func (s *v1AlphaStreamAdapter) Context() context.Context {
	return s.stream.Context()
}

func (s *v1AlphaStreamAdapter) SetHeader(md metadata.MD) error {
	return s.stream.SetHeader(md)
}

func (s *v1AlphaStreamAdapter) SendHeader(md metadata.MD) error {
	return s.stream.SendHeader(md)
}

func (s *v1AlphaStreamAdapter) SetTrailer(md metadata.MD) {
	s.stream.SetTrailer(md)
}

func (s *v1AlphaStreamAdapter) SendMsg(m interface{}) error {
	return s.stream.SendMsg(m)
}

func (s *v1AlphaStreamAdapter) RecvMsg(m interface{}) error {
	return s.stream.RecvMsg(m)
}

// Converters between v1alpha and v1 messages.
// nolint:staticcheck // v1alpha reflection is kept for backward compatibility with older clients.
func toV1Request(v1alpha *reflectionv1alpha.ServerReflectionRequest) *reflectionv1.ServerReflectionRequest {
	var v1 reflectionv1.ServerReflectionRequest
	v1.Host = v1alpha.Host
	switch mr := v1alpha.MessageRequest.(type) {
	case *reflectionv1alpha.ServerReflectionRequest_FileByFilename:
		v1.MessageRequest = &reflectionv1.ServerReflectionRequest_FileByFilename{
			FileByFilename: mr.FileByFilename,
		}
	case *reflectionv1alpha.ServerReflectionRequest_FileContainingSymbol:
		v1.MessageRequest = &reflectionv1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: mr.FileContainingSymbol,
		}
	case *reflectionv1alpha.ServerReflectionRequest_FileContainingExtension:
		if mr.FileContainingExtension != nil {
			v1.MessageRequest = &reflectionv1.ServerReflectionRequest_FileContainingExtension{
				FileContainingExtension: &reflectionv1.ExtensionRequest{
					ContainingType:  mr.FileContainingExtension.GetContainingType(),
					ExtensionNumber: mr.FileContainingExtension.GetExtensionNumber(),
				},
			}
		}
	case *reflectionv1alpha.ServerReflectionRequest_AllExtensionNumbersOfType:
		v1.MessageRequest = &reflectionv1.ServerReflectionRequest_AllExtensionNumbersOfType{
			AllExtensionNumbersOfType: mr.AllExtensionNumbersOfType,
		}
	case *reflectionv1alpha.ServerReflectionRequest_ListServices:
		v1.MessageRequest = &reflectionv1.ServerReflectionRequest_ListServices{
			ListServices: mr.ListServices,
		}
	}
	return &v1
}

// nolint:staticcheck // v1alpha reflection is kept for backward compatibility with older clients.
func toV1AlphaResponse(v1 *reflectionv1.ServerReflectionResponse) *reflectionv1alpha.ServerReflectionResponse {
	var v1alpha reflectionv1alpha.ServerReflectionResponse
	v1alpha.ValidHost = v1.ValidHost
	if v1.OriginalRequest != nil {
		v1alpha.OriginalRequest = toV1AlphaRequest(v1.OriginalRequest)
	}
	switch mr := v1.MessageResponse.(type) {
	case *reflectionv1.ServerReflectionResponse_FileDescriptorResponse:
		if mr != nil {
			v1alpha.MessageResponse = &reflectionv1alpha.ServerReflectionResponse_FileDescriptorResponse{
				FileDescriptorResponse: &reflectionv1alpha.FileDescriptorResponse{
					FileDescriptorProto: mr.FileDescriptorResponse.GetFileDescriptorProto(),
				},
			}
		}
	case *reflectionv1.ServerReflectionResponse_AllExtensionNumbersResponse:
		if mr != nil {
			v1alpha.MessageResponse = &reflectionv1alpha.ServerReflectionResponse_AllExtensionNumbersResponse{
				AllExtensionNumbersResponse: &reflectionv1alpha.ExtensionNumberResponse{
					BaseTypeName:    mr.AllExtensionNumbersResponse.GetBaseTypeName(),
					ExtensionNumber: mr.AllExtensionNumbersResponse.GetExtensionNumber(),
				},
			}
		}
	case *reflectionv1.ServerReflectionResponse_ListServicesResponse:
		if mr != nil {
			svcs := make([]*reflectionv1alpha.ServiceResponse, len(mr.ListServicesResponse.GetService()))
			for i, svc := range mr.ListServicesResponse.GetService() {
				svcs[i] = &reflectionv1alpha.ServiceResponse{Name: svc.GetName()}
			}
			v1alpha.MessageResponse = &reflectionv1alpha.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &reflectionv1alpha.ListServiceResponse{
					Service: svcs,
				},
			}
		}
	case *reflectionv1.ServerReflectionResponse_ErrorResponse:
		if mr != nil {
			v1alpha.MessageResponse = &reflectionv1alpha.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &reflectionv1alpha.ErrorResponse{
					ErrorCode:    mr.ErrorResponse.GetErrorCode(),
					ErrorMessage: mr.ErrorResponse.GetErrorMessage(),
				},
			}
		}
	}
	return &v1alpha
}

// nolint:staticcheck // v1alpha reflection is kept for backward compatibility with older clients.
func toV1AlphaRequest(v1 *reflectionv1.ServerReflectionRequest) *reflectionv1alpha.ServerReflectionRequest {
	var v1alpha reflectionv1alpha.ServerReflectionRequest
	v1alpha.Host = v1.Host
	switch mr := v1.MessageRequest.(type) {
	case *reflectionv1.ServerReflectionRequest_FileByFilename:
		if mr != nil {
			v1alpha.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_FileByFilename{
				FileByFilename: mr.FileByFilename,
			}
		}
	case *reflectionv1.ServerReflectionRequest_FileContainingSymbol:
		if mr != nil {
			v1alpha.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: mr.FileContainingSymbol,
			}
		}
	case *reflectionv1.ServerReflectionRequest_FileContainingExtension:
		if mr != nil {
			v1alpha.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_FileContainingExtension{
				FileContainingExtension: &reflectionv1alpha.ExtensionRequest{
					ContainingType:  mr.FileContainingExtension.GetContainingType(),
					ExtensionNumber: mr.FileContainingExtension.GetExtensionNumber(),
				},
			}
		}
	case *reflectionv1.ServerReflectionRequest_AllExtensionNumbersOfType:
		if mr != nil {
			v1alpha.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_AllExtensionNumbersOfType{
				AllExtensionNumbersOfType: mr.AllExtensionNumbersOfType,
			}
		}
	case *reflectionv1.ServerReflectionRequest_ListServices:
		if mr != nil {
			v1alpha.MessageRequest = &reflectionv1alpha.ServerReflectionRequest_ListServices{
				ListServices: mr.ListServices,
			}
		}
	}
	return &v1alpha
}
