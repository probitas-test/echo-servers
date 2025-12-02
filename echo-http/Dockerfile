FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder
ARG TARGETOS TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o echo-http .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/jsr-probitas/echo-servers"
LABEL org.opencontainers.image.description="HTTP echo server for testing HTTP clients"
LABEL org.opencontainers.image.licenses="MIT"
COPY --from=builder /app/echo-http /echo-http
EXPOSE 80
ENTRYPOINT ["/echo-http"]
