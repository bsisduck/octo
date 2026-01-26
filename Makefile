# Makefile for Octo

.PHONY: all build clean install uninstall test lint fmt release

# Build configuration
BINARY_NAME := octo
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Build flags
LDFLAGS := -s -w \
	-X 'github.com/kamilkrawczyk/octo/cmd.Version=$(VERSION)' \
	-X 'github.com/kamilkrawczyk/octo/cmd.BuildTime=$(BUILD_TIME)' \
	-X 'github.com/kamilkrawczyk/octo/cmd.GitCommit=$(GIT_COMMIT)'

# Output directory
BIN_DIR := bin
INSTALL_DIR := /usr/local/bin

# Detect OS and architecture
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)

all: build

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Build for current platform
build: deps
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "Built: $(BIN_DIR)/$(BINARY_NAME)"

# Build with debug symbols
build-debug: deps
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags="-X 'github.com/kamilkrawczyk/octo/cmd.Version=$(VERSION)'" -o $(BIN_DIR)/$(BINARY_NAME) .

# Run the application
run: build
	./$(BIN_DIR)/$(BINARY_NAME)

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Lint the code
lint:
	@echo "Linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Verify code formatting
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

# Install locally
install: build
	@echo "Installing to $(INSTALL_DIR)..."
	@sudo cp $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installed: $(INSTALL_DIR)/$(BINARY_NAME)"
	@echo "Creating 'oc' alias..."
	@sudo ln -sf $(INSTALL_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/oc
	@echo "Installed: $(INSTALL_DIR)/oc"

# Uninstall
uninstall:
	@echo "Uninstalling..."
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo rm -f $(INSTALL_DIR)/oc
	@echo "Uninstalled $(BINARY_NAME)"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html

# Release builds for multiple platforms
release: clean deps
	@echo "Building release binaries..."
	@mkdir -p $(BIN_DIR)

	@echo "Building for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-amd64 .

	@echo "Building for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-darwin-arm64 .

	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 .

	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-linux-arm64 .

	@echo "Building for windows/amd64..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe .

	@echo "Release binaries built in $(BIN_DIR)/"
	@ls -la $(BIN_DIR)/

# Create checksums for release
checksums:
	@echo "Creating checksums..."
	@cd $(BIN_DIR) && sha256sum * > checksums.txt
	@cat $(BIN_DIR)/checksums.txt

# Docker build (for testing in container)
docker-build:
	docker build -t octo:dev .

# Help
help:
	@echo "Octo Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build for current platform"
	@echo "  make build        Build for current platform"
	@echo "  make run          Build and run"
	@echo "  make install      Install to /usr/local/bin"
	@echo "  make uninstall    Remove from /usr/local/bin"
	@echo "  make test         Run tests"
	@echo "  make lint         Run linter"
	@echo "  make fmt          Format code"
	@echo "  make release      Build for all platforms"
	@echo "  make clean        Clean build artifacts"
	@echo "  make help         Show this help"
