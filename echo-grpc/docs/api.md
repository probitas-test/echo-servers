# echo-grpc API Reference

## Base URL

| Environment    | Address           |
| -------------- | ----------------- |
| Container      | `localhost:50051` |
| Docker Compose | `localhost:50051` |

> **Note:** The container listens on port 50051. The same port is exposed
> when using `docker compose up`.

## Environment Variables

### Server Configuration

| Variable | Default   | Description  |
| -------- | --------- | ------------ |
| `HOST`   | `0.0.0.0` | Bind address |
| `PORT`   | `50051`   | Listen port  |

### gRPC Reflection Configuration

| Variable                          | Default | Description                                   |
| --------------------------------- | ------- | --------------------------------------------- |
| `REFLECTION_INCLUDE_DEPENDENCIES` | `false` | Include transitive dependencies in reflection |
| `DISABLE_REFLECTION_V1`           | `false` | Disable gRPC reflection v1 API                |
| `DISABLE_REFLECTION_V1ALPHA`      | `false` | Disable gRPC reflection v1alpha API           |

These flags allow testing client compatibility with different reflection API versions.

---

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

Standard gRPC health checking protocol.

```protobuf
service Health {
  rpc Check (HealthCheckRequest) returns (HealthCheckResponse);
  rpc Watch (HealthCheckRequest) returns (stream HealthCheckResponse);
}
```

## Messages

### EchoRequest

```protobuf
message EchoRequest {
  string message = 1;
}
```

| Field     | Type   | Description          |
| --------- | ------ | -------------------- |
| `message` | string | Message to echo back |

### EchoResponse

```protobuf
message EchoResponse {
  string message = 1;
  map<string, string> metadata = 2;
}
```

| Field      | Type               | Description               |
| ---------- | ------------------ | ------------------------- |
| `message`  | string             | Echoed message            |
| `metadata` | map<string,string> | Request metadata (echoed) |

### EchoWithDelayRequest

```protobuf
message EchoWithDelayRequest {
  string message = 1;
  int32 delay_ms = 2;
}
```

| Field      | Type   | Description           |
| ---------- | ------ | --------------------- |
| `message`  | string | Message to echo back  |
| `delay_ms` | int32  | Delay before response |

### EchoErrorRequest

```protobuf
message EchoErrorRequest {
  string message = 1;
  int32 code = 2;
  string details = 3;
}
```

| Field     | Type   | Description               |
| --------- | ------ | ------------------------- |
| `message` | string | Message (unused in error) |
| `code`    | int32  | gRPC status code (0-16)   |
| `details` | string | Error details message     |

### ServerStreamRequest

```protobuf
message ServerStreamRequest {
  string message = 1;
  int32 count = 2;
  int32 interval_ms = 3;
}
```

| Field         | Type   | Description                      |
| ------------- | ------ | -------------------------------- |
| `message`     | string | Message to echo in each response |
| `count`       | int32  | Number of responses to stream    |
| `interval_ms` | int32  | Interval between responses       |

### EchoRequestMetadataRequest

```protobuf
message EchoRequestMetadataRequest {
  repeated string keys = 1;
}
```

| Field  | Type            | Description                                  |
| ------ | --------------- | -------------------------------------------- |
| `keys` | repeated string | Filter to specific keys (empty = return all) |

### EchoRequestMetadataResponse

```protobuf
message EchoRequestMetadataResponse {
  map<string, MetadataValues> metadata = 1;
}

message MetadataValues {
  repeated string values = 1;
}
```

| Field      | Type                        | Description                    |
| ---------- | --------------------------- | ------------------------------ |
| `metadata` | map<string, MetadataValues> | All request metadata as values |

### EchoWithTrailersRequest

```protobuf
message EchoWithTrailersRequest {
  string message = 1;
  map<string, string> trailers = 2;
}
```

| Field      | Type               | Description                    |
| ---------- | ------------------ | ------------------------------ |
| `message`  | string             | Message to echo back           |
| `trailers` | map<string,string> | Trailers to send with response |

### EchoLargePayloadRequest

```protobuf
message EchoLargePayloadRequest {
  int32 size_bytes = 1;
  string pattern = 2;
}
```

| Field        | Type   | Description                          |
| ------------ | ------ | ------------------------------------ |
| `size_bytes` | int32  | Size of payload to return (max 10MB) |
| `pattern`    | string | Pattern to repeat (default: 'X')     |

### EchoLargePayloadResponse

```protobuf
message EchoLargePayloadResponse {
  bytes payload = 1;
  int32 actual_size = 2;
}
```

