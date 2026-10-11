# Echo Servers

Echo servers for testing HTTP, gRPC, GraphQL, and Connect RPC clients. Built for
testing [Probitas](https://github.com/probitas-test/probitas) and other client
implementations.

## How it is built

- **Test first.** Red → green → refactor. Write the failing test before the
  code that makes it pass.
- **Each layer holds one thing.** Put a fact in the one layer that owns it, and
  nowhere else.

  | Layer                | Holds                                                              |
  | -------------------- | ------------------------------------------------------------------ |
  | `docs/adr/`          | one decision and why it was made                                   |
  | `docs/spec/`         | the behaviour users and other components can rely on               |
  | `echo-*/docs/api.md` | each server's API reference                                        |
  | Tests                | the details that pin the behaviour down: rules, boundaries, tables |
  | Code                 | how it is done                                                     |
  | Commits              | why a change was made                                              |
  | Comments             | why it is not done another way                                     |

  Before writing, fixing or reviewing a record, read `docs/adr/README.md`;
  before writing, fixing or reviewing a spec, read `docs/spec/README.md`.
  Follow their writing order.
- **Dependencies are current.** Before adding or bumping one, look up its
  latest release in the registry. Do not copy a version from memory or from
  another repository.

## Quality gate

Run `just ci` before reporting a change as done. It runs each server's
`lint`, `fmt-check`, `test` and `build`, then `just fmt-check`, which checks
the formatting that spans the repository (Markdown, JSON and YAML with dprint,
Nix with nixfmt). CI runs the same recipes: the per-server build workflows run
`just <server>::ci`, and the Format workflow runs `just fmt-check`.

## Layout

| Path               | What lives there                                     |
| ------------------ | ---------------------------------------------------- |
| `echo-http/`       | HTTP echo server                                     |
| `echo-grpc/`       | gRPC echo server                                     |
| `echo-graphql/`    | GraphQL echo server                                  |
| `echo-connectrpc/` | Connect RPC echo server (Connect, gRPC and gRPC-Web) |
| `docs/`            | ADRs and specs                                       |
| `.github/`         | per-server build and Docker workflows                |
| `.claude/rules/`   | rules that make the ADR and spec procedures get read |

## Language

- README, docs and commit messages: English
- Code comments, log and error messages: English

## Commits

[Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/), as
described in `.gitmessage` (`git config commit.template .gitmessage`):

```
<type>[optional scope]: <description>
```

- Types: `feat`, `fix`, `deps`, `refactor`, `test`, `docs`, `style`, `chore`.
- The scope names the server the change is in (`feat(echo-http): …`); leave
  it out for a change across the repository (`docs: …`).
- A breaking change carries `!` after the type or scope
  (`refactor(echo-http)!: …`).

## Project Overview

- **Language**: Go 1.25
- **Container Registry**: GHCR (`ghcr.io/probitas-test/*`)
- **Build System**: just (modular justfile)
- **Development Environment**: Nix Flakes

## Project Structure

```
echo-servers/
├── AGENTS.md                 # This guide
├── flake.nix                 # Nix development environment
├── justfile                  # Root justfile (imports submodules)
├── compose.yaml              # Local orchestration
├── dprint.json               # Markdown/JSON/YAML formatter
├── docs/                     # ADRs and specs
├── echo-http/                # HTTP echo server
│   ├── Dockerfile
│   ├── justfile              # Package-specific commands
│   ├── .golangci.yml         # Linter config (v2 format)
│   ├── main.go
│   ├── handlers/             # HTTP handlers
│   └── docs/api.md           # API reference
├── echo-grpc/                # gRPC echo server
│   ├── Dockerfile
│   ├── justfile              # `generate` recipe runs protoc
│   ├── .golangci.yml
│   ├── main.go
│   ├── proto/                # Protobuf definitions
│   ├── server/               # gRPC server implementation
│   └── docs/api.md
├── echo-graphql/             # GraphQL echo server
│   ├── Dockerfile
│   ├── justfile
│   ├── .golangci.yml
│   ├── gqlgen.yml            # GraphQL code generator config
│   ├── main.go
│   ├── graph/                # GraphQL schema and resolvers
│   │   ├── schema.graphqls
│   │   ├── resolver.go       # Contains //go:generate directive
│   │   ├── generated.go      # Generated (excluded from lint)
│   │   └── schema.resolvers.go
│   └── docs/api.md
└── echo-connectrpc/          # Connect RPC echo server
    ├── Dockerfile
    ├── justfile              # `generate` recipe runs protoc
    ├── .golangci.yml
    ├── main.go
    ├── config.go             # Environment variable configuration
    ├── proto/                # Protobuf definitions (shared with echo-grpc)
    ├── server/               # Connect RPC server implementation
    └── docs/api.md
```

## Development

### Setup

```bash
# Enter development environment
nix develop

# Or with direnv
cp .envrc.sample .envrc
direnv allow
```

### Commands

```bash
# List available commands
just

# The quality gate: every server's checks plus the repository-wide formatting
just ci

# Run one kind of check for all apps
just lint          # golangci-lint for every server
just test
just build

# Per-app commands
just echo-http::ci
just echo-http::lint
just echo-http::test
just echo-http::build
just echo-http::run

# Format code
just fmt           # Go + dprint (markdown/json/yaml) + nixfmt
just fmt-check     # dprint and nixfmt checks across the repository

# Code generation (run automatically by build)
# - echo-grpc: protoc (proto/*.pb.go)
# - echo-graphql: gqlgen (graph/generated.go, graph/models_gen.go)
# - echo-connectrpc: protoc with protoc-gen-connect-go (proto/protoconnect/*.connect.go)
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
curl -X POST http://localhost:18081/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'

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
      - proto/*.pb.go # Each server lists its own generated files

formatters:
  enable:
    - gofmt
    - goimports
  settings:
    goimports:
      local-prefixes:
        - github.com/probitas-test
```

### Code Generation

Generated files are committed and excluded from linting. Generation runs in
each server's `build` recipe:

- **echo-grpc**: the private `generate` recipe in `echo-grpc/justfile` runs
  `protoc` with `protoc-gen-go` and `protoc-gen-go-grpc`, then `goimports`
  so the output matches `fmt-check`
- **echo-connectrpc**: the private `generate` recipe in
  `echo-connectrpc/justfile` runs `protoc` with `protoc-gen-go` and
  `protoc-gen-connect-go`, then `goimports`
- **echo-graphql**: `go generate ./...` runs
  `//go:generate go run github.com/99designs/gqlgen generate` in
  `graph/resolver.go`

In CI, each of these servers' `ci` recipe ends with `generated-check`, which
fails when the build changed the committed generated files: regenerate with
`just <server>::build` and commit the result.

### Dockerfile Pattern

Multi-stage build with scratch base and OCI labels:

```dockerfile
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Code generation goes here when the server needs it (protoc, go generate)
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o app .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/probitas-test/echo-servers"
LABEL org.opencontainers.image.description="Description of the server"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
```

## CI/CD

### Build Workflows (`build.echo-*.yml`)

Triggered on push/PR to the respective app directory, `flake.*`, or the
workflow file itself. A single `build` job installs Nix
(`nixbuild/nix-quick-install-action`), loads the development shell into the
job (`nicknovitski/nix-develop`), and runs the server's gate:

```yaml
- uses: nixbuild/nix-quick-install-action@v35
- uses: nicknovitski/nix-develop@v1
- run: just echo-http::ci
```

### Format Workflow (`fmt-check.yml`)

Runs `just fmt-check` (dprint and nixfmt) on every push and pull request, since
formatting applies to every file rather than one server.

### Docker Workflows (`docker.echo-*.yml`)

Triggered on push to main or release publish:

- Builds multi-arch images (linux/amd64, linux/arm64)
- Pushes to GHCR with tags: `latest`, branch name, version tag (on release)
