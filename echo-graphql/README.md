# echo-graphql

[![Build](https://github.com/jsr-probitas/echo-servers/actions/workflows/build.echo-graphql.yml/badge.svg)](https://github.com/jsr-probitas/echo-servers/actions/workflows/build.echo-graphql.yml)
[![Docker](https://github.com/jsr-probitas/echo-servers/actions/workflows/docker.echo-graphql.yml/badge.svg)](https://github.com/jsr-probitas/echo-servers/actions/workflows/docker.echo-graphql.yml)

GraphQL echo server for testing GraphQL clients.

## Image

```
ghcr.io/jsr-probitas/echo-graphql:latest
```

## Quick Start

```bash
docker run -p 8080:8080 ghcr.io/jsr-probitas/echo-graphql:latest
```

Access GraphQL Playground at http://localhost:8080/

## Environment Variables

| Variable | Default   | Description  |
| -------- | --------- | ------------ |
| `HOST`   | `0.0.0.0` | Bind address |
| `PORT`   | `8080`    | Listen port  |

```bash
# Custom port
docker run -p 3000:3000 -e PORT=3000 ghcr.io/jsr-probitas/echo-graphql:latest

# Using .env file
docker run -p 8080:8080 -v $(pwd)/.env:/app/.env ghcr.io/jsr-probitas/echo-graphql:latest
```

## API

### Endpoints

| Path       | Description        |
| ---------- | ------------------ |
| `/`        | GraphQL Playground |
| `/graphql` | GraphQL endpoint   |
| `/health`  | Health check       |

### Schema

```graphql
type Query {
  echo(message: String!): String!
  echoWithDelay(message: String!, delayMs: Int!): String!
  echoError(message: String!): String!
  echoPartialError(messages: [String!]!): [EchoResult!]!
  echoWithExtensions(message: String!): String!
}

type Mutation {
  createMessage(text: String!): Message!
  updateMessage(id: ID!, text: String!): Message!
  deleteMessage(id: ID!): Boolean!
}

type Subscription {
  messageCreated: Message!
  countdown(from: Int!): Int!
}
```

See [docs/api.md](./docs/api.md) for detailed API reference.

## Features

| Feature       | Description                                                                    |
| ------------- | ------------------------------------------------------------------------------ |
| Introspection | Enabled by default                                                             |
| Query         | `echo`, `echoWithDelay`, `echoError`, `echoPartialError`, `echoWithExtensions` |
| Mutation      | `createMessage`, `updateMessage`, `deleteMessage`                              |
| Subscription  | `messageCreated`, `countdown` (WebSocket)                                      |
| Playground    | Available at root path                                                         |
| Health Check  | `/health` endpoint                                                             |

## Examples

```bash
# Simple echo
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echo(message: \"hello\") }"}'

# Echo with delay (for timeout testing)
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echoWithDelay(message: \"hello\", delayMs: 5000) }"}'

# Intentional error
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echoError(message: \"test\") }"}'

# Partial error (returns data and errors)
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echoPartialError(messages: [\"hello\", \"error\", \"world\"]) { message error } }"}'

# Create message
curl -X POST http://localhost:8080/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { createMessage(text: \"hello\") { id text createdAt } }"}'
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
