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
RUN protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/echo.proto

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o echo-grpc .

FROM scratch
COPY --from=builder /app/echo-grpc /echo-grpc
EXPOSE 50051
ENTRYPOINT ["/echo-grpc"]
