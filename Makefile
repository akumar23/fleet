.PHONY: help build build-static build-all test test-unit test-integration test-bench lint clean install completion coverage

# Build variables
BINARY_NAME=fleet
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(shell go version | awk '{print $$3}')

# Directories
BUILD_DIR=bin
DIST_DIR=dist

# Go build variables
LDFLAGS=-ldflags "\
	-X github.com/aryankumar/fleet/pkg/version.Version=$(VERSION) \
	-X github.com/aryankumar/fleet/pkg/version.Commit=$(COMMIT) \
	-X github.com/aryankumar/fleet/pkg/version.BuildTime=$(BUILD_TIME) \
	-X github.com/aryankumar/fleet/pkg/version.GoVersion=$(GO_VERSION)"

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build binary for current platform
	@echo "Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/fleet
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-static: ## Build static binary with CGO disabled
	@echo "Building static binary..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/fleet
	@echo "Static build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-all: ## Cross-compile for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)

	# Linux amd64
	@echo "Building for linux/amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/fleet

	# Linux arm64
	@echo "Building for linux/arm64..."
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/fleet

	# Darwin amd64
	@echo "Building for darwin/amd64..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/fleet

	# Darwin arm64
	@echo "Building for darwin/arm64..."
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/fleet

	# Windows amd64
	@echo "Building for windows/amd64..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo \
		-o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/fleet

	# Windows arm64
	@echo "Building for windows/arm64..."
	@GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -a -installsuffix cgo \
		-o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/fleet

	@echo "Cross-compilation complete. Binaries in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

test: ## Run all tests with race detection
	@echo "Running tests with race detection..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	@go tool cover -func=coverage.out | tail -1

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	go test -v -race -short ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -race -run Integration ./internal/integration/...
	go test -v -race -run Integration ./internal/config/...

test-bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem -run=^$$ ./internal/executor/...
	go test -bench=. -benchmem -run=^$$ ./internal/cluster/...

coverage: test ## Generate coverage report
	@echo "Generating HTML coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run golangci-lint
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) coverage.out
	@echo "Clean complete"

install: build ## Install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME) to $(shell go env GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(shell go env GOPATH)/bin/
	@echo "Installation complete"

completion: build ## Generate shell completion scripts
	@echo "Generating shell completion scripts..."
	@mkdir -p completions
	@$(BUILD_DIR)/$(BINARY_NAME) completion bash > completions/fleet.bash
	@$(BUILD_DIR)/$(BINARY_NAME) completion zsh > completions/fleet.zsh
	@$(BUILD_DIR)/$(BINARY_NAME) completion fish > completions/fleet.fish
	@$(BUILD_DIR)/$(BINARY_NAME) completion powershell > completions/fleet.ps1
	@echo "Completion scripts generated in completions/"
	@echo ""
	@echo "To install completions:"
	@echo "  Bash:       sudo cp completions/fleet.bash /etc/bash_completion.d/fleet"
	@echo "  Zsh:        cp completions/fleet.zsh \$${fpath[1]}/_fleet"
	@echo "  Fish:       cp completions/fleet.fish ~/.config/fish/completions/fleet.fish"
	@echo "  PowerShell: See completions/fleet.ps1 for instructions"
