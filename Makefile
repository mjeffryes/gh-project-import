# GitHub Project Import Extension Makefile
# Provides common development tasks and build targets

# Variables
BINARY_NAME = gh-project-import
BUILD_DIR = build
SNAPSHOT_DIR = testdata/snapshots
GO_FILES = $(shell find . -name '*.go' -not -path './$(BUILD_DIR)/*')
VERSION = $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Tool paths
GOTESTSUM = $(shell which gotestsum 2>/dev/null || echo "$(shell go env GOPATH)/bin/gotestsum")
TEST_CMD = $(shell if [ -x "$(GOTESTSUM)" ]; then echo "$(GOTESTSUM) --format testname"; else echo "go test -v"; fi)

# Build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)"

# Default target
.PHONY: help
help: ## Show this help message
	@echo "GitHub Project Import Extension"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
.PHONY: build
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

.PHONY: build-all
build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

.PHONY: install
install: build ## Build and install to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

# Testing targets
.PHONY: install-gotestsum
install-gotestsum: ## Install gotestsum for better test output
	@echo "Installing gotestsum..."
	go install gotest.tools/gotestsum@latest

.PHONY: test
test: ## Run all tests with gotestsum
	@echo "Running tests..."
	@if [ ! -x "$(GOTESTSUM)" ]; then echo "Note: Install gotestsum with 'make install-gotestsum' for better test output"; fi
	$(TEST_CMD) ./...

.PHONY: test-unit
test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	$(TEST_CMD) $(if $(findstring gotestsum,$(TEST_CMD)),-- -run "^Test[^S]", -run "^Test[^S]") ./...

# Snapshot management
.PHONY: test-record-snapshots
test-record-snapshots: ## Record new snapshots from real API calls
	@echo "Recording new snapshots..."
	@echo "Warning: This will make real GitHub API calls!"
	SNAPSHOT_MODE=record $(TEST_CMD) ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(TEST_CMD) $(if $(findstring gotestsum,$(TEST_CMD)),-- -coverprofile=coverage.out, -coverprofile=coverage.out) ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Targets that don't correspond to files
.PHONY: all
all: clean build ## Clean, check, and build

# Default goal
.DEFAULT_GOAL := help
