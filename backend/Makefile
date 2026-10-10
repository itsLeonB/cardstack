.PHONY: help \
http \
http-hot \
job \
ingest-pokemon-asia \
host-images \
host-series-images \
presize-images \
embed-catalog \
collect-embeddings \
wire \
lint \
vulncheck \
mocks \
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
	@echo "  make ingest-pokemon-asia     - Ingest Pokémon Asia catalog data (manual, no scheduler)"
	@echo "  make host-images             - Copy images missing from R2 straight from the database (manual)"
	@echo "  make host-series-images      - Upload assets/series logos to R2 and set series.image_key (manual)"
	@echo "  make presize-images          - Resize Expansion Set covers to 128px WebP in R2 and set expansion_sets.cover_key (manual, needs vips)"
	@echo "  make embed-catalog           - Submit every card's hosted image without an embedding to Gemini's batch API (manual, rerunnable; ARGS=\"-set MA6\" for one set)"
	@echo "  make collect-embeddings      - Store the vectors of finished embedding batch jobs in card_embeddings (manual, rerunnable; jobs can take up to 24 hours)"
	@echo "  make wire                    - Regenerate internal/provider/wire_gen.go"
	@echo "  make lint                    - Run golangci-lint on the codebase"
	@echo "  make vulncheck               - Run govulncheck for known vulnerabilities"
	@echo "  make mocks                   - Regenerate mockery mocks (see .mockery.yaml)"
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

ingest-pokemon-asia:
	go run ./cmd/ingest-pokemon-asia

host-images:
	go run ./cmd/host-images

host-series-images:
	go run ./cmd/host-series-images

presize-images:
	go run ./cmd/presize-images

embed-catalog:
	go run ./cmd/embed-catalog $(ARGS)

collect-embeddings:
	go run ./cmd/collect-embeddings

wire:
	go generate ./internal/provider/...

lint:
	go vet ./...
	golangci-lint run ./... --timeout=5m

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

mocks:
	go run github.com/vektra/mockery/v2@v2.53.7

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