| Field         | Type  | Description                |
| ------------- | ----- | -------------------------- |
| `payload`     | bytes | Generated payload          |
| `actual_size` | int32 | Actual size of the payload |

### EchoDeadlineRequest

```protobuf
message EchoDeadlineRequest {
  string message = 1;
}
```

| Field     | Type   | Description          |
| --------- | ------ | -------------------- |
| `message` | string | Message to echo back |

### EchoDeadlineResponse

```protobuf
message EchoDeadlineResponse {
  string message = 1;
  int64 deadline_remaining_ms = 2;
  bool has_deadline = 3;
}
```

| Field                   | Type   | Description                     |
| ----------------------- | ------ | ------------------------------- |
| `message`               | string | Echoed message                  |
| `deadline_remaining_ms` | int64  | Remaining deadline (-1 if none) |
| `has_deadline`          | bool   | Whether a deadline was set      |

### EchoErrorWithDetailsRequest

```protobuf
message EchoErrorWithDetailsRequest {
  int32 code = 1;
  string message = 2;
  repeated ErrorDetail details = 3;
}

message ErrorDetail {
  string type = 1;
  repeated FieldViolation field_violations = 2;
  int64 retry_delay_ms = 3;
  repeated string stack_entries = 4;
  string debug_detail = 5;
  repeated QuotaViolation quota_violations = 6;
}

message FieldViolation {
  string field = 1;
  string description = 2;
}

message QuotaViolation {
  string subject = 1;
  string description = 2;
}
```

| Field     | Type                 | Description             |
| --------- | -------------------- | ----------------------- |
| `code`    | int32                | gRPC status code (0-16) |
| `message` | string               | Error message           |
| `details` | repeated ErrorDetail | Rich error details      |

**ErrorDetail types:**

- `bad_request` - Uses `field_violations` for validation errors
- `retry_info` - Uses `retry_delay_ms` for retry guidance
- `debug_info` - Uses `stack_entries` and `debug_detail`
- `quota_failure` - Uses `quota_violations` for quota errors

## RPCs

### Echo (Unary)

Simple echo - returns the input message.

```bash
grpcurl -plaintext -d '{"message": "hello"}' \
  localhost:50051 echo.v1.Echo/Echo
```

**Response:**

```json
{
  "message": "hello",
  "metadata": {
    "content-type": "application/grpc"
  }
}
```

### EchoWithDelay (Unary)

Echo with delay for timeout testing.

```bash
grpcurl -plaintext -d '{"message": "hello", "delay_ms": 5000}' \
  localhost:50051 echo.v1.Echo/EchoWithDelay
```

**Response:** Same as Echo, returned after specified delay.

### EchoError (Unary)

Returns a gRPC error with the specified status code.

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
grpcurl -plaintext -d '{"message": "test", "code": 5, "details": "resource not found"}' \
  localhost:50051 echo.v1.Echo/EchoError
```

**Response:**

```
ERROR:
  Code: NotFound
  Message: resource not found
```

### EchoRequestMetadata (Unary)

Returns all request metadata in response body. Useful for verifying auth tokens and custom headers.

```bash
grpcurl -plaintext \
  -H "authorization: Bearer token123" \
  -H "x-request-id: req-456" \
  -d '{}' \
  localhost:50051 echo.v1.Echo/EchoRequestMetadata
```

**Response:**

```json
{
  "metadata": {
    "authorization": { "values": ["Bearer token123"] },
    "x-request-id": { "values": ["req-456"] },
    "content-type": { "values": ["application/grpc"] }
  }
}
```

**Filter to specific keys:**

```bash
grpcurl -plaintext \
  -H "authorization: Bearer token123" \
  -H "x-request-id: req-456" \
  -d '{"keys": ["authorization"]}' \
  localhost:50051 echo.v1.Echo/EchoRequestMetadata
```

### EchoWithTrailers (Unary)

Return response with specified trailing metadata. Useful for testing trailer handling.

```bash
grpcurl -plaintext -d '{
  "message": "hello",
  "trailers": {
    "x-custom-trailer": "value1",
    "x-timing": "100ms"
  }
}' localhost:50051 echo.v1.Echo/EchoWithTrailers
```

**Response:**

```json
{
  "message": "hello",
  "metadata": {
    "content-type": "application/grpc"
  }
}
```

Trailers can be received using grpcurl's `-v` flag or programmatically via `grpc.Trailer()`.

### EchoLargePayload (Unary)

Returns a large payload of specified size. Useful for testing payload limits and chunking.

```bash
grpcurl -plaintext -d '{"size_bytes": 1024, "pattern": "ABC"}' \
  localhost:50051 echo.v1.Echo/EchoLargePayload
