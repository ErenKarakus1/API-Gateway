# Go API Gateway

A portfolio API Gateway built with Go and Gin. The project is designed to demonstrate practical backend infrastructure patterns such as reverse proxying, route configuration, authentication, rate limiting, observability, and containerized local development.

## Planned Features

- Config-driven route forwarding
- Health and readiness endpoints
- Request IDs and structured logging
- JWT authentication
- Role-based authorization
- Per-route rate limiting
- Upstream timeout and retry handling
- Prometheus metrics
- Docker Compose demo environment
- Gateway and middleware tests

## Getting Started

```bash
go run ./cmd/gateway
```

The gateway starts on `http://localhost:8080`.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```
