# echo-http API Reference

## Base URL

| Environment    | URL                      |
| -------------- | ------------------------ |
| Container      | `http://localhost:80`    |
| Docker Compose | `http://localhost:18080` |

> **Note:** The container listens on port 80. When using `docker compose up`, the
> port is mapped to 18080 on the host.

## Environment Variables

### Server Configuration

| Variable | Default   | Description  |
| -------- | --------- | ------------ |
| `HOST`   | `0.0.0.0` | Bind address |
| `PORT`   | `80`      | Listen port  |

### Authentication Configuration

Shared credentials used across all authentication methods.

| Variable                 | Default    | Description                                                    |
| ------------------------ | ---------- | -------------------------------------------------------------- |
| `AUTH_ALLOWED_USERNAME`  | `testuser` | Username for Basic Auth, Bearer Token, and OAuth2/OIDC flows   |
| `AUTH_ALLOWED_PASSWORD`  | `testpass` | Password for Basic Auth, Bearer Token, and OAuth2/OIDC flows   |

### OAuth2/OIDC Configuration

Configure OAuth2/OIDC server behavior with these environment variables:

**OAuth2 Configuration (shared across all flows):**

| Variable                     | Default                                                           | Description                                    |
| ---------------------------- | ----------------------------------------------------------------- | ---------------------------------------------- |
| `AUTH_ALLOWED_CLIENT_ID`     | (empty - accept any)                                              | Allowed client_id for validation (empty = any) |
| `AUTH_ALLOWED_CLIENT_SECRET` | (empty - public client)                                           | Required client_secret (empty = not required)  |
| `AUTH_SUPPORTED_SCOPES`      | `openid,profile,email`                                            | Comma-separated list of supported scopes       |
| `AUTH_TOKEN_EXPIRY`          | `3600`                                                            | Access token expiry in seconds                 |
| `AUTH_ALLOWED_GRANT_TYPES`   | `authorization_code,client_credentials,password,refresh_token`    | Comma-separated list of allowed grant types    |

**Authorization Code Flow Configuration:**

| Variable                          | Default             | Description                             |
| --------------------------------- | ------------------- | --------------------------------------- |
| `AUTH_CODE_REQUIRE_PKCE`          | `false`             | Require PKCE for all clients (RFC 8252) |
| `AUTH_CODE_SESSION_TTL`           | `300`               | Session timeout in seconds              |
| `AUTH_CODE_VALIDATE_REDIRECT_URI` | `false`             | Enable redirect_uri validation          |
| `AUTH_CODE_ALLOWED_REDIRECT_URIS` | (empty - allow all) | Comma-separated redirect URI patterns   |

**Example Configuration:**

```bash
# Strict validation for production-like testing
export AUTH_ALLOWED_CLIENT_ID=my-app-client-id
export AUTH_ALLOWED_CLIENT_SECRET=my-app-secret
export AUTH_SUPPORTED_SCOPES=openid,profile,email,custom_scope
export AUTH_TOKEN_EXPIRY=3600
export AUTH_ALLOWED_GRANT_TYPES=authorization_code,client_credentials,password,refresh_token
export AUTH_CODE_REQUIRE_PKCE=true
export AUTH_CODE_VALIDATE_REDIRECT_URI=true
export AUTH_CODE_ALLOWED_REDIRECT_URIS=http://localhost:*,https://myapp.com/callback
export AUTH_CODE_SESSION_TTL=300
```

### Redirect URI Patterns

When `AUTH_CODE_VALIDATE_REDIRECT_URI=true`, supports these patterns:

- **Exact match**: `http://localhost:8080/callback`
- **Wildcard port**: `http://localhost:*/callback` (any port)
- **Wildcard path**: `http://localhost:8080/*` (any path)
- **Multiple patterns**: Comma-separated list

---

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

### GET /basic-auth

Validate Basic Authentication credentials.

Configure credentials via environment variables:

- `AUTH_ALLOWED_USERNAME`: Expected username (default: `testuser`)
- `AUTH_ALLOWED_PASSWORD`: Expected password (default: `testpass`)

**Request:**

```bash
curl -u testuser:testpass http://localhost:80/basic-auth
```

**Response (success):**

```json
{
  "authenticated": true,
  "user": "testuser"
}
```

**Response (failure):** 401 Unauthorized with `WWW-Authenticate: Basic` header.

### GET /bearer-auth

Validate Bearer token authentication. The expected token is SHA1(username:password).

Configure credentials via environment variables:

- `AUTH_ALLOWED_USERNAME`: Username (default: `testuser`)
- `AUTH_ALLOWED_PASSWORD`: Password (default: `testpass`)

Generate the token:

```bash
echo -n "username:password" | shasum -a 1 | cut -d' ' -f1
```

**Request:**

```bash
curl -H "Authorization: Bearer <sha1-token>" http://localhost:80/bearer-auth
```

**Response (success):**

```json
{
  "authenticated": true,
  "token": "<sha1-token>"
}
```

**Response (failure):** 401 Unauthorized with `WWW-Authenticate: Bearer` header.

---

## OAuth2/OIDC Endpoints

A fully-featured OAuth2/OIDC test server for developing and testing OAuth2 and OIDC clients.
Implements OAuth 2.0 Authorization Framework and OpenID Connect Core 1.0 Authorization Code Flow
with support for PKCE, scope validation, and configurable client authentication.

