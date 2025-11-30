mod echo-http
mod echo-grpc
mod echo-graphql

[private]
default:
    @just --list

# Run linter on all packages
lint: echo-http::lint echo-grpc::lint echo-graphql::lint

# Run tests on all packages
test: echo-http::test echo-grpc::test echo-graphql::test

# Build all packages
build: echo-http::build echo-grpc::build echo-graphql::build

# Format all code (Go + Markdown/JSON/YAML)
fmt: echo-http::fmt echo-grpc::fmt echo-graphql::fmt
    dprint fmt

# Clean all packages
clean: echo-http::clean echo-grpc::clean echo-graphql::clean

# Tidy all packages
tidy: echo-http::tidy echo-grpc::tidy echo-graphql::tidy
