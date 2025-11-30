# Probitas Test Servers

[![Build echo-http](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-http.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-http.yml)
[![Build echo-grpc](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-grpc.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-grpc.yml)
[![Build echo-graphql](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-graphql.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-graphql.yml)

Docker images for testing [Probitas](https://github.com/jsr-probitas/probitas)
clients.

## Images

| Image                               | Protocol | Default Port | Status                                                                                                                                                                                              |
| ----------------------------------- | -------- | ------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ghcr.io/jsr-probitas/echo-http`    | HTTP     | 80           | [![Docker](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-http.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-http.yml)       |
| `ghcr.io/jsr-probitas/echo-grpc`    | gRPC     | 50051        | [![Docker](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-grpc.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-grpc.yml)       |
| `ghcr.io/jsr-probitas/echo-graphql` | GraphQL  | 8080         | [![Docker](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-graphql.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-graphql.yml) |

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

- [echo-http](./echo-http/README.md)
- [echo-grpc](./echo-grpc/README.md)
- [echo-graphql](./echo-graphql/README.md)

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