All endpoints use environment-based authentication. See the [Environment Variables](#environment-variables)
section for configuration options.

### GET /.well-known/oauth-authorization-server

OAuth 2.0 Authorization Server Metadata endpoint (RFC 8414).

**Request:**

```bash
curl http://localhost:80/.well-known/oauth-authorization-server
```

**Response:**

```json
{
  "issuer": "http://localhost:80",
  "authorization_endpoint": "http://localhost:80/oauth2/authorize",
  "token_endpoint": "http://localhost:80/oauth2/token",
  "response_types_supported": ["code"],
  "grant_types_supported": ["authorization_code"],
  "code_challenge_methods_supported": ["plain", "S256"]
}
```

### GET /.well-known/openid-configuration

OpenID Connect Discovery endpoint (OIDC Discovery 1.0). Returns provider metadata
including endpoints, supported features, and capabilities.

**Request:**

```bash
curl http://localhost:80/.well-known/openid-configuration
```

**Response:**

```json
{
  "issuer": "http://localhost:80",
  "authorization_endpoint": "http://localhost:80/oauth2/authorize",
  "token_endpoint": "http://localhost:80/oauth2/token",
  "userinfo_endpoint": "http://localhost:80/oauth2/userinfo",
  "jwks_uri": "http://localhost:80/.well-known/jwks.json",
  "response_types_supported": ["code"],
  "subject_types_supported": ["public"],
  "id_token_signing_alg_values_supported": ["none"],
  "scopes_supported": ["openid", "profile", "email"],
  "grant_types_supported": ["authorization_code"],
  "code_challenge_methods_supported": ["plain", "S256"]
}
```

### GET /.well-known/jwks.json

JWKS (JSON Web Key Set) endpoint returning the public keys used to verify JWT signatures (RFC 7517).

**Request:**

```bash
curl http://localhost:80/.well-known/jwks.json
```

**Response:**

```json
{
  "keys": []
}
```

**Notes:**

- Returns an empty key set because this implementation uses `alg: "none"` (no signature)
- In a production OIDC provider, this would contain public keys in JWK format

### GET/POST /oauth2/authorize

OAuth2/OIDC authorization endpoint implementing Authorization Code Flow with full parameter validation.

**GET:** Display login form for user authentication
**POST:** Process credentials and generate authorization code

**GET Query Parameters:**

| Parameter               | Required | Description                                                          |
| ----------------------- | -------- | -------------------------------------------------------------------- |
| `client_id`             | **Yes**  | Client identifier (validated if `AUTH_ALLOWED_CLIENT_ID` configured) |
| `redirect_uri`          | **Yes**  | Callback URI (validated if `AUTH_CODE_VALIDATE_REDIRECT_URI=true`)   |
| `response_type`         | **Yes**  | Must be `code`                                                       |
| `scope`                 | No       | Space-separated scopes (default: all supported scopes)               |
| `state`                 | No       | CSRF protection token (recommended)                                  |
| `nonce`                 | No       | Replay attack protection (included in ID token)                      |
| `code_challenge`        | No       | PKCE code challenge (required if `AUTH_CODE_REQUIRE_PKCE=true`)      |
| `code_challenge_method` | No       | PKCE method: `plain` or `S256` (default: `plain`)                    |

**POST Form Parameters:**

- `username` (required): Must match `AUTH_ALLOWED_USERNAME`
- `password` (required): Must match `AUTH_ALLOWED_PASSWORD`

**Request:**

```bash
curl "http://localhost:80/oauth2/authorize?\
client_id=my-app&\
redirect_uri=http://localhost:8080/callback&\
response_type=code&\
scope=openid%20profile&\
state=random-csrf-token"
```

### GET /oauth2/callback

Display the authorization code and state received from the authorization server.

**Query Parameters:**

- `code`: Authorization code
- `state`: State parameter for validation

**Request:**

```bash
curl "http://localhost:80/oauth2/callback?code=abc123&state=xyz789"
```

### POST /oauth2/token

Token endpoint implementing OAuth 2.0 / OIDC token exchange.

**Form Parameters:**

| Parameter       | Required | Description                                                   |
| --------------- | -------- | ------------------------------------------------------------- |
| `grant_type`    | **Yes**  | Must be `authorization_code`                                  |
| `code`          | **Yes**  | Authorization code from authorize endpoint                    |
| `client_id`     | **Yes**  | Client identifier (validated if `AUTH_ALLOWED_CLIENT_ID` set) |
| `redirect_uri`  | **Yes**  | Must match the URI from authorization request                 |
| `client_secret` | No       | Required if `AUTH_ALLOWED_CLIENT_SECRET` is configured        |
| `code_verifier` | No       | PKCE verifier (required if `code_challenge` was provided)     |

**Request:**

```bash
curl -X POST http://localhost:80/oauth2/token \
  -d "grant_type=authorization_code" \
  -d "code=<authorization-code>" \
  -d "client_id=my-app" \
  -d "redirect_uri=http://localhost:8080/callback"
```

**Response:**

```json
{
  "access_token": "a1b2c3d4e5f6...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "r1e2f3r4e5s6h...",
  "id_token": "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJpc3MiOi...",
  "scope": "openid profile email"
}
```

### GET /oauth2/userinfo

UserInfo endpoint returning user profile information based on the access token (OIDC Core Section 5.3).

**Headers:**

- `Authorization` (required): Bearer token from token endpoint

**Request:**

```bash
curl -H "Authorization: Bearer <access-token>" \
  http://localhost:80/oauth2/userinfo
```

**Response:**

```json
{
  "sub": "user",
  "name": "user",
  "email": "user@example.com"
}
```

### GET /oauth2/demo

Interactive demonstration of the complete OAuth2/OIDC Authorization Code Flow.

**Usage:**

```bash
# Open in browser for interactive demo
open "http://localhost:80/oauth2/demo"
```

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
