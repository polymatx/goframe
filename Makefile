export ROOT=$(realpath $(dir $(firstword $(MAKEFILE_LIST))))
export GO=$(shell which go)
export DOCKER=$(shell which docker)
export BINARY_NAME=goframe
export VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
export BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
export LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
.DEFAULT_GOAL := help

# Declare all phony targets
.PHONY: all build test test-coverage test-all quick-test bench clean fmt fmt-check vet lint tidy deps \
	docker-build docker-build-dev docker-up docker-down docker-logs install-cli check ci \
	run-basic run-rest-api run-database run-cache run-websocket run-rabbitmq run-mqtt \
	run-elasticsearch run-mongodb run-full-stack run-ioc help

all: check test

# Build
build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/goframe

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/goframe

build-darwin:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/goframe
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/goframe

# Testing
test:
	$(GO) test -v -race ./pkg/...

test-all:
	$(GO) test -v -race ./...

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./pkg/... ./internal/... ./cmd/...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

quick-test:
	$(GO) test -short ./pkg/...

bench:
	$(GO) test -bench=. -benchmem ./pkg/...

# Code quality
fmt:
	$(GO) fmt ./...

fmt-check:
	@test -z "$$($(GO) fmt ./...)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

vet:
	$(GO) vet ./...

lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run ./...

# Combined checks (useful for CI)
check: fmt-check vet

ci: deps check lint test-coverage

# Dependencies
tidy:
	$(GO) mod tidy

deps:
	$(GO) mod download

deps-verify:
	$(GO) mod verify

# Cleanup
clean:
	$(GO) clean -cache
	rm -f coverage.out coverage.html
	rm -rf bin/

# Docker
docker-build:
	$(DOCKER) build -f build/Dockerfile -t goframe:latest .

docker-build-dev:
	$(DOCKER) build -f build/Dockerfile.dev -t goframe:dev .

docker-up:
	$(DOCKER) compose -f build/docker-compose.yml up -d

docker-down:
	$(DOCKER) compose -f build/docker-compose.yml down

docker-logs:
	$(DOCKER) compose -f build/docker-compose.yml logs -f

# Install
install-cli:
	$(GO) install $(LDFLAGS) ./cmd/goframe

# Examples
run-basic:
	$(GO) run ./examples/basic/main.go

run-rest-api:
	$(GO) run ./examples/rest-api/main.go

run-database:
	$(GO) run ./examples/database/main.go

run-cache:
	$(GO) run ./examples/cache/main.go

run-websocket:
	$(GO) run ./examples/websocket-chat/main.go

run-rabbitmq:
	$(GO) run ./examples/rabbitmq/main.go

run-mqtt:
	$(GO) run ./examples/mqtt/main.go

run-elasticsearch:
	$(GO) run ./examples/elasticsearch/main.go

run-mongodb:
	$(GO) run ./examples/mongodb/main.go

run-full-stack:
	$(GO) run ./examples/full-stack/main.go

run-ioc:
	$(GO) run ./examples/ioc-container/main.go

# Help
help:
	@echo "GoFrame - A Simple Golang Framework"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build            - Build the CLI binary"
	@echo "  build-linux      - Build for Linux (amd64)"
	@echo "  build-darwin     - Build for macOS (amd64 + arm64)"
	@echo "  install-cli      - Install goframe CLI tool"
	@echo ""
	@echo "Testing:"
	@echo "  test             - Run tests (pkg only)"
	@echo "  test-all         - Run all tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  quick-test       - Run short tests only"
	@echo "  bench            - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt              - Format code"
	@echo "  fmt-check        - Check if code is formatted"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run golangci-lint"
	@echo "  check            - Run fmt-check and vet"
	@echo "  ci               - Run full CI pipeline (deps, check, lint, test-coverage)"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps             - Download dependencies"
	@echo "  deps-verify      - Verify dependencies"
	@echo "  tidy             - Tidy go.mod"
	@echo "  clean            - Clean build cache and artifacts"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-build-dev - Build development Docker image"
	@echo "  docker-up        - Start services (Postgres, Redis, MongoDB, RabbitMQ)"
	@echo "  docker-down      - Stop services"
	@echo "  docker-logs      - View service logs"
	@echo ""
	@echo "Examples:"
	@echo "  run-basic        - Basic HTTP server"
	@echo "  run-rest-api     - REST API with CRUD"
	@echo "  run-database     - Database operations (PostgreSQL)"
	@echo "  run-cache        - Redis caching"
	@echo "  run-websocket    - WebSocket chat"
	@echo "  run-rabbitmq     - RabbitMQ messaging"
	@echo "  run-mqtt         - MQTT pub/sub"
	@echo "  run-elasticsearch - Elasticsearch search"
	@echo "  run-mongodb      - MongoDB operations"
	@echo "  run-full-stack   - Full stack (DB + Cache + Auth)"
	@echo "  run-ioc          - IoC Container example"
