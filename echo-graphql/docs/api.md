# echo-graphql API Reference

## Base URL

| Environment    | URL                      |
| -------------- | ------------------------ |
| Container      | `http://localhost:8080`  |
| Docker Compose | `http://localhost:14000` |

> **Note:** The container listens on port 8080. When using `docker compose up`, the
> port is mapped to 14000 on the host.

## Environment Variables

### Server Configuration

| Variable | Default   | Description  |
| -------- | --------- | ------------ |
| `HOST`   | `0.0.0.0` | Bind address |
| `PORT`   | `8080`    | Listen port  |

---

## Schema

### Types

#### Message

```graphql
type Message {
  id: ID!
  text: String!
  createdAt: String!
}
```

| Field       | Type    | Description               |
| ----------- | ------- | ------------------------- |
| `id`        | ID!     | Unique message identifier |
| `text`      | String! | Message content           |
| `createdAt` | String! | ISO 8601 timestamp        |

#### EchoResult

```graphql
type EchoResult {
  message: String
  error: String
}
```

| Field     | Type   | Description                     |
| --------- | ------ | ------------------------------- |
| `message` | String | Echoed message (null on error)  |
| `error`   | String | Error message (null on success) |

#### Headers

```graphql
type Headers {
  authorization: String
  contentType: String
  custom(name: String!): String
  all: [HeaderEntry!]!
}
```

| Field           | Type           | Description                    |
| --------------- | -------------- | ------------------------------ |
| `authorization` | String         | Authorization header value     |
| `contentType`   | String         | Content-Type header value      |
| `custom`        | String         | Custom header by name          |
| `all`           | [HeaderEntry!] | All headers as key-value pairs |

#### HeaderEntry

```graphql
type HeaderEntry {
  name: String!
  value: String!
}
```

#### NestedEcho

```graphql
type NestedEcho {
  value: String!
  child: NestedEcho
}
```

| Field   | Type       | Description                       |
| ------- | ---------- | --------------------------------- |
| `value` | String!    | The value at this level           |
| `child` | NestedEcho | Child node (null if at max depth) |

#### EchoListItem

```graphql
type EchoListItem {
  index: Int!
  message: String!
}
```

| Field     | Type    | Description         |
| --------- | ------- | ------------------- |
| `index`   | Int!    | Zero-based index    |
| `message` | String! | The message content |

## Queries

### echo

Echo back the input message.

```graphql
query {
  echo(message: "hello")
}
```

**Response:**

```json
{
  "data": {
    "echo": "hello"
  }
}
```

**curl:**

```bash
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echo(message: \"hello\") }"}'
```

### echoWithDelay

Echo with delay for timeout testing.

| Argument  | Type    | Description           |
| --------- | ------- | --------------------- |
| `message` | String! | Message to echo       |
| `delayMs` | Int!    | Delay in milliseconds |

```graphql
query {
  echoWithDelay(message: "hello", delayMs: 5000)
}
```

**curl:**

```bash
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ echoWithDelay(message: \"hello\", delayMs: 5000) }"}'
```

### echoError

Always returns a GraphQL error.

```graphql
query {
  echoError(message: "test")
}
```

**Response:**

```json
{
  "data": {
    "echoError": null
  },
  "errors": [
    {
      "message": "echo error: test",
      "path": ["echoError"]
    }
  ]
}
```

### echoPartialError

Returns partial data with errors. Messages containing "error" will fail.

```graphql
query {
  echoPartialError(messages: ["hello", "error", "world"]) {
    message
    error
  }
}
```

**Response:**

```json
{
  "data": {
    "echoPartialError": [
      { "message": "hello", "error": null },
      { "message": null, "error": "message contains 'error'" },
      { "message": "world", "error": null }
    ]
  }
}
```

### echoWithExtensions

Returns data with custom GraphQL extensions.

```graphql
query {
  echoWithExtensions(message: "hello")
}
```

**Response:**

```json
{
  "data": {
    "echoWithExtensions": "hello"
  },
  "extensions": {
    "timing": {
      "startTime": "2024-01-01T00:00:00.000000000Z",
      "duration": "0ms"
    },
    "tracing": {
      "version": 1,
      "requestId": "req-1234567890"
    }
  }
}
```

### echoHeaders

Return request headers for auth verification testing.

```graphql
query {
  echoHeaders {
    authorization
    contentType
    custom(name: "X-Custom-Header")
    all { name value }
  }
}
```

**Response:**

```json
{
  "data": {
    "echoHeaders": {
      "authorization": "Bearer token123",
      "contentType": "application/json",
      "custom": "custom-value",
      "all": [
        { "name": "Authorization", "value": "Bearer token123" },
        { "name": "Content-Type", "value": "application/json" }
      ]
    }
  }
}
```

**curl:**

```bash
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -d '{"query": "{ echoHeaders { authorization contentType all { name value } } }"}'
```

### echoNested

Return deeply nested object for recursive response parsing tests.

| Argument  | Type    | Description          |
| --------- | ------- | -------------------- |
| `message` | String! | Message to include   |
| `depth`   | Int!    | Nesting depth (>= 1) |

```graphql
query {
  echoNested(message: "test", depth: 3) {
    value
    child {
      value
      child {
        value
      }
    }
  }
}
```

**Response:**

```json
{
  "data": {
    "echoNested": {
      "value": "test (level 1)",
      "child": {
        "value": "test (level 2)",
        "child": {
          "value": "test (level 3)"
        }
      }
    }
  }
}
```

### echoList

Return list of n items for pagination/list handling tests.

