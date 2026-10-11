# Run `nix develop` (or use direnv) first so every tool is on PATH.

set shell := ["bash", "-euo", "pipefail", "-c"]

mod echo-http
mod echo-grpc
mod echo-graphql
mod echo-connectrpc

[private]
default:
    @just --list

# Run linter on all packages
lint: echo-http::lint echo-grpc::lint echo-graphql::lint echo-connectrpc::lint

# Run tests on all packages
test: echo-http::test echo-grpc::test echo-graphql::test echo-connectrpc::test

# Build all packages
build: echo-http::build echo-grpc::build echo-graphql::build echo-connectrpc::build

# Format all code (Go + Markdown/JSON/YAML + Nix)
fmt: echo-http::fmt echo-grpc::fmt echo-graphql::fmt echo-connectrpc::fmt
    dprint fmt
    git ls-files -z '*.nix' | xargs -0 nixfmt

# Clean all packages
clean: echo-http::clean echo-grpc::clean echo-graphql::clean echo-connectrpc::clean

# Tidy all packages
tidy: echo-http::tidy echo-grpc::tidy echo-graphql::tidy echo-connectrpc::tidy

# Check the formatting that spans the repository: Markdown/JSON/YAML (dprint)
# and Nix. Each server checks its Go formatting in its own fmt-check.
fmt-check:
    dprint check
    git ls-files -z '*.nix' | xargs -0 nixfmt --check

# Everything CI runs: each server's lint, fmt-check, test and build, plus the
# repository-wide formatting
ci: echo-http::ci echo-grpc::ci echo-graphql::ci echo-connectrpc::ci fmt-check
