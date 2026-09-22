.PHONY: help \
http \
http-hot \
job \
wire \
lint \
vulncheck \
test \
test-verbose \
test-coverage \
test-coverage-html \
test-clean \
build \
genspec

help:
	@echo "Makefile commands:"
	@echo "  make http                    - Start the HTTP server"
	@echo "  make http-hot                - Start the HTTP server with hot reload (requires air)"
	@echo "  make job                     - Run migrations (production)"
	@echo "  make wire                    - Regenerate internal/provider/wire_gen.go"
	@echo "  make lint                    - Run golangci-lint on the codebase"
	@echo "  make vulncheck               - Run govulncheck for known vulnerabilities"
	@echo "  make test                    - Run all tests"
	@echo "  make test-verbose            - Run all tests with verbose output"
	@echo "  make test-coverage           - Run all tests with coverage report"
	@echo "  make test-coverage-html      - Run all tests and generate HTML coverage report"
	@echo "  make test-clean              - Clean test cache and run tests"
	@echo "  make build                   - Build the API server for production"
	@echo "  make genspec                 - Generate openapi.json from the Huma routes"

http:
	go run ./cmd/api

http-hot:
	air --build.cmd "go build -o bin/api ./cmd/api" --build.bin "./bin/api"

job:
	go run ./cmd/job

wire:
	go generate ./internal/provider/...

lint:
	go vet ./...
	golangci-lint run ./... --timeout=5m

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

test:
	@echo "Running all tests..."
	go test -race ./...

test-verbose:
	@echo "Running all tests with verbose output..."
	go test -race -v ./...

test-coverage:
	@echo "Running all tests with coverage report..."
	go test -race -v -coverprofile=coverage.out -covermode=atomic ./...

test-coverage-html:
	@echo "Running all tests and generating HTML coverage report..."
	go test -race -v -coverprofile=coverage.out -covermode=atomic ./... && \
	go tool cover -html=coverage.out -o coverage.html && \
	echo "Coverage report generated: coverage.html"

test-clean:
	@echo "Cleaning test cache and running tests..."
	go clean -testcache && go test -race -v ./...

build:
	@echo "Building API server..."
	CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags='-w -s' -o bin/api ./cmd/api
	@echo "Build success! Binary is located at bin/api"

genspec:
	go run ./cmd/genspec
