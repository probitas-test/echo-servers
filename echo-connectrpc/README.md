# echo-connectrpc

Connect RPC echo server for testing Connect RPC, gRPC, and gRPC-Web clients.

## Features

- **Multi-protocol support** - Single server supports Connect RPC, gRPC, and gRPC-Web
- **Protocol flexibility** - Each protocol can be individually enabled/disabled via environment variables
- **HTTP/1.1 and HTTP/2** - Full support for both HTTP versions
- **JSON and Protobuf** - Dual encoding support
- **Browser compatible** - Built-in gRPC-Web support for browser clients
- **Reflection API** - Full gRPC reflection support (v1 and v1alpha)
- **Health checks** - Standard gRPC health checking protocol
- **Streaming support** - Server, client, and bidirectional streaming

## Quick Start

### Using Docker

```bash
docker run -p 8080:8080 ghcr.io/probitas-test/echo-connectrpc:latest
```

### Using Docker Compose

```bash
docker compose up echo-connectrpc
```

### From Source

```bash
cd echo-connectrpc
just run
```

## Usage Examples

### Connect RPC (JSON)

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -d '{"message": "hello world"}'
```

### gRPC Protocol

```bash
grpcurl -plaintext -d '{"message": "hello"}' \
  localhost:8080 echo.v1.Echo/Echo
```

### Server Reflection

```bash
# List all services
grpcurl -plaintext localhost:8080 list

# Describe a service
grpcurl -plaintext localhost:8080 describe echo.v1.Echo
```

## Environment Variables

### Protocol Control

| Variable             | Default | Description                  |
| -------------------- | ------- | ---------------------------- |
| `HOST`               | 0.0.0.0 | Host address to bind         |
| `PORT`               | 8080    | Port number to listen on     |
| `DISABLE_CONNECTRPC` | false   | Disable Connect RPC protocol |
| `DISABLE_GRPC`       | false   | Disable gRPC protocol        |
| `DISABLE_GRPC_WEB`   | false   | Disable gRPC-Web protocol    |

### Reflection Control

| Variable                          | Default | Description                                                 |
| --------------------------------- | ------- | ----------------------------------------------------------- |
| `REFLECTION_INCLUDE_DEPENDENCIES` | false   | Include transitive dependencies (note: not fully supported) |
| `DISABLE_REFLECTION_V1`           | false   | Disable gRPC reflection v1 API                              |
| `DISABLE_REFLECTION_V1ALPHA`      | false   | Disable gRPC reflection v1alpha API                         |

**Note:** At least one protocol must be enabled. The server will refuse to start if all protocols are disabled.

### Examples

```bash
# Run with only Connect RPC protocol
DISABLE_GRPC=true DISABLE_GRPC_WEB=true ./echo-connectrpc

# Run with gRPC protocol disabled
DISABLE_GRPC=true ./echo-connectrpc

# Disable reflection v1alpha for compatibility testing
DISABLE_REFLECTION_V1ALPHA=true ./echo-connectrpc
```

## Protocol Comparison with echo-grpc

| Feature                  | echo-grpc     | echo-connectrpc          |
| ------------------------ | ------------- | ------------------------ |
| **gRPC Protocol**        | ✅            | ✅ (can be disabled)     |
| **Connect RPC Protocol** | ❌            | ✅ (can be disabled)     |
| **gRPC-Web Protocol**    | ❌            | ✅ (can be disabled)     |
| **HTTP/1.1 Support**     | ❌            | ✅                       |
| **HTTP/2 Support**       | ✅            | ✅                       |
| **JSON Encoding**        | ❌            | ✅ (Connect RPC)         |
| **Protobuf Encoding**    | ✅            | ✅ (all protocols)       |
| **Browser Support**      | ❌            | ✅ (gRPC-Web built-in)   |
| **Reflection v1**        | ✅ (optional) | ✅ (optional)            |
| **Reflection v1alpha**   | ✅ (optional) | ✅ (optional)            |
| **Custom Reflection**    | ✅            | ❌ (uses grpcreflect)    |
| **Dependency Control**   | ✅            | ⚠️ (limited support)      |
| **API Compatibility**    | -             | 100% (same .proto files) |

## API Documentation

See [docs/api.md](./docs/api.md) for complete API reference.

## Development

### Prerequisites

- Go 1.24+
- protoc
- protoc-gen-go
- protoc-gen-connect-go

Or use Nix:

```bash
nix develop
```

### Commands

```bash
# List available commands
just

# Run linter
just lint

# Run tests
just test

# Build binary
just build

# Run server
just run

# Format code
just fmt

# Clean artifacts
just clean
```

### Code Generation

Protobuf code is generated automatically during build:

```bash
just generate
```

This generates:

- `proto/*.pb.go` - Protobuf message definitions
- `proto/*.connect.go` - Connect RPC service stubs

## License

MIT
