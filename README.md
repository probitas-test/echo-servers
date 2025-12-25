# Echo Servers

[![Build echo-http](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-http.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-http.yml)
[![Build echo-grpc](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-grpc.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-grpc.yml)
[![Build echo-graphql](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-graphql.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-graphql.yml)
[![Build echo-connectrpc](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-connectrpc.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/build.echo-connectrpc.yml)

Echo servers for testing HTTP, gRPC, GraphQL, and Connect RPC clients. Built for
testing [Probitas](https://github.com/probitas-test/probitas) and other client
implementations.

## Images

| Image                                   | Protocol                      | Default Port | Status                                                                                                                                                                                                        |
| --------------------------------------- | ----------------------------- | ------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ghcr.io/probitas-test/echo-http`       | HTTP                          | 80           | [![Docker](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-http.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-http.yml)             |
| `ghcr.io/probitas-test/echo-grpc`       | gRPC                          | 50051        | [![Docker](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-grpc.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-grpc.yml)             |
| `ghcr.io/probitas-test/echo-graphql`    | GraphQL                       | 8080         | [![Docker](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-graphql.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-graphql.yml)       |
| `ghcr.io/probitas-test/echo-connectrpc` | Connect RPC / gRPC / gRPC-Web | 8080         | [![Docker](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-connectrpc.yml/badge.svg)](https://github.com/probitas-test/echo-servers/actions/workflows/docker.echo-connectrpc.yml) |

## Quick Start

```bash
# Start all servers
docker compose up -d

# Test HTTP
curl "http://localhost:18080/get?hello=world"

# Test gRPC
grpcurl -plaintext -d '{"message":"hello"}' localhost:50051 echo.v1.Echo/Echo

# Test GraphQL
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query":"{ echo(message: \"hello\") }"}'

# Test Connect RPC (JSON)
curl -X POST http://localhost:18081/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -d '{"message":"hello"}'

# Test Connect RPC (gRPC protocol)
grpcurl -plaintext -d '{"message":"hello"}' localhost:18081 echo.v1.Echo/Echo

# Stop all servers
docker compose down
```

## Environment Variables

All servers support the following environment variables:

| Variable | Description                             |
| -------- | --------------------------------------- |
| `HOST`   | Bind address (default: `0.0.0.0`)       |
| `PORT`   | Listen port (default: varies by server) |

Servers also support `.env` file for configuration.

## Features

All servers are designed for testing purposes:

- **No rate limits** - Test high-throughput scenarios
- **Configurable delays** - Test timeout handling
- **Error injection** - Test error handling
- **Streaming support** - Test streaming clients (gRPC, GraphQL subscriptions)
- **Minimal images** - Built on scratch, ~10-20MB each

## Documentation

- [echo-http](./echo-http/README.md) - HTTP echo server
- [echo-grpc](./echo-grpc/README.md) - gRPC echo server
- [echo-graphql](./echo-graphql/README.md) - GraphQL echo server
- [echo-connectrpc](./echo-connectrpc/README.md) - Connect RPC echo server (supports Connect RPC, gRPC, and gRPC-Web)

## Development

### Prerequisites

Uses [Nix Flakes](https://nixos.wiki/wiki/Flakes) for development environment
management.

```bash
# Enter development environment
nix develop

# Or use direnv for automatic activation
echo "use flake" > .envrc
direnv allow
```

### Commands

Requires [just](https://github.com/casey/just) command runner (included in Nix
environment).

```bash
# Show available commands
just

# Run linter on all packages
just lint

# Run tests on all packages
just test

# Build all packages
just build

# Format all code (Go + Markdown/JSON/YAML)
just fmt

# Build and run Docker images
docker compose up --build
```
