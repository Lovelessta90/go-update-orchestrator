.PHONY: help build test test-verbose test-integration test-unit bench clean lint fmt vet install run-orchestrator run-server

# Default target
help:
	@echo "Go Update Orchestrator - Makefile Commands"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build              - Build all binaries"
	@echo "  make install            - Install binaries to GOPATH/bin"
	@echo "  make clean              - Remove build artifacts"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test               - Run all tests"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make test-unit          - Run unit tests only"
	@echo "  make test-integration   - Run integration tests only"
	@echo "  make bench              - Run benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint               - Run linters"
	@echo "  make fmt                - Format code"
	@echo "  make vet                - Run go vet"
	@echo ""
	@echo "Run Commands:"
	@echo "  make run-orchestrator   - Run orchestrator CLI"
	@echo "  make run-server         - Run HTTP server"

# Build targets
build: build-orchestrator build-server

build-orchestrator:
	@echo "Building orchestrator..."
	@go build -o bin/orchestrator ./cmd/orchestrator

build-server:
	@echo "Building server..."
	@go build -o bin/server ./cmd/server

install:
	@echo "Installing binaries..."
	@go install ./cmd/orchestrator
	@go install ./cmd/server

# Test targets
test:
	@echo "Running all tests..."
	@go test -race -cover ./...

test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -race -cover ./...

test-unit:
	@echo "Running unit tests..."
	@go test -race -cover -short ./...

test-integration:
	@echo "Running integration tests..."
	@go test -race -cover -run Integration ./testing/integration/...

bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Code quality targets
lint:
	@echo "Running linters..."
	@go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

# Run targets
run-orchestrator:
	@go run ./cmd/orchestrator

run-server:
	@go run ./cmd/server

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean -cache -testcache
