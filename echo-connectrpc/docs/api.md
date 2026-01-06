# echo-connectrpc API Reference

## Base URL

| Environment    | Address          |
| -------------- | ---------------- |
| Container      | `localhost:8080` |
| Docker Compose | `localhost:8080` |

> **Note:** The container listens on port 8080. The same port is exposed when using
> `docker compose up`.

## Environment Variables

### Server Configuration

| Variable | Default   | Description  |
| -------- | --------- | ------------ |
| `HOST`   | `0.0.0.0` | Bind address |
| `PORT`   | `8080`    | Listen port  |

### Protocol Control

| Variable             | Default | Description                  |
| -------------------- | ------- | ---------------------------- |
| `DISABLE_CONNECTRPC` | `false` | Disable Connect RPC protocol |
| `DISABLE_GRPC`       | `false` | Disable gRPC protocol        |
| `DISABLE_GRPC_WEB`   | `false` | Disable gRPC-Web protocol    |

### Reflection Control

| Variable                          | Default | Description                                                 |
| --------------------------------- | ------- | ----------------------------------------------------------- |
| `REFLECTION_INCLUDE_DEPENDENCIES` | `false` | Include transitive dependencies (note: not fully supported) |
| `DISABLE_REFLECTION_V1`           | `false` | Disable gRPC reflection v1 API                              |
| `DISABLE_REFLECTION_V1ALPHA`      | `false` | Disable gRPC reflection v1alpha API                         |

**Note:** At least one protocol must be enabled. The server will refuse to start if all protocols are disabled.

**Examples:**

```bash
# Disable gRPC protocol (keep Connect RPC and gRPC-Web)
DISABLE_GRPC=true ./echo-connectrpc

# Only enable Connect RPC protocol
DISABLE_GRPC=true DISABLE_GRPC_WEB=true ./echo-connectrpc

# Disable reflection v1alpha (for compatibility testing)
DISABLE_REFLECTION_V1ALPHA=true ./echo-connectrpc
```

---

## Protocol

Connect RPC supports three protocols:

- **Connect Protocol** (recommended) - Modern, efficient protocol with JSON/binary support
- **gRPC Protocol** - Compatible with standard gRPC clients
- **gRPC-Web Protocol** - Browser-compatible gRPC

All protocols are available over HTTP/1.1 and HTTP/2, with both JSON and Protocol Buffers encoding.

