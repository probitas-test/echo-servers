FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app

# Install protoc and plugins
RUN apk add --no-cache protobuf && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.35.2 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Generate proto files
RUN protoc -I proto --go_out=proto --go_opt=paths=source_relative \
    --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
    proto/*.proto

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o echo-grpc .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/jsr-probitas/dockerfiles"
LABEL org.opencontainers.image.description="gRPC echo server for testing gRPC clients"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/echo-grpc /echo-grpc
EXPOSE 50051
ENTRYPOINT ["/echo-grpc"]
