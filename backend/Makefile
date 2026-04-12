APP_NAME := taskflow-api
GOCACHE_DIR := $(CURDIR)/.cache/go-build

.PHONY: tidy test test-integration build run dev

tidy:
	go mod tidy

test:
	@mkdir -p $(GOCACHE_DIR)
	GOCACHE=$(GOCACHE_DIR) go test ./...

test-integration:
	@mkdir -p $(GOCACHE_DIR)
	GOCACHE=$(GOCACHE_DIR) go test ./tests -v

build:
	@mkdir -p $(GOCACHE_DIR)
	GOCACHE=$(GOCACHE_DIR) go build -o $(APP_NAME) ./cmd/server

run:
	@mkdir -p $(GOCACHE_DIR)
	GOCACHE=$(GOCACHE_DIR) go run ./cmd/server

dev:
	@if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "air is not installed, so hot reload is unavailable."; \
		echo "Install it with: go install github.com/air-verse/air@latest"; \
		echo "Falling back to: go run ./cmd/server"; \
		mkdir -p $(GOCACHE_DIR); \
		GOCACHE=$(GOCACHE_DIR) go run ./cmd/server; \
	fi
