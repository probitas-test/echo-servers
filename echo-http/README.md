# echo-http

[![Build](https://github.com/jsr-probitas/echo-servers/actions/workflows/build.echo-http.yml/badge.svg)](https://github.com/jsr-probitas/echo-servers/actions/workflows/build.echo-http.yml)
[![Docker](https://github.com/jsr-probitas/echo-servers/actions/workflows/docker.echo-http.yml/badge.svg)](https://github.com/jsr-probitas/echo-servers/actions/workflows/docker.echo-http.yml)

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

### Echo Endpoints

| Endpoint      | Method | Description                               |
| ------------- | ------ | ----------------------------------------- |
| `/get`        | GET    | Echo request info (query params, headers) |
| `/post`       | POST   | Echo request body (JSON, form data)       |
| `/put`        | PUT    | Echo request body                         |
| `/patch`      | PATCH  | Echo request body                         |
| `/delete`     | DELETE | Echo request info                         |
| `/anything`   | ANY    | Echo any request (method, headers, body)  |
| `/anything/*` | ANY    | Echo any request with path                |

### Utility Endpoints

| Endpoint           | Method | Description                            |
| ------------------ | ------ | -------------------------------------- |
| `/headers`         | GET    | Echo headers only                      |
| `/ip`              | GET    | Return client IP address               |
| `/user-agent`      | GET    | Return User-Agent header               |
| `/status/{code}`   | ANY    | Return specified status code (100-599) |
| `/delay/{seconds}` | GET    | Echo after delay (max 30s)             |
| `/health`          | GET    | Health check                           |

### Redirect Endpoints

| Endpoint                 | Method | Description                             |
| ------------------------ | ------ | --------------------------------------- |
| `/redirect/{n}`          | GET    | Redirect n times before final response  |
| `/redirect-to`           | GET    | Redirect to URL (?url=...&status_code=) |
| `/absolute-redirect/{n}` | GET    | Redirect n times with absolute URLs     |
| `/relative-redirect/{n}` | GET    | Redirect n times with relative URLs     |

### Authentication Endpoints

| Endpoint                           | Method | Description                              |
| ---------------------------------- | ------ | ---------------------------------------- |
| `/basic-auth/{user}/{pass}`        | GET    | Basic auth (200 if match, 401 otherwise) |
| `/hidden-basic-auth/{user}/{pass}` | GET    | Basic auth (200 if match, 404 otherwise) |
| `/bearer`                          | GET    | Bearer token validation                  |

### Cookie Endpoints

| Endpoint          | Method | Description                            |
| ----------------- | ------ | -------------------------------------- |
| `/cookies`        | GET    | Echo request cookies                   |
| `/cookies/set`    | GET    | Set cookies (?name=value) and redirect |
| `/cookies/delete` | GET    | Delete cookies (?name) and redirect    |

### Binary Data Endpoints

| Endpoint      | Method | Description                             |
| ------------- | ------ | --------------------------------------- |
| `/bytes/{n}`  | GET    | Return n random bytes (max 100KB)       |
| `/stream/{n}` | GET    | Stream n JSON lines (max 100)           |
| `/drip`       | GET    | Drip data (?duration=&numbytes=&delay=) |

### Compression Endpoints

| Endpoint   | Method | Description                        |
| ---------- | ------ | ---------------------------------- |
| `/gzip`    | GET    | Return gzip-compressed response    |
| `/deflate` | GET    | Return deflate-compressed response |
| `/brotli`  | GET    | Return brotli-compressed response  |

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

# Redirect testing
curl -L http://localhost:8080/redirect/3

# Basic authentication
curl -u user:pass http://localhost:8080/basic-auth/user/pass

# Bearer token
curl -H "Authorization: Bearer my-token" http://localhost:8080/bearer

# Cookie handling
curl -c cookies.txt http://localhost:8080/cookies/set?session=abc123
curl -b cookies.txt http://localhost:8080/cookies

# Get client IP
curl http://localhost:8080/ip

# Compression testing
curl --compressed http://localhost:8080/gzip
curl --compressed http://localhost:8080/brotli

# Stream data
curl http://localhost:8080/stream/5

# Random bytes
curl http://localhost:8080/bytes/100 --output random.bin
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
