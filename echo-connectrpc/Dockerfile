FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app

# Install protoc and plugins
RUN apk add --no-cache protobuf && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.2 && \
    go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.18.1

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Generate proto files
RUN protoc -I proto --go_out=proto --go_opt=paths=source_relative \
    --connect-go_out=proto --connect-go_opt=paths=source_relative \
    proto/*.proto

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o echo-connectrpc .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/probitas-test/echo-servers"
LABEL org.opencontainers.image.description="Connect RPC echo server for testing Connect RPC clients"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/echo-connectrpc /echo-connectrpc
EXPOSE 8080
ENTRYPOINT ["/echo-connectrpc"]
