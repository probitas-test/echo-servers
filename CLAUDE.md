# Echo Servers

Echo servers for testing HTTP, gRPC, and GraphQL clients.

## Project Overview

- **Language**: Go 1.24
- **Container Registry**: GHCR (`ghcr.io/jsr-probitas/*`)
- **Build System**: just (modular justfile)
- **Development Environment**: Nix Flakes

## Project Structure

```
echo-servers/
├── flake.nix                 # Nix development environment
├── justfile                  # Root justfile (imports submodules)
├── compose.yaml              # Local orchestration
├── dprint.json               # Markdown/JSON/YAML formatter
├── echo-http/                # HTTP echo server
│   ├── Dockerfile
│   ├── justfile              # Package-specific commands
│   ├── .golangci.yml         # Linter config (v2 format)
│   ├── main.go
│   ├── handlers/             # HTTP handlers
│   └── docs/api.md           # API reference
├── echo-grpc/                # gRPC echo server
│   ├── Dockerfile
│   ├── justfile
│   ├── .golangci.yml
│   ├── main.go               # Contains //go:generate directive
│   ├── proto/                # Protobuf definitions
│   ├── server/               # gRPC server implementation
│   └── docs/api.md
└── echo-graphql/             # GraphQL echo server
    ├── Dockerfile
    ├── justfile
    ├── .golangci.yml
    ├── gqlgen.yml            # GraphQL code generator config
    ├── main.go
    ├── graph/                # GraphQL schema and resolvers
    │   ├── schema.graphqls
    │   ├── resolver.go       # Contains //go:generate directive
    │   ├── generated.go      # Generated (excluded from lint)
    │   └── schema.resolvers.go
    └── docs/api.md
```

## Development

### Setup

```bash
# Enter development environment
nix develop

# Or with direnv
direnv allow
```

### Commands

```bash
# List available commands
just

# Run all checks for all apps
just lint
just test
just build

# Per-app commands
just echo-http::lint
just echo-http::test
just echo-http::build
just echo-http::run

# Format code
just fmt           # Go + dprint (markdown/json/yaml)

# Code generation (run automatically by build)
# - echo-grpc: protoc (proto/*.pb.go)
# - echo-graphql: gqlgen (graph/generated.go, graph/models_gen.go)
```

### Local Testing

```bash
# Start all servers
docker compose up -d

# Test endpoints
curl http://localhost:18080/get
grpcurl -plaintext localhost:50051 list
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echo(message: \"hello\") }"}'

# Stop
docker compose down
```

## Code Patterns

### golangci-lint v2 Configuration

All `.golangci.yml` files use v2 format:

```yaml
version: "2"

linters:
  default: none
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - ineffassign
    - misspell
  exclusions:
    paths:
      - "generated/*.go" # Exclude generated files

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/jsr-probitas
```

### Code Generation

Generated files are excluded from linting. Generation runs via `go generate ./...`
in the build task:

- **echo-grpc**: `//go:generate protoc ...` in `main.go`
- **echo-graphql**: `//go:generate go run github.com/99designs/gqlgen generate` in
  `graph/resolver.go`

### Dockerfile Pattern

Multi-stage build with scratch base and OCI labels:

```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go generate ./...  # If needed
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o app .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/jsr-probitas/echo-servers"
LABEL org.opencontainers.image.description="Description of the server"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
```

## CI/CD

### Build Workflows (`build.echo-*.yml`)

Triggered on push/PR to respective app directories:

```
check (lint, fmt, git diff) → test → build
                              ↘     ↙
                            (parallel)
```

Uses Nix for reproducible environment:

```yaml
- uses: DeterminateSystems/nix-installer-action@main
- uses: DeterminateSystems/magic-nix-cache-action@main
- run: nix develop -c just echo-http::lint
```

### Docker Workflows (`docker.echo-*.yml`)

Triggered on push to main or release publish:

- Builds multi-arch images (linux/amd64, linux/arm64)
- Pushes to GHCR with tags: `latest`, branch name, version tag (on release)

---

## STRICT RULES (MUST FOLLOW)

### 1. Git Commit Restriction

**NEVER commit without explicit user permission.**

- Commits are forbidden by default
- Only perform a commit when the user explicitly grants permission
- After committing, recite this rule:
  > "Reminder: Commits are forbidden by default. I will not commit again unless
  > explicitly permitted."

### 2. Pre-Completion Verification

**BEFORE reporting task completion, run ALL of the following and ensure zero
errors:**

```bash
just lint   # includes dprint check
just test
just build
```

### 3. English for Version-Controlled Content

**Use English for ALL content tracked by Git:**

- Code (variable names, function names)
- Comments
- Documentation (README, CLAUDE.md, docs/*.md)
- Commit messages

### 4. Backup Before Destructive Operations

**ALWAYS create a backup before any operation that may lose working tree state.**

Examples requiring backup:

- `git restore`
- `git reset`
- `git checkout` (switching branches with uncommitted changes)
- Any file deletion or overwrite of uncommitted work

Use backup branch pattern:

```bash
git checkout -b "backup/$(git branch --show-current)/$(date +%Y%m%d-%H%M%S)"
git commit -am "WIP: before risky refactoring"
git checkout -
git cherry-pick --no-commit HEAD@{1}
```
