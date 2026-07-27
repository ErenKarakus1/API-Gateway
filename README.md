# Go API Gateway

A portfolio API Gateway built with Go and Gin. The project demonstrates practical backend infrastructure patterns: config-driven reverse proxying, JWT authentication, role-based authorization, rate limiting, timeout handling, structured logs, Prometheus metrics, and a Docker Compose demo environment.

## Features

- Config-driven route forwarding
- Health and readiness endpoints
- Request IDs and structured logging
- JWT authentication
- Role-based authorization
- Per-route rate limiting
- Upstream timeout handling
- Prometheus metrics
- Docker Compose demo environment
- Gateway and middleware tests

## Architecture

```text
Client
  |
  v
Go API Gateway
  |-- request id middleware
  |-- structured logging
  |-- prometheus metrics
  |-- JWT authentication
  |-- role authorization
  |-- per-route rate limiting
  |-- reverse proxy with timeout handling
  |
  v
Upstream services
```

The gateway reads route definitions from YAML and registers matching Gin routes at startup. Each configured route can define its upstream URL, allowed methods, auth requirements, roles, rate limit, and timeout.

## Getting Started

```bash
go run ./cmd/gateway
```

The gateway starts on `http://localhost:8080`.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/metrics
```

## Docker Demo

```bash
docker compose up --build
```

The Compose demo starts the gateway on `http://localhost:8080` and a mock users service behind it.

## Configuration

The default config lives at `config/gateway.yaml`.

```yaml
server:
  port: "8080"

auth:
  jwt_secret: "change-me-in-production"

routes:
  - id: users-service
    path: /api/users
    upstream: http://localhost:9001
    auth_required: true
    roles:
      - user
      - admin
    rate_limit:
      requests: 60
      window: 1m
    timeout: 3s
    methods:
      - GET
      - POST
```

Use `CONFIG_PATH` to load a different config file:

```bash
CONFIG_PATH=config/gateway.compose.yaml go run ./cmd/gateway
```

## Protected Routes

Protected routes expect a JWT signed with the configured `auth.jwt_secret`. The token subject is forwarded to upstream services as `X-User-ID`, and roles are read from a `roles` claim.

Example payload:

```json
{
  "sub": "user-123",
  "roles": ["user"],
  "exp": 1893456000
}
```

## Endpoints

- `GET /health`: liveness check
- `GET /ready`: readiness check
- `GET /metrics`: Prometheus metrics
- Configured gateway routes, such as `GET /api/users`

## Development

Run the test suite with local caches:

```bash
GOCACHE=.gocache GOMODCACHE=.gomodcache go test ./...
```
