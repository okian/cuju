.PHONY: build run test clean fmt vet lint cover tidy generate race help bench swagger docker-build docker-run docker-stop docker-clean docker-logs gen-site gen-openapi fetch-redoc stress-test

# Use a workspace-local Go build cache when available (safe default)
GOCACHE ?= $(CURDIR)/.gocache

# Build the binary
build:
	env GOCACHE=$(GOCACHE) go build -o bin/cuju cmd/main.go

# Run the application (server mode by default)
run: build
	@echo "Starting CUJU leaderboard service..."
	@echo "Dashboard will open at: http://localhost:9080/dashboard"
	@echo "Press Ctrl+C to stop the service"
	@echo ""
	@# Start the service in background and open dashboard
	@./bin/cuju & \
	SERVER_PID=$$!; \
	sleep 2; \
	if command -v open >/dev/null 2>&1; then \
		open http://localhost:9080/dashboard; \
	elif command -v xdg-open >/dev/null 2>&1; then \
		xdg-open http://localhost:9080/dashboard; \
	elif command -v start >/dev/null 2>&1; then \
		start http://localhost:9080/dashboard; \
	else \
		echo "Please open http://localhost:9080/dashboard in your browser"; \
	fi; \
	wait $$SERVER_PID

# Run the application without opening browser
run-simple: build
	./bin/cuju

# Run specific commands
gen-site: build
	./bin/cuju -cmd=gen-site

gen-openapi: build
	./bin/cuju -cmd=gen-openapi

fetch-redoc: build
	./bin/cuju -cmd=fetch-redoc

# Run tests
test:
	env GOCACHE=$(GOCACHE) go test ./... -run . -count=1

# Run tests with race detector
race:
	env GOCACHE=$(GOCACHE) go test ./... -race -run . -count=1

# Clean build artifacts
clean:
	rm -rf bin/

# Run with specific configuration
run-dev:
	ADDR=:8080 QUEUE_SIZE=1024 WORKERS=4 go run cmd/main.go

# Run with production-like settings
run-prod:
	ADDR=:8080 QUEUE_SIZE=4096 WORKERS=8 go run cmd/main.go

# Format all Go files
fmt:
	go fmt ./...

# Static analysis
vet:
	env GOCACHE=$(GOCACHE) go vet ./...

# Lint if golangci-lint is installed (optional)
lint:
	@golangci-lint version >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

# Update module deps
tidy:
	env GOCACHE=$(GOCACHE) go mod tidy

# Code generation hook - generates site and fetches dependencies
generate: build gen-site gen-openapi fetch-redoc
	env GOCACHE=$(GOCACHE) go generate ./...

# Regenerate only swagger assets
swagger: gen-openapi fetch-redoc
	@echo "Swagger assets regenerated"

# Generate coverage profile
cover:
	env GOCACHE=$(GOCACHE) go test ./... -coverprofile=coverage.out
	@echo "wrote coverage.out"

# Run microbenchmarks across packages
bench:
	env GOCACHE=$(GOCACHE) go test ./... -bench=. -benchmem -run ^$$

# Run comprehensive repository benchmarks (30M talents)
bench-repo:
	env GOCACHE=$(GOCACHE) ./scripts/run_benchmarks.sh

# Run individual repository benchmarks
bench-repo-heavy:
	env GOCACHE=$(GOCACHE) go test -bench="BenchmarkTreapStore_30MTalents_HeavyLoad" -benchmem -benchtime=10m ./internal/adapters/repository/...

bench-repo-write:
	env GOCACHE=$(GOCACHE) go test -bench="BenchmarkTreapStore_30MTalents_WriteHeavy" -benchmem -benchtime=5m ./internal/adapters/repository/...

bench-repo-read:
	env GOCACHE=$(GOCACHE) go test -bench="BenchmarkTreapStore_30MTalents_ReadHeavy" -benchmem -benchtime=5m ./internal/adapters/repository/...

bench-repo-snapshot:
	env GOCACHE=$(GOCACHE) go test -bench="BenchmarkTreapStore_30MTalents_SnapshotImpact" -benchmem -benchtime=5m ./internal/adapters/repository/...

# Docker targets
docker-setup:
	./deployments/docker-setup.sh

docker-build:
	docker compose build

docker-run:
	docker compose up -d

docker-stop:
	docker compose down

docker-clean:
	docker compose down -v --remove-orphans
	docker system prune -f

docker-logs:
	docker compose logs -f

docker-restart:
	docker compose restart

# Stress testing targets
stress-test:
	./scripts/run_stress_tests.sh

# Show common targets
help:
	@echo "make build        - build the binary"
	@echo "make run          - build then run server (opens dashboard in browser)"
	@echo "make run-simple   - build then run server (no browser)"
	@echo "make gen-site     - generate documentation site"
	@echo "make gen-openapi  - copy OpenAPI spec to swagger package"
	@echo "make fetch-redoc  - fetch ReDoc JavaScript bundle"
	@echo "make test         - run tests"
	@echo "make race         - run tests with race detector"
	@echo "make fmt          - format code"
	@echo "make vet          - static analysis"
	@echo "make lint         - run golangci-lint if available"
	@echo "make tidy         - update go.mod/go.sum"
	@echo "make clean        - remove build artifacts"
	@echo "make cover        - generate coverage profile"
	@echo "make bench        - run benchmarks"
	@echo "make bench-repo   - run comprehensive repository benchmarks (30M talents)"
	@echo "make bench-repo-* - run individual repository benchmarks"
	@echo "make stress-test  - run comprehensive stress tests"
	@echo "make swagger      - regenerate swagger assets"
	@echo ""
	@echo "Docker targets:"
	@echo "make docker-setup - check Docker prerequisites"
	@echo "make docker-build - build Docker images"
	@echo "make docker-run   - start services"
	@echo "make docker-stop  - stop services"
	@echo "make docker-clean - stop and clean up volumes"
	@echo "make docker-logs  - show service logs"
	@echo "make docker-restart - restart services"
	@echo ""
	@echo "Testing targets:"
	@echo "make stress-test   - run comprehensive stress tests"