| Argument  | Type    | Description           |
| --------- | ------- | --------------------- |
| `message` | String! | Message for each item |
| `count`   | Int!    | Number of items       |

```graphql
query {
  echoList(message: "item", count: 3) {
    index
    message
  }
}
```

**Response:**

```json
{
  "data": {
    "echoList": [
      { "index": 0, "message": "item" },
      { "index": 1, "message": "item" },
      { "index": 2, "message": "item" }
    ]
  }
}
```

### echoNull

Always returns null for null handling tests.

```graphql
query {
  echoNull
}
```

**Response:**

```json
{
  "data": {
    "echoNull": null
  }
}
```

### echoOptional

Returns value or null based on flag for optional value tests.

| Argument     | Type     | Description           |
| ------------ | -------- | --------------------- |
| `message`    | String!  | Message to return     |
| `returnNull` | Boolean! | If true, returns null |

```graphql
query {
  echoOptional(message: "hello", returnNull: false)
}
```

**Response:**

```json
{
  "data": {
    "echoOptional": "hello"
  }
}
```

## Mutations

### createMessage

Create a new message.

```graphql
mutation {
  createMessage(text: "Hello, World!") {
    id
    text
    createdAt
  }
}
```

**Response:**

```json
{
  "data": {
    "createMessage": {
      "id": "1",
      "text": "Hello, World!",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  }
}
```

**curl:**

```bash
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "mutation { createMessage(text: \"Hello\") { id text createdAt } }"}'
```

### updateMessage

Update an existing message.

```graphql
mutation {
  updateMessage(id: "1", text: "Updated text") {
    id
    text
    createdAt
  }
}
```

**Response:**

```json
{
  "data": {
    "updateMessage": {
      "id": "1",
      "text": "Updated text",
      "createdAt": "2024-01-01T00:00:00Z"
    }
  }
}
```

### deleteMessage

Delete a message. Returns `true` if the message existed.

```graphql
mutation {
  deleteMessage(id: "1")
}
```

**Response:**

```json
{
  "data": {
    "deleteMessage": true
  }
}
```

### batchCreateMessages

Create multiple messages at once for batch operation testing.

| Argument | Type       | Description           |
| -------- | ---------- | --------------------- |
| `texts`  | [String!]! | List of message texts |

```graphql
mutation {
  batchCreateMessages(texts: ["first", "second", "third"]) {
    id
    text
    createdAt
  }
}
```

**Response:**

```json
{
  "data": {
    "batchCreateMessages": [
      { "id": "1", "text": "first", "createdAt": "2024-01-01T00:00:00Z" },
      { "id": "2", "text": "second", "createdAt": "2024-01-01T00:00:00Z" },
      { "id": "3", "text": "third", "createdAt": "2024-01-01T00:00:00Z" }
    ]
  }
}
```

## Subscriptions

Subscriptions use WebSocket protocol. Connect to `ws://localhost:14000/graphql`.

### messageCreated

Subscribe to new message events. Triggered when `createMessage` is called.

```graphql
subscription {
  messageCreated {
    id
    text
    createdAt
  }
}
```

**Using websocat:**

```bash
echo '{"type":"start","id":"1","payload":{"query":"subscription { messageCreated { id text } }"}}' | \
  websocat ws://localhost:14000/graphql -n
```

### countdown

Subscribe to countdown events. Streams integers from `from` down to 0.

```graphql
subscription {
  countdown(from: 5)
}
```

**Response stream:**

```json
{"data": {"countdown": 5}}
{"data": {"countdown": 4}}
{"data": {"countdown": 3}}
{"data": {"countdown": 2}}
{"data": {"countdown": 1}}
{"data": {"countdown": 0}}
```

### messageCreatedFiltered

Subscribe to messages with optional text filter.

| Argument       | Type   | Description                          |
| -------------- | ------ | ------------------------------------ |
| `textContains` | String | Filter messages containing this text |

```graphql
subscription {
  messageCreatedFiltered(textContains: "important") {
    id
    text
    createdAt
  }
}
```

Only messages containing "important" in their text will be received.

### heartbeat

Periodic heartbeat for connection testing. Sends ISO 8601 timestamps at the specified interval.

| Argument     | Type | Description              |
| ------------ | ---- | ------------------------ |
| `intervalMs` | Int! | Interval in milliseconds |

```graphql
subscription {
  heartbeat(intervalMs: 1000)
}
```

**Response stream:**

```json
{"data": {"heartbeat": "2024-01-01T00:00:00.000000000Z"}}
{"data": {"heartbeat": "2024-01-01T00:00:01.000000000Z"}}
{"data": {"heartbeat": "2024-01-01T00:00:02.000000000Z"}}
```

## Introspection

GraphQL introspection is enabled. Query the schema:

```graphql
query {
  __schema {
    types {
      name
    }
  }
}
```

**curl:**

```bash
curl -X POST http://localhost:14000/graphql \
  -H "Content-Type: application/json" \
  -d '{"query": "{ __schema { types { name } } }"}'
```

## Error Handling

### Standard Errors

```json
{
  "data": null,
  "errors": [
    {
      "message": "error description",
      "path": ["fieldName"],
      "locations": [{ "line": 1, "column": 3 }]
    }
  ]
}
```

### Partial Errors

GraphQL allows partial success. Some fields may return data while others return errors:

```json
{
  "data": {
    "echo": "hello",
    "echoError": null
  },
  "errors": [
    {
      "message": "echo error: test",
      "path": ["echoError"]
    }
  ]
}
```

## Health Check

```bash
curl http://localhost:14000/health
```

**Response:**

```json
{
  "status": "ok"
}
```
