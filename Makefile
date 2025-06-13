.PHONY: build test clean lint install man install-man help

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

lint: ## Run linter
	@echo "Running linter..."
	@go vet ./...
	@go fmt ./...

install: build ## Install binary locally
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

man: build ## Generate man page
	@echo "Generating man page..."
	@mkdir -p $(MAN_DIR)
	@$(BUILD_DIR)/$(BINARY_NAME) man --output $(MAN_FILE)
	@echo "Man page generated at $(MAN_FILE)"

install-man: man ## Install man page to system
	@echo "Installing man page..."
	@sudo mkdir -p /usr/local/share/man/man1
	@sudo cp $(MAN_FILE) /usr/local/share/man/man1/
	@echo "Man page installed to /usr/local/share/man/man1/"

.DEFAULT_GOAL := help