Each protocol can be individually disabled using environment variables (see [Environment Variables](#environment-variables) above).

## Services

### Echo Service (echo.v1.Echo)

```protobuf
package echo.v1;

service Echo {
  // Unary RPCs
  rpc Echo (EchoRequest) returns (EchoResponse);
  rpc EchoWithDelay (EchoWithDelayRequest) returns (EchoResponse);
  rpc EchoError (EchoErrorRequest) returns (EchoResponse);

  // Metadata/Headers RPCs
  rpc EchoRequestMetadata (EchoRequestMetadataRequest) returns (EchoRequestMetadataResponse);
  rpc EchoWithTrailers (EchoWithTrailersRequest) returns (EchoResponse);

  // Payload Testing RPCs
  rpc EchoLargePayload (EchoLargePayloadRequest) returns (EchoLargePayloadResponse);

  // Deadline/Timeout RPCs
  rpc EchoDeadline (EchoDeadlineRequest) returns (EchoDeadlineResponse);

  // Error Scenarios RPCs
  rpc EchoErrorWithDetails (EchoErrorWithDetailsRequest) returns (EchoResponse);

  // Streaming RPCs
  rpc ServerStream (ServerStreamRequest) returns (stream EchoResponse);
  rpc ClientStream (stream EchoRequest) returns (EchoResponse);
  rpc BidirectionalStream (stream EchoRequest) returns (stream EchoResponse);
}
```

### Health Service (grpc.health.v1.Health)

Standard gRPC health checking protocol, compatible with Connect RPC.

```protobuf
service Health {
  rpc Check (HealthCheckRequest) returns (HealthCheckResponse);
  rpc Watch (HealthCheckRequest) returns (stream HealthCheckResponse);
}
```

## Messages

For detailed message definitions, see the [echo-grpc API reference](../echo-grpc/docs/api.md).
The message formats are identical between echo-grpc and echo-connectrpc.

## RPCs

All examples below use curl with the Connect protocol and JSON encoding.

### Echo (Unary)

Simple echo - returns the input message.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

**Response:**

```json
{
  "message": "hello",
  "metadata": {
    "content-type": "application/json"
  }
}
```

### EchoWithDelay (Unary)

Echo with delay for timeout testing.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoWithDelay \
  -H "Content-Type: application/json" \
  -d '{"message": "hello", "delayMs": 5000}'
```

**Response:** Same as Echo, returned after specified delay.

### EchoError (Unary)

Returns a Connect error with the specified status code.

| Code | Name                |
| ---- | ------------------- |
| 0    | OK                  |
| 1    | CANCELLED           |
| 2    | UNKNOWN             |
| 3    | INVALID_ARGUMENT    |
| 4    | DEADLINE_EXCEEDED   |
| 5    | NOT_FOUND           |
| 6    | ALREADY_EXISTS      |
| 7    | PERMISSION_DENIED   |
| 8    | RESOURCE_EXHAUSTED  |
| 9    | FAILED_PRECONDITION |
| 10   | ABORTED             |
| 11   | OUT_OF_RANGE        |
| 12   | UNIMPLEMENTED       |
| 13   | INTERNAL            |
| 14   | UNAVAILABLE         |
| 15   | DATA_LOSS           |
| 16   | UNAUTHENTICATED     |

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoError \
  -H "Content-Type: application/json" \
  -d '{"message": "test", "code": 5, "details": "resource not found"}'
```

**Response:**

```json
{
  "code": "not_found",
  "message": "resource not found"
}
```

### EchoRequestMetadata (Unary)

Returns all request headers in response body. Useful for verifying auth tokens and custom headers.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoRequestMetadata \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -H "X-Request-Id: req-456" \
  -d '{}'
```

**Response:**

```json
{
  "metadata": {
    "authorization": { "values": ["Bearer token123"] },
    "x-request-id": { "values": ["req-456"] },
    "content-type": { "values": ["application/json"] }
  }
}
```

**Filter to specific keys:**

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoRequestMetadata \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -H "X-Request-Id: req-456" \
  -d '{"keys": ["authorization"]}'
```

### EchoWithTrailers (Unary)

Return response with specified trailing metadata. Useful for testing trailer handling.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoWithTrailers \
  -H "Content-Type: application/json" \
  -d '{
    "message": "hello",
    "trailers": {
      "x-custom-trailer": "value1",
      "x-timing": "100ms"
    }
  }'
```

**Response:**

```json
{
  "message": "hello",
  "metadata": {
    "content-type": "application/json"
  }
}
```

Trailers are returned in HTTP response trailers and can be inspected with verbose mode (`curl -v`).

### EchoLargePayload (Unary)

Returns a large payload of specified size. Useful for testing payload limits and chunking.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoLargePayload \
  -H "Content-Type: application/json" \
  -d '{"sizeBytes": 1024, "pattern": "ABC"}'
```

**Response:**

```json
{
  "payload": "QUJDQUJDQUJD...",
  "actualSize": 1024
}
```

**Limits:** Maximum 10MB (10,485,760 bytes)

### EchoDeadline (Unary)

Echo the remaining deadline/timeout. Useful for verifying timeout propagation.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoDeadline \
  -H "Content-Type: application/json" \
  -H "Connect-Timeout-Ms: 10000" \
  -d '{"message": "test"}'
```

**Response:**

```json
{
  "message": "test",
  "deadlineRemainingMs": 9850,
  "hasDeadline": true
}
```

**Without deadline:**

```json
{
  "message": "test",
  "deadlineRemainingMs": -1,
  "hasDeadline": false
}
```

### EchoErrorWithDetails (Unary)

Returns error with rich error details using `google.rpc.Status`.

**BadRequest example:**

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoErrorWithDetails \
  -H "Content-Type: application/json" \
  -d '{
    "code": 3,
    "message": "validation failed",
    "details": [{
      "type": "bad_request",
      "fieldViolations": [
        {"field": "email", "description": "invalid email format"},
        {"field": "age", "description": "must be positive"}
      ]
    }]
  }'
```

**RetryInfo example:**

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/EchoErrorWithDetails \
  -H "Content-Type: application/json" \
  -d '{
    "code": 14,
    "message": "service temporarily unavailable",
    "details": [{
      "type": "retry_info",
      "retryDelayMs": 5000
    }]
  }'
