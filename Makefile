.PHONY: build test clean lint fmt ci coverage install man install-man help

# Build variables
BINARY_NAME=replbac
BUILD_DIR=bin
CMD_DIR=cmd/replbac
MAN_DIR=man
MAN_FILE=$(MAN_DIR)/$(BINARY_NAME).1

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		echo "Running goimports..."; \
		goimports -w .; \
	else \
		echo "goimports not found, skipping import organization"; \
	fi

lint: ## Run linter
	@echo "Running linter..."
	@go vet ./...

install: build ## Install binary locally
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

man: ## Generate man page
	@echo "Generating man page..."
	@mkdir -p $(MAN_DIR)
	@go run cmd/gen-man/main.go --output $(MAN_FILE)
	@echo "Man page generated at $(MAN_FILE)"

install-man: man ## Install man page to system
	@echo "Installing man page..."
	@sudo mkdir -p /usr/local/share/man/man1
	@sudo cp $(MAN_FILE) /usr/local/share/man/man1/
	@echo "Man page installed to /usr/local/share/man/man1/"

coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

ci: fmt lint test ## Run all CI checks locally
	@echo "All CI checks passed!"

.DEFAULT_GOAL := help