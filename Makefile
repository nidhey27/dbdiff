# Makefile for dbdiff - Database Schema Comparison Tool

# Binary name
BINARY_NAME=dbdiff

# Version (can be overridden: make VERSION=1.2.3)
VERSION?=dev

# Go command (auto-detect or use system go)
GO := $(shell which go 2>/dev/null || echo "/usr/local/go/bin/go")

# Build flags
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

# Output directory
BUILD_DIR=build

.PHONY: all build clean test install build-all help

# Default target
all: build

## build: Build binary for current platform
build:
	@echo "Building $(BINARY_NAME) for current platform..."
	$(GO) build $(LDFLAGS) -o $(BINARY_NAME) main.go
	@echo "✓ Built: $(BINARY_NAME)"

## build-all: Build binaries for all platforms
build-all: clean
	@echo "Building binaries for all platforms..."
	@mkdir -p $(BUILD_DIR)

	@echo "Building Linux AMD64..."
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go

	@echo "Building Linux ARM64..."
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go

	@echo "Building Windows AMD64..."
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go

	@echo "Building Windows ARM64..."
	GOOS=windows GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe main.go

	@echo "Building macOS AMD64 (Intel)..."
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go

	@echo "Building macOS ARM64 (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go

	@echo "✓ All binaries built in $(BUILD_DIR)/"
	@ls -lh $(BUILD_DIR)/

## checksums: Generate SHA256 checksums for all binaries
checksums: build-all
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && sha256sum $(BINARY_NAME)-* > checksums.txt
	@echo "✓ Checksums saved to $(BUILD_DIR)/checksums.txt"
	@cat $(BUILD_DIR)/checksums.txt

## install: Install binary to /usr/local/bin (requires sudo)
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✓ Installed: /usr/local/bin/$(BINARY_NAME)"

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test -v ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "✓ Clean complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy
	@echo "✓ Dependencies ready"

## run: Build and run with example (requires databases)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --help

## release: Create a release build with version
release:
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "Error: VERSION must be set for release builds"; \
		echo "Usage: make release VERSION=1.0.0"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@$(MAKE) build-all VERSION=$(VERSION)
	@$(MAKE) checksums VERSION=$(VERSION)
	@echo "✓ Release $(VERSION) ready in $(BUILD_DIR)/"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help

