COMPOSE ?= docker compose
GO_ENV ?= GOCACHE=$(CURDIR)/.gocache GOPATH=$(CURDIR)/.gopath
LINT_ENV ?= $(GO_ENV) GOLANGCI_LINT_CACHE=$(CURDIR)/.golangci-cache
LOCAL_DB_ENV ?= DB_HOST=localhost DB_PORT=5432 DB_NAME=room_booking DB_USER=room_booking DB_PASSWORD=room_booking

.PHONY: up down down-v logs test coverage migrate seed lint swagger

up:
	$(COMPOSE) up --build

down:
	$(COMPOSE) down

down-v:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f service db

test:
	$(GO_ENV) go test ./...

coverage:
	$(GO_ENV) go test ./internal/... -coverprofile=internal_coverage.out
	$(GO_ENV) go tool cover -func=internal_coverage.out

migrate:
	$(LOCAL_DB_ENV) $(GO_ENV) go run ./cmd/migrate

seed:
	$(LOCAL_DB_ENV) $(GO_ENV) go run ./cmd/seed

lint:
	$(LINT_ENV) golangci-lint run

swagger:
	@echo "Swagger UI: http://localhost:8080/swagger"
