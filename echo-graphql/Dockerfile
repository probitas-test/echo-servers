FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS TARGETARCH

WORKDIR /app

# Copy go mod files first
COPY go.mod ./

# Download dependencies and generate go.sum
RUN go mod download

# Copy source files
COPY . .

# Tidy modules (this ensures go.sum is created properly)
RUN go mod tidy

# Generate GraphQL code
RUN go run github.com/99designs/gqlgen generate

# Build the server
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o echo-graphql .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/jsr-probitas/echo-servers"
LABEL org.opencontainers.image.description="GraphQL echo server for testing GraphQL clients"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/echo-graphql /echo-graphql
EXPOSE 8080
ENTRYPOINT ["/echo-graphql"]
