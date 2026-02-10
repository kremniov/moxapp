# MoxApp - Golang
# Build and run commands

.PHONY: all build run test clean docker docker-run help frontend frontend-dev frontend-build

# Variables
BINARY_NAME=moxapp
VERSION?=1.0.0
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: build

frontend-build:
	@echo "Building frontend..."
	npm --prefix frontend install
	npm --prefix frontend run build
	rm -rf internal/web/dist
	mkdir -p internal/web
	cp -r frontend/dist internal/web/dist
	@echo "Frontend build complete"

# Build the binary
build: frontend-build
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/moxapp
	@echo "Build complete: bin/$(BINARY_NAME)"

# Build for multiple platforms
build-all: frontend-build
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/moxapp
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/moxapp
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/moxapp
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/moxapp
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/moxapp
	@echo "Build complete for all platforms"

# Run the application
run: build
	./bin/$(BINARY_NAME)

# Run with dry-run flag
dry-run: build
	./bin/$(BINARY_NAME) --dry-run

# Run with custom multiplier
run-low: build
	./bin/$(BINARY_NAME) --multiplier=0.1 --yes

run-high: build
	./bin/$(BINARY_NAME) --multiplier=2.0 --yes

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run benchmarks
bench:
	go test -bench=. -benchmem ./...

# Download dependencies
deps:
	go mod download
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Run linter
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed"; exit 1; }
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker build
docker:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t moxapp:$(VERSION) \
		-t moxapp:latest \
		.

# Docker run
docker-run:
	docker run --rm -it \
		-p 8080:8080 \
		--env-file .env \
		moxapp:latest

# Docker compose up
up:
	docker-compose up --build

# Docker compose down
down:
	docker-compose down

# Docker compose logs
logs:
	docker-compose logs -f moxapp

# Validate configuration
validate: build
	./bin/$(BINARY_NAME) --validate

# Show configuration summary
config: build
	./bin/$(BINARY_NAME) --dry-run

# Install binary to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/moxapp

# Create .env file from template
env:
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "Created .env file from template"; \
	else \
		echo ".env file already exists"; \
	fi

# Help
help:
	@echo "MoxApp - Golang"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build the binary"
	@echo "  make build-all    Build for multiple platforms"
	@echo "  make run          Build and run"
	@echo "  make dry-run      Build and show config (no execution)"
	@echo "  make run-low      Run with 10% load"
	@echo "  make run-high     Run with 200% load"
	@echo "  make test         Run tests"
	@echo "  make test-coverage Run tests with coverage"
	@echo "  make bench        Run benchmarks"
	@echo "  make deps         Download dependencies"
	@echo "  make lint         Run linter"
	@echo "  make fmt          Format code"
	@echo "  make clean        Remove build artifacts"
	@echo "  make docker       Build Docker image"
	@echo "  make docker-run   Run Docker container"
	@echo "  make up           Docker compose up"
	@echo "  make down         Docker compose down"
	@echo "  make logs         View Docker compose logs"
	@echo "  make validate     Validate configuration"
	@echo "  make install      Install binary to GOPATH/bin"
	@echo "  make help         Show this help"
