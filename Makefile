# HTTP Remote Makefile

BINARY_NAME=http-remote
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
PKG=github.com/pandeptwidyaop/http-remote/internal/version
LDFLAGS=-ldflags="-s -w -X $(PKG).Version=$(VERSION) -X $(PKG).BuildTime=$(BUILD_TIME) -X $(PKG).GitCommit=$(GIT_COMMIT)"

.PHONY: all build clean run dev test lint help

# Default target
all: build

# Build binary
build:
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/server
	@echo "Build complete: $(BINARY_NAME)"

# Build with debug symbols
build-debug:
	@echo "Building $(BINARY_NAME) with debug symbols..."
	CGO_ENABLED=1 go build -o $(BINARY_NAME) ./cmd/server

# Build for Linux AMD64
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc \
		go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/server
	@echo "Build complete: $(BINARY_NAME)-linux-amd64"

# Build for Linux ARM64
build-linux-arm64:
	@echo "Building for Linux ARM64..."
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-musl-gcc \
		go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 ./cmd/server
	@echo "Build complete: $(BINARY_NAME)-linux-arm64"

# Build all platforms
build-all: build build-linux-amd64 build-linux-arm64
	@echo "All builds complete"

# Run the application
run: build
	./$(BINARY_NAME)

# Run with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint code (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Please install golangci-lint" && exit 1)
	golangci-lint run

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-linux-amd64
	rm -f $(BINARY_NAME)-linux-arm64
	rm -f coverage.out coverage.html
	rm -rf tmp/
	@echo "Clean complete"

# Docker build
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# Docker run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/data:/app/data $(BINARY_NAME):latest

# Install to /usr/local/bin (requires sudo)
install: build
	@echo "Installing to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installed: /usr/local/bin/$(BINARY_NAME)"

# Uninstall from /usr/local/bin (requires sudo)
uninstall:
	@echo "Uninstalling..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled"

# Show help
help:
	@echo "HTTP Remote - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build            Build binary for current platform"
	@echo "  build-debug      Build with debug symbols"
	@echo "  build-linux-amd64 Build for Linux AMD64"
	@echo "  build-linux-arm64 Build for Linux ARM64"
	@echo "  build-all        Build for all platforms"
	@echo ""
	@echo "Run:"
	@echo "  run              Build and run"
	@echo "  dev              Run with hot reload (requires air)"
	@echo ""
	@echo "Test:"
	@echo "  test             Run tests"
	@echo "  test-coverage    Run tests with coverage report"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt              Format code"
	@echo "  lint             Lint code (requires golangci-lint)"
	@echo "  tidy             Tidy go.mod"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build     Build Docker image"
	@echo "  docker-run       Run Docker container"
	@echo ""
	@echo "Install:"
	@echo "  install          Install to /usr/local/bin"
	@echo "  uninstall        Remove from /usr/local/bin"
	@echo ""
	@echo "Other:"
	@echo "  clean            Remove build artifacts"
	@echo "  help             Show this help"