```

**Response:**

```json
{
  "payload": "QUJDQUJDQUJD...", // base64 encoded
  "actualSize": 1024
}
```

**Limits:** Maximum 10MB (10,485,760 bytes)

### EchoDeadline (Unary)

Echo the remaining deadline/timeout. Useful for verifying timeout propagation.

```bash
grpcurl -plaintext -max-time 10 -d '{"message": "test"}' \
  localhost:50051 echo.v1.Echo/EchoDeadline
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
grpcurl -plaintext -d '{
  "code": 3,
  "message": "validation failed",
  "details": [{
    "type": "bad_request",
    "field_violations": [
      {"field": "email", "description": "invalid email format"},
      {"field": "age", "description": "must be positive"}
    ]
  }]
}' localhost:50051 echo.v1.Echo/EchoErrorWithDetails
```

**RetryInfo example:**

```bash
grpcurl -plaintext -d '{
  "code": 14,
  "message": "service temporarily unavailable",
  "details": [{
    "type": "retry_info",
    "retry_delay_ms": 5000
  }]
}' localhost:50051 echo.v1.Echo/EchoErrorWithDetails
```

**DebugInfo example:**

```bash
grpcurl -plaintext -d '{
  "code": 13,
  "message": "internal error",
  "details": [{
    "type": "debug_info",
    "stack_entries": ["main.go:42", "handler.go:15"],
    "debug_detail": "null pointer exception"
  }]
}' localhost:50051 echo.v1.Echo/EchoErrorWithDetails
```

**QuotaFailure example:**

```bash
grpcurl -plaintext -d '{
  "code": 8,
  "message": "quota exceeded",
  "details": [{
    "type": "quota_failure",
    "quota_violations": [
      {"subject": "user:123", "description": "API calls per minute exceeded"}
    ]
  }]
}' localhost:50051 echo.v1.Echo/EchoErrorWithDetails
```

### ServerStream (Server Streaming)

Server sends multiple responses over time.

```bash
grpcurl -plaintext -d '{"message": "ping", "count": 5, "interval_ms": 1000}' \
  localhost:50051 echo.v1.Echo/ServerStream
```

**Response:** Streams `count` responses with `interval_ms` delay between each:

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
echo '{"message": "hello"}
{"message": "world"}
{"message": "!"}' | grpcurl -plaintext -d @ \
  localhost:50051 echo.v1.Echo/ClientStream
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
echo '{"message": "one"}
{"message": "two"}
{"message": "three"}' | grpcurl -plaintext -d @ \
  localhost:50051 echo.v1.Echo/BidirectionalStream
```

**Response:**

```json
{"message": "one", "metadata": {...}}
{"message": "two", "metadata": {...}}
{"message": "three", "metadata": {...}}
```

## Health Checking

Standard gRPC health checking protocol is supported.

### Check (Unary)

```bash
grpcurl -plaintext -d '{"service": ""}' \
  localhost:50051 grpc.health.v1.Health/Check
```

**Response:**

```json
{
  "status": "SERVING"
}
```

**Check specific service:**

```bash
grpcurl -plaintext -d '{"service": "echo.v1.Echo"}' \
  localhost:50051 grpc.health.v1.Health/Check
```

### Watch (Server Streaming)

Stream health status changes.

```bash
grpcurl -plaintext -d '{"service": ""}' \
  localhost:50051 grpc.health.v1.Health/Watch
```

## Server Reflection

The server supports gRPC server reflection for service discovery (both v1 and v1alpha versions).

> **Configuration:** See [Environment Variables](#environment-variables) section for reflection configuration options.

```bash
# List all services
grpcurl -plaintext localhost:50051 list

# Describe service
grpcurl -plaintext localhost:50051 describe echo.v1.Echo

# Describe message
grpcurl -plaintext localhost:50051 describe echo.v1.EchoRequest
```

## Metadata

Request metadata is echoed back in the `metadata` field of every response. Custom metadata can be sent using grpcurl's `-H` flag:

```bash
grpcurl -plaintext \
  -H "x-request-id: abc123" \
  -H "x-custom-header: value" \
  -d '{"message": "hello"}' \
  localhost:50051 echo.v1.Echo/Echo
```

**Response:**

```json
{
  "message": "hello",
  "metadata": {
    "content-type": "application/grpc",
    "x-custom-header": "value",
    "x-request-id": "abc123"
  }
}
```
