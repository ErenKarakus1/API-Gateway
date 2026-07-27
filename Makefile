GOCACHE ?= .gocache
GOMODCACHE ?= .gomodcache
CONFIG_PATH ?= config/gateway.yaml
JWT_SECRET ?= change-me-in-production
JWT_SUB ?= user-123
JWT_ROLES ?= user

.PHONY: test run token docker-up docker-down compose-config demo-health demo-users

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) go test ./...

run:
	CONFIG_PATH=$(CONFIG_PATH) go run ./cmd/gateway

token:
	go run ./cmd/token -secret $(JWT_SECRET) -sub $(JWT_SUB) -roles $(JWT_ROLES)

docker-up:
	docker compose up --build

docker-down:
	docker compose down

compose-config:
	docker compose config

demo-health:
	curl http://localhost:8080/health

demo-users:
	curl -H "Authorization: Bearer $(TOKEN)" http://localhost:8080/api/users
