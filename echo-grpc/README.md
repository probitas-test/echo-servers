# echo-grpc

[![Build](https://github.com/jsr-probitas/echo-servers/actions/workflows/build.echo-grpc.yml/badge.svg)](https://github.com/jsr-probitas/echo-servers/actions/workflows/build.echo-grpc.yml)
[![Docker](https://github.com/jsr-probitas/echo-servers/actions/workflows/docker.echo-grpc.yml/badge.svg)](https://github.com/jsr-probitas/echo-servers/actions/workflows/docker.echo-grpc.yml)

gRPC echo server for testing gRPC clients.

## Image

```
ghcr.io/jsr-probitas/echo-grpc:latest
```

## Quick Start

```bash
docker run -p 50051:50051 ghcr.io/jsr-probitas/echo-grpc:latest
```

## Environment Variables

- `HOST` (default `0.0.0.0`): Bind address
- `PORT` (default `50051`): Listen port
- `REFLECTION_INCLUDE_DEPENDENCIES` (default `false`): If `true`, server reflection returns transitive proto dependencies (standard gRPC behavior). Default `false` returns only the containing file to reproduce missing-import scenarios.
- `DISABLE_REFLECTION_V1` (default `false`): Disable gRPC reflection v1 API
- `DISABLE_REFLECTION_V1ALPHA` (default `false`): Disable gRPC reflection v1alpha API

```bash
# Custom port
docker run -p 9000:9000 -e PORT=9000 ghcr.io/jsr-probitas/echo-grpc:latest

# Using .env file
docker run -p 50051:50051 -v $(pwd)/.env:/app/.env ghcr.io/jsr-probitas/echo-grpc:latest

# Disable v1alpha reflection (v1 only)
docker run -p 50051:50051 -e DISABLE_REFLECTION_V1ALPHA=true ghcr.io/jsr-probitas/echo-grpc:latest

# Disable v1 reflection (v1alpha only)
docker run -p 50051:50051 -e DISABLE_REFLECTION_V1=true ghcr.io/jsr-probitas/echo-grpc:latest
```

## API

```protobuf
service Echo {
  // Unary RPCs
  rpc Echo (EchoRequest) returns (EchoResponse);
  rpc EchoWithDelay (EchoWithDelayRequest) returns (EchoResponse);
  rpc EchoError (EchoErrorRequest) returns (EchoResponse);

  // Streaming RPCs
  rpc ServerStream (ServerStreamRequest) returns (stream EchoResponse);
  rpc ClientStream (stream EchoRequest) returns (EchoResponse);
  rpc BidirectionalStream (stream EchoRequest) returns (stream EchoResponse);
}
```

See [docs/api.md](./docs/api.md) for detailed API reference.

## Features

| Feature                 | Description                                      |
| ----------------------- | ------------------------------------------------ |
| Unary RPC               | `Echo`, `EchoWithDelay`, `EchoError`             |
| Server Streaming        | Send N responses with configurable interval      |
| Client Streaming        | Aggregate multiple requests into single response |
| Bidirectional Streaming | Echo each message back immediately               |
| Metadata Echo           | Request metadata included in response            |
| Server Reflection       | v1 and v1alpha supported                         |
| Error Responses         | Return any gRPC status code (0-16)               |

## Examples

```bash
# List services (requires grpcurl)
grpcurl -plaintext localhost:50051 list

# Simple echo
grpcurl -plaintext -d '{"message": "hello"}' \
  localhost:50051 echo.v1.Echo/Echo

# Echo with delay (for timeout testing)
grpcurl -plaintext -d '{"message": "hello", "delay_ms": 5000}' \
  localhost:50051 echo.v1.Echo/EchoWithDelay

# Return specific error code
grpcurl -plaintext -d '{"message": "test", "code": 5, "details": "not found"}' \
  localhost:50051 echo.v1.Echo/EchoError

# Server streaming
grpcurl -plaintext -d '{"message": "hello", "count": 5, "interval_ms": 1000}' \
  localhost:50051 echo.v1.Echo/ServerStream
```

## Development

### Prerequisites

```bash
# Enter development environment with Nix (from repository root)
nix develop
```

### Commands

```bash
# Run linter, tests, and build
just

# Run linter
just lint

# Run tests
just test

# Build binary
just build

# Run locally
just run

# Format code
just fmt
```
