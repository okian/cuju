.PHONY: build run test clean fmt vet lint cover tidy generate help bench

# Variables
BINARY_NAME := cuju
BINARY_PATH := bin/$(BINARY_NAME)
GOCACHE ?= $(CURDIR)/.gocache
GO_FLAGS := env GOCACHE=$(GOCACHE)


# Build targets
build:
	$(GO_FLAGS) go build -o $(BINARY_PATH) cmd/main.go

# Run targets
run: build
	@echo "Starting CUJU leaderboard service..."
	@echo "Dashboard: http://localhost:9080/dashboard"
	@echo "Press Ctrl+C to stop"
	@(sleep 2 && \
		if command -v open >/dev/null 2>&1; then \
			open http://localhost:9080/dashboard; \
		elif command -v xdg-open >/dev/null 2>&1; then \
			xdg-open http://localhost:9080/dashboard; \
		elif command -v start >/dev/null 2>&1; then \
			start http://localhost:9080/dashboard; \
		else \
			echo "Please open http://localhost:9080/dashboard in your browser"; \
		fi) &
	@./$(BINARY_PATH)

run-dev:
	ADDR=:8080 QUEUE_SIZE=1024 WORKERS=4 $(GO_FLAGS) go run cmd/main.go

run-prod:
	ADDR=:8080 QUEUE_SIZE=4096 WORKERS=8 $(GO_FLAGS) go run cmd/main.go

# Test targets
test:
	$(GO_FLAGS) go test ./... -v

test-race:
	$(GO_FLAGS) go test ./... -race -v

cover:
	$(GO_FLAGS) go test ./... -coverprofile=coverage.out
	@echo "Coverage report written to coverage.out"

# Code quality targets
fmt:
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	goimports -w .

vet:
	$(GO_FLAGS) go vet ./...

lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

tidy:
	$(GO_FLAGS) go mod tidy

# Benchmark targets
bench:
	$(GO_FLAGS) go test ./... -bench=. -benchmem -run ^$$

bench-repo:
	$(GO_FLAGS) go test -bench="BenchmarkTreapStore" -benchmem ./internal/adapters/repository/...

# Generation targets
generate: build
	./$(BINARY_PATH) -cmd=gen-site
	./$(BINARY_PATH) -cmd=gen-openapi
	./$(BINARY_PATH) -cmd=fetch-redoc
	$(GO_FLAGS) go generate ./...


# Utility targets
clean:
	rm -rf bin/ .gocache/

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Run:"
	@echo "  build      - build the binary"
	@echo "  run        - build and run the server"
	@echo "  run-dev    - run with development settings"
	@echo "  run-prod   - run with production settings"
	@echo ""
	@echo "Testing:"
	@echo "  test       - run all tests"
	@echo "  test-race  - run tests with race detector"
	@echo "  cover      - generate coverage report"
	@echo "  bench      - run benchmarks"
	@echo "  bench-repo - run repository benchmarks"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt        - format code and organize imports"
	@echo "  vet        - run go vet"
	@echo "  lint       - run golangci-lint"
	@echo "  tidy       - update dependencies"
	@echo ""
	@echo "Generation:"
	@echo "  generate   - generate all assets"
	@echo ""
	@echo "Utilities:"
	@echo "  clean      - remove build artifacts"
	@echo "  help       - show this help"