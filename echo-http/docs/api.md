# echo-http API Reference

## Base URL

| Environment    | URL                      |
| -------------- | ------------------------ |
| Container      | `http://localhost:80`    |
| Docker Compose | `http://localhost:18080` |

> **Note:** The container listens on port 80. When using `docker compose up`, the
> port is mapped to 18080 on the host.

## Endpoints

### GET /get

Echo request information including query parameters and headers.

**Request:**

```bash
curl "http://localhost:80/get?name=test&count=5"
```

**Response:**

```json
{
  "method": "GET",
  "url": "/get?name=test&count=5",
  "args": {
    "name": "test",
    "count": "5"
  },
  "headers": {
    "Accept": "*/*",
    "User-Agent": "curl/8.0.0"
  }
}
```

### POST /post

Echo request body with JSON or form data parsing.

**Request (JSON):**

```bash
curl -X POST http://localhost:80/post \
  -H "Content-Type: application/json" \
  -d '{"message": "hello", "count": 42}'
```

**Response:**

```json
{
  "method": "POST",
  "url": "/post",
  "args": {},
  "headers": {
    "Content-Type": "application/json"
  },
  "data": "{\"message\": \"hello\", \"count\": 42}",
  "json": {
    "message": "hello",
    "count": 42
  }
}
```

**Request (Form):**

```bash
curl -X POST http://localhost:80/post \
  -d "name=test" \
  -d "email=test@example.com"
```

**Response:**

```json
{
  "method": "POST",
  "url": "/post",
  "args": {},
  "headers": {
    "Content-Type": "application/x-www-form-urlencoded"
  },
  "data": "name=test&email=test@example.com",
  "form": {
    "name": "test",
    "email": "test@example.com"
  }
}
```

### PUT /put

Echo request body (same format as POST).

```bash
curl -X PUT http://localhost:80/put \
  -H "Content-Type: application/json" \
  -d '{"id": 1, "name": "updated"}'
```

### PATCH /patch

Echo request body (same format as POST).

```bash
curl -X PATCH http://localhost:80/patch \
  -H "Content-Type: application/json" \
  -d '{"name": "patched"}'
```

### DELETE /delete

Echo request information.

```bash
curl -X DELETE "http://localhost:80/delete?id=123"
```

### GET /headers

Return request headers only.

**Request:**

```bash
curl http://localhost:80/headers \
  -H "X-Custom-Header: custom-value" \
  -H "Authorization: Bearer token123"
```

**Response:**

```json
{
  "headers": {
    "Accept": "*/*",
    "Authorization": "Bearer token123",
    "User-Agent": "curl/8.0.0",
    "X-Custom-Header": "custom-value"
  }
}
```

### GET /response-header

Set response headers based on query parameters. Each query parameter key-value
pair is set as a response header. Useful for testing HTTP client header processing.

**Request:**

```bash
curl -i "http://localhost:80/response-header?X-Custom-Header=custom-value&Cache-Control=no-cache"
```

**Response:**

```
HTTP/1.1 200 OK
Cache-Control: no-cache
Content-Type: application/json
X-Custom-Header: custom-value

{
  "headers": {
    "X-Custom-Header": "custom-value",
    "Cache-Control": "no-cache"
  }
}
```

**Examples:**

```bash
# Set custom response headers
curl -i "http://localhost:80/response-header?X-Request-Id=12345&X-Correlation-Id=abc-xyz"

# Test cache control headers
curl -i "http://localhost:80/response-header?Cache-Control=max-age=3600&Expires=Wed,%2021%20Oct%202025%2007:28:00%20GMT"

# Set content language
curl -i "http://localhost:80/response-header?Content-Language=en-US"
```

### GET /status/{code}

Return the specified HTTP status code.

| Parameter | Type | Range   | Description      |
| --------- | ---- | ------- | ---------------- |
| `code`    | int  | 100-599 | HTTP status code |

**Examples:**

```bash
# 200 OK
curl -i http://localhost:80/status/200

# 404 Not Found
curl -i http://localhost:80/status/404

# 418 I'm a teapot
curl -i http://localhost:80/status/418

# 500 Internal Server Error
curl -i http://localhost:80/status/500
```

**Response:**

Returns an empty body with the specified status code.

### GET /delay/{seconds}

Echo after a specified delay. Useful for timeout testing.

| Parameter | Type  | Range | Description      |
| --------- | ----- | ----- | ---------------- |
| `seconds` | float | 0-300 | Delay in seconds |

**Examples:**

```bash
# 2 second delay
curl http://localhost:80/delay/2

# 0.5 second delay
curl http://localhost:80/delay/0.5

# Test client timeout (10 seconds)
curl --max-time 5 http://localhost:80/delay/10
```

