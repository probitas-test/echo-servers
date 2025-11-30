# echo-http

[![Build](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-http.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/build.echo-http.yml)
[![Docker](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-http.yml/badge.svg)](https://github.com/jsr-probitas/dockerfiles/actions/workflows/docker.echo-http.yml)

HTTP echo server for testing HTTP clients.

## Image

```
ghcr.io/jsr-probitas/echo-http:latest
```

## Quick Start

```bash
docker run -p 8080:80 ghcr.io/jsr-probitas/echo-http:latest
```

## Environment Variables

| Variable | Default   | Description  |
| -------- | --------- | ------------ |
| `HOST`   | `0.0.0.0` | Bind address |
| `PORT`   | `80`      | Listen port  |

```bash
# Custom port
docker run -p 3000:3000 -e PORT=3000 ghcr.io/jsr-probitas/echo-http:latest

# Using .env file
docker run -p 8080:8080 -v $(pwd)/.env:/app/.env ghcr.io/jsr-probitas/echo-http:latest
```

## API

| Endpoint           | Method | Description                               |
| ------------------ | ------ | ----------------------------------------- |
| `/get`             | GET    | Echo request info (query params, headers) |
| `/post`            | POST   | Echo request body (JSON, form data)       |
| `/put`             | PUT    | Echo request body                         |
| `/patch`           | PATCH  | Echo request body                         |
| `/delete`          | DELETE | Echo request info                         |
| `/headers`         | GET    | Echo headers only                         |
| `/status/{code}`   | GET    | Return specified status code (100-599)    |
| `/delay/{seconds}` | GET    | Echo after delay                          |
| `/health`          | GET    | Health check                              |

See [docs/api.md](./docs/api.md) for detailed API reference.

## Response Format

```json
{
  "method": "POST",
  "url": "/post?foo=bar",
  "args": { "foo": "bar" },
  "headers": { "Content-Type": "application/json" },
  "data": "raw body string",
  "json": { "key": "value" },
  "form": { "field": "value" }
}
```

## Examples

```bash
# GET with query parameters
curl "http://localhost:8080/get?name=test"

# POST with JSON
curl -X POST http://localhost:8080/post \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'

# Custom status code
curl http://localhost:8080/status/418

# Delayed response (for timeout testing)
curl http://localhost:8080/delay/5
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
