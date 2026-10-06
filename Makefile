.PHONY: dev build test lint docker-build docker-run migrate up down logs

COMPOSE ?= docker compose

dev:
	$(COMPOSE) up --build

build:
	$(COMPOSE) build

test:
	$(COMPOSE) run --rm --no-deps -e CGO_ENABLED=1 \
		-v "$(PWD)/backend:/app/backend" -w /app/backend \
		golang:1.23-alpine sh -c "apk add --no-cache gcc musl-dev git && go test ./..."
	$(COMPOSE) run --rm --no-deps \
		-v "$(PWD)/frontend:/app/frontend" -w /app/frontend \
		node:20-alpine sh -c "npm ci || npm install && npm test"

lint:
	$(COMPOSE) run --rm --no-deps \
		-v "$(PWD)/frontend:/app/frontend" -w /app/frontend \
		node:20-alpine sh -c "npm ci || npm install && npm run lint"
	$(COMPOSE) run --rm --no-deps -e CGO_ENABLED=1 \
		-v "$(PWD)/backend:/app/backend" -w /app/backend \
		golang:1.23-alpine sh -c "apk add --no-cache gcc musl-dev git && go vet ./..."

docker-build:
	$(COMPOSE) build

docker-run:
	$(COMPOSE) up -d

migrate:
	@echo "Migrations run automatically on application startup (GORM AutoMigrate)."

up: docker-run

down:
	$(COMPOSE) down

logs:
	$(COMPOSE) logs -f