**Response:**

Same format as `/get` but returned after the delay.

### GET /health

Health check endpoint.

**Request:**

```bash
curl http://localhost:80/health
```

**Response:**

```json
{
  "status": "ok"
}
```

---

## Utility Endpoints

### ANY /anything and /anything/{path}

Echo any request information including method, headers, body, query parameters, and
client IP.

**Request:**

```bash
curl -X POST "http://localhost:80/anything/path/to/resource?foo=bar" \
  -H "Content-Type: application/json" \
  -d '{"key": "value"}'
```

**Response:**

```json
{
  "method": "POST",
  "url": "/anything/path/to/resource?foo=bar",
  "args": {
    "foo": "bar"
  },
  "headers": {
    "Content-Type": "application/json"
  },
  "origin": "127.0.0.1",
  "data": "{\"key\": \"value\"}",
  "json": {
    "key": "value"
  }
}
```

| Field     | Type   | Description                                  |
| --------- | ------ | -------------------------------------------- |
| `method`  | string | HTTP method used                             |
| `url`     | string | Request URL including query string           |
| `args`    | object | Parsed query parameters                      |
| `headers` | object | Request headers                              |
| `origin`  | string | Client IP address                            |
| `data`    | string | Raw request body (POST/PUT/PATCH only)       |
| `json`    | object | Parsed JSON body (if Content-Type: json)     |
| `form`    | object | Parsed form body (if Content-Type: form)     |
| `files`   | object | Uploaded file names (if multipart/form-data) |

### GET /ip

Return the client's IP address.

**Request:**

```bash
curl http://localhost:80/ip
```

**Response:**

```json
{
  "origin": "127.0.0.1"
}
```

### GET /user-agent

Return the User-Agent header.

**Request:**

```bash
curl http://localhost:80/user-agent
```

**Response:**

```json
{
  "user-agent": "curl/8.0.0"
}
```

---

## Redirect Endpoints

### GET /redirect/{n}

Redirect n times before returning 200 OK with a final response.

| Parameter | Type | Range | Description                         |
| --------- | ---- | ----- | ----------------------------------- |
| `n`       | int  | 0-100 | Number of redirects before response |

**Examples:**

```bash
# Redirect 3 times then return 200
curl -L http://localhost:80/redirect/3

# No redirect (immediate response)
curl http://localhost:80/redirect/0
```

**Final Response:**

```json
{
  "redirected": true
}
```

### GET /redirect-to

Redirect to a specified URL.

| Parameter     | Type   | Description                                    |
| ------------- | ------ | ---------------------------------------------- |
| `url`         | string | Target URL (required)                          |
| `status_code` | int    | Redirect status code (301, 302, 303, 307, 308) |

**Examples:**

```bash
# Default 302 redirect
curl -i "http://localhost:80/redirect-to?url=https://example.com"

# 301 permanent redirect
curl -i "http://localhost:80/redirect-to?url=https://example.com&status_code=301"
```

### GET /absolute-redirect/{n}

Redirect n times using absolute URLs.

| Parameter | Type | Range | Description         |
| --------- | ---- | ----- | ------------------- |
| `n`       | int  | 0-100 | Number of redirects |

```bash
curl -L http://localhost:80/absolute-redirect/3
```

### GET /relative-redirect/{n}

Redirect n times using relative URLs.

| Parameter | Type | Range | Description         |
| --------- | ---- | ----- | ------------------- |
| `n`       | int  | 0-100 | Number of redirects |

```bash
curl -L http://localhost:80/relative-redirect/3
```

---

## Authentication Endpoints

### GET /basic-auth/{user}/{pass}

Validate Basic Authentication credentials. Returns 200 if credentials match, 401
otherwise.

| Parameter | Type   | Description       |
| --------- | ------ | ----------------- |
| `user`    | string | Expected username |
| `pass`    | string | Expected password |

**Request:**

```bash
curl -u testuser:testpass http://localhost:80/basic-auth/testuser/testpass
```

**Response (success):**

```json
{
  "authenticated": true,
  "user": "testuser"
}
```

**Response (failure):** 401 Unauthorized with `WWW-Authenticate: Basic` header.

### GET /hidden-basic-auth/{user}/{pass}

Similar to `/basic-auth` but returns 404 instead of 401 on authentication failure.
Useful for testing authentication without browser prompts.

| Parameter | Type   | Description       |
| --------- | ------ | ----------------- |
| `user`    | string | Expected username |
| `pass`    | string | Expected password |

```bash
curl -u testuser:testpass http://localhost:80/hidden-basic-auth/testuser/testpass
```

### GET /bearer

Validate Bearer token authentication. Returns 200 if a valid Bearer token is present,
401 otherwise.

**Request:**