```

### ServerStream (Server Streaming)

Server sends multiple responses over time.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/ServerStream \
  -H "Content-Type: application/json" \
  -d '{"message": "ping", "count": 5, "intervalMs": 1000}' \
  --no-buffer
```

**Response:** Streams `count` responses with `intervalMs` delay between each (newline-delimited JSON):

```json
{"message": "ping [1/5]", "metadata": {...}}
{"message": "ping [2/5]", "metadata": {...}}
{"message": "ping [3/5]", "metadata": {...}}
{"message": "ping [4/5]", "metadata": {...}}
{"message": "ping [5/5]", "metadata": {...}}
```

### ClientStream (Client Streaming)

Client sends multiple messages, server responds once with aggregated result.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/ClientStream \
  -H "Content-Type: application/connect+json" \
  -d '{"message": "hello"}
{"message": "world"}
{"message": "!"}'
```

**Response:**

```json
{
  "message": "hello, world, !",
  "metadata": {...}
}
```

### BidirectionalStream (Bidirectional Streaming)

Both client and server stream messages simultaneously. Each client message is echoed back immediately.

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/BidirectionalStream \
  -H "Content-Type: application/connect+json" \
  -d '{"message": "one"}
{"message": "two"}
{"message": "three"}' \
  --no-buffer
```

**Response:**

```json
{"message": "one", "metadata": {...}}
{"message": "two", "metadata": {...}}
{"message": "three", "metadata": {...}}
```

## Health Checking

Standard gRPC health checking protocol is supported via Connect RPC.

### Check (Unary)

```bash
curl -X POST http://localhost:8080/grpc.health.v1.Health/Check \
  -H "Content-Type: application/json" \
  -d '{"service": ""}'
```

**Response:**

```json
{
  "status": "SERVING"
}
```

**Check specific service:**

```bash
curl -X POST http://localhost:8080/grpc.health.v1.Health/Check \
  -H "Content-Type: application/json" \
  -d '{"service": "echo.v1.Echo"}'
```

### Watch (Server Streaming)

Stream health status changes.

```bash
curl -X POST http://localhost:8080/grpc.health.v1.Health/Watch \
  -H "Content-Type: application/json" \
  -d '{"service": ""}' \
  --no-buffer
```

## Server Reflection

The server supports gRPC server reflection for service discovery (both v1 and v1alpha versions).

You can use `grpcurl` with the Connect RPC server:

```bash
# List all services
grpcurl -plaintext localhost:8080 list

# Describe service
grpcurl -plaintext localhost:8080 describe echo.v1.Echo

# Describe message
grpcurl -plaintext localhost:8080 describe echo.v1.EchoRequest
```

## Protocol Support

### Connect Protocol (Recommended)

Use JSON encoding with standard HTTP:

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -d '{"message": "hello"}'
```

Use Protocol Buffers encoding:

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/Echo \
  -H "Content-Type: application/proto" \
  --data-binary @request.bin
```

### gRPC Protocol

Standard gRPC clients work directly:

```bash
grpcurl -plaintext -d '{"message": "hello"}' \
  localhost:8080 echo.v1.Echo/Echo
```

### gRPC-Web Protocol

Browser-compatible gRPC-Web clients can connect:

```javascript
const client = createPromiseClient(EchoService, createConnectTransport({
  baseUrl: "http://localhost:8080",
}));

const response = await client.echo({ message: "hello" });
```

## Headers and Metadata

Request headers are echoed back in the `metadata` field of every response:

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -H "X-Request-Id: abc123" \
  -H "X-Custom-Header: value" \
  -d '{"message": "hello"}'
```

**Response:**

```json
{
  "message": "hello",
  "metadata": {
    "content-type": "application/json",
    "x-custom-header": "value",
    "x-request-id": "abc123"
  }
}
```

## Timeout/Deadline

Set timeout using the `Connect-Timeout-Ms` header:

```bash
curl -X POST http://localhost:8080/echo.v1.Echo/Echo \
  -H "Content-Type: application/json" \
  -H "Connect-Timeout-Ms: 5000" \
  -d '{"message": "hello"}'
```

For gRPC clients, use standard gRPC timeout:

```bash
grpcurl -plaintext -max-time 5 -d '{"message": "hello"}' \
  localhost:8080 echo.v1.Echo/Echo
```
