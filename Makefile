.PHONY: help demo build test test-verbose test-integration test-unit bench clean lint fmt vet install ui-compile

# Default target
help:
	@echo "Go Update Orchestrator - Makefile Commands"
	@echo ""
	@echo "Quick Start:"
	@echo "  make demo               - Run the demo application with web UI (http://localhost:8081)"
	@echo "  make test               - Run all tests"
	@echo ""
	@echo "Library Usage:"
	@echo "  This is primarily a Go library. Import it in your project:"
	@echo "    go get github.com/dovaclean/go-update-orchestrator"
	@echo "  See examples/ directory for usage examples"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build              - Build demo binary"
	@echo "  make install            - Install Go dependencies"
	@echo "  make clean              - Remove build artifacts and databases"
	@echo ""
	@echo "Test Commands:"
	@echo "  make test               - Run all tests (without race detector)"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make test-unit          - Run unit tests only (fast)"
	@echo "  make test-integration   - Run integration tests only"
	@echo ""
	@echo "Benchmark Commands:"
	@echo "  make bench              - Run benchmarks"
	@echo "  make bench-baseline     - Run baseline benchmarks (3s, save to file)"
	@echo "  make bench-compare      - Compare current vs baseline (requires benchstat)"
	@echo ""
	@echo "Code Quality:"
	@echo "  make lint               - Run linters"
	@echo "  make fmt                - Format code"
	@echo "  make vet                - Run go vet"
	@echo ""
	@echo "UI Development (Optional):"
	@echo "  make ui-compile         - Compile TypeScript to JavaScript"
	@echo "                            (Only needed if you modify .ts files)"

# Demo target
demo:
	@echo "Starting Go Update Orchestrator Demo..."
	@echo "Web UI will be available at: http://localhost:8081"
	@echo ""
	@echo "Features:"
	@echo "  - Dashboard: Real-time device and update statistics"
	@echo "  - Devices:   View all registered devices"
	@echo "  - Updates:   Monitor update progress with live progress bars"
	@echo ""
	@echo "Press Ctrl+C to stop"
	@echo ""
	@go run cmd/demo/main.go

# Build targets
build:
	@echo "Building demo binary..."
	@mkdir -p bin
	@go build -o bin/orchestrator-demo cmd/demo/main.go
	@echo "✓ Binary created: bin/orchestrator-demo"
	@echo ""
	@echo "Run with: ./bin/orchestrator-demo"

install:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies installed"

# Test targets
test:
	@echo "Running all tests..."
	@go test -cover ./...

test-race:
	@echo "Running tests with race detector..."
	@echo "⚠ Note: Benign data races may be detected in shared payload readers."
	@echo "  These are mutex-protected and safe in production."
	@go test -race -cover ./...

test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -cover ./...

test-unit:
	@echo "Running unit tests..."
	@go test -cover -short ./...

test-integration:
	@echo "Running integration tests..."
	@go test -cover -run Integration ./testing/integration/...

bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

bench-baseline:
	@echo "Running baseline benchmarks (3 second runs)..."
	@echo "Results will be saved to baseline_bench.txt"
	@go test -bench=. -benchmem -benchtime=3s ./pkg/delivery/http/ | tee baseline_bench.txt

bench-compare:
	@echo "Running benchmarks for comparison..."
	@echo "Comparing against baseline_bench.txt"
	@go test -bench=. -benchmem -benchtime=3s ./pkg/delivery/http/ > current_bench.txt
	@if command -v benchstat >/dev/null 2>&1; then \
		benchstat baseline_bench.txt current_bench.txt; \
	else \
		echo ""; \
		echo "benchstat not installed. Install with:"; \
		echo "  go install golang.org/x/perf/cmd/benchstat@latest"; \
		echo ""; \
		echo "Current results saved to current_bench.txt"; \
	fi

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

# UI Development
ui-compile:
	@echo "Compiling TypeScript to JavaScript..."
	@command -v npx >/dev/null 2>&1 || { echo "⚠ npx not found. Install Node.js to compile TypeScript."; exit 1; }
	@cd web/static/js && npx -y -p typescript tsc --target ES2020 --module ES2020 --moduleResolution bundler types.ts dashboard.ts devices.ts updates.ts
	@echo "✓ TypeScript compilation complete"
	@echo ""
	@echo "Note: Compiled .js files are already included in the repository."
	@echo "      This is only needed if you modify the .ts source files."

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f orchestrator.db
	@rm -f *.txt
	@go clean -cache -testcache
	@echo "✓ Clean complete"