```bash
curl -H "Authorization: Bearer my-token-123" http://localhost:80/bearer
```

**Response (success):**

```json
{
  "authenticated": true,
  "token": "my-token-123"
}
```

**Response (failure):** 401 Unauthorized with `WWW-Authenticate: Bearer` header.

---

## Cookie Endpoints

### GET /cookies

Echo all cookies sent with the request.

**Request:**

```bash
curl -b "session=abc123; user=john" http://localhost:80/cookies
```

**Response:**

```json
{
  "cookies": {
    "session": "abc123",
    "user": "john"
  }
}
```

### GET /cookies/set

Set cookies from query parameters and redirect to `/cookies`.

**Request:**

```bash
curl -c - -L "http://localhost:80/cookies/set?session=abc123&user=john"
```

Cookies are set with `Path=/`, `HttpOnly`, and `SameSite=Lax`.

### GET /cookies/delete

Delete cookies specified in query parameters and redirect to `/cookies`.

**Request:**

```bash
curl -b "session=abc123" -c - -L "http://localhost:80/cookies/delete?session"
```

---

## Data Generation Endpoints

### GET /bytes/{n}

Return n random bytes.

| Parameter | Type | Range    | Description     |
| --------- | ---- | -------- | --------------- |
| `n`       | int  | 0-102400 | Number of bytes |

**Request:**

```bash
# Get 100 random bytes
curl http://localhost:80/bytes/100 --output random.bin
```

**Response:** Binary data with `Content-Type: application/octet-stream`.

### GET /stream/{n}

Stream n lines of JSON data using chunked transfer encoding.

| Parameter | Type | Range | Description     |
| --------- | ---- | ----- | --------------- |
| `n`       | int  | 0-100 | Number of lines |

**Request:**

```bash
curl http://localhost:80/stream/5
```

**Response:** Newline-delimited JSON objects:

```json
{"id":0,"url":"/stream/5","args":{},"headers":{},"origin":"127.0.0.1"}
{"id":1,"url":"/stream/5","args":{},"headers":{},"origin":"127.0.0.1"}
{"id":2,"url":"/stream/5","args":{},"headers":{},"origin":"127.0.0.1"}
{"id":3,"url":"/stream/5","args":{},"headers":{},"origin":"127.0.0.1"}
{"id":4,"url":"/stream/5","args":{},"headers":{},"origin":"127.0.0.1"}
```

### GET /drip

Drip data byte-by-byte over a specified duration.

| Parameter  | Type  | Default | Range   | Description              |
| ---------- | ----- | ------- | ------- | ------------------------ |
| `duration` | float | 2       | 0-60    | Total duration (seconds) |
| `numbytes` | int   | 10      | 0-10240 | Number of bytes to drip  |
| `delay`    | float | 0       | 0-60    | Initial delay (seconds)  |

**Request:**

```bash
# Drip 20 bytes over 5 seconds
curl "http://localhost:80/drip?duration=5&numbytes=20"
```

**Response:** `*` characters streamed at regular intervals.

---

## Compression Endpoints

### GET /gzip

Return a gzip-compressed response.

**Request:**

```bash
curl --compressed http://localhost:80/gzip
```

**Response:**

```json
{
  "compressed": true,
  "method": "gzip",
  "origin": "127.0.0.1",
  "headers": {}
}
```

Response includes `Content-Encoding: gzip` header.

### GET /deflate

Return a deflate-compressed response.

**Request:**

```bash
curl --compressed http://localhost:80/deflate
```

**Response:**

```json
{
  "compressed": true,
  "method": "deflate",
  "origin": "127.0.0.1",
  "headers": {}
}
```

Response includes `Content-Encoding: deflate` header.

### GET /brotli

Return a brotli-compressed response.

**Request:**

```bash
curl --compressed http://localhost:80/brotli
```

**Response:**

```json
{
  "compressed": true,
  "method": "br",
  "origin": "127.0.0.1",
  "headers": {}
}
```

Response includes `Content-Encoding: br` header.

---

## Response Format

All echo endpoints return a JSON object with the following structure:

| Field     | Type   | Description                              |
| --------- | ------ | ---------------------------------------- |
| `method`  | string | HTTP method used                         |
| `url`     | string | Request URL including query string       |
| `args`    | object | Parsed query parameters                  |
| `headers` | object | Request headers                          |
| `data`    | string | Raw request body (POST/PUT/PATCH only)   |
| `json`    | object | Parsed JSON body (if Content-Type: json) |
| `form`    | object | Parsed form body (if Content-Type: form) |

## Error Responses

| Status | Description           |
| ------ | --------------------- |
| 400    | Invalid request       |
| 404    | Endpoint not found    |
| 405    | Method not allowed    |
| 500    | Internal server error |
