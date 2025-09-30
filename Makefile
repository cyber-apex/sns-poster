# SNS Notify Makefile

# Variables
BINARY_NAME = sns-poster
BUILD_DIR = ./cmd

# Directories
BIN_DIR = ./bin
SCRIPT_DIR = ./scripts
LOG_DIR = /var/logs/sns-poster
SERVICE_DIR = /opt/sns-poster

.PHONY: all build build-dev clean test deps help dev run install uninstall start stop restart status logs test-api test-post

# Default target
all: clean deps test build

help: ## Show this help message
	@echo "SNS Poster - Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build for Linux AMD64
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(BUILD_DIR)
	@echo "Build completed: $(BIN_DIR)/$(BINARY_NAME)-linux-amd64"

build-dev: ## Build for development
	@echo "Building for development..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	@echo "Development build completed: $(BIN_DIR)/$(BINARY_NAME)"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@go clean
	@rm -rf $(BIN_DIR)
	@echo "Cleanup completed"

dev: build-dev ## Build and run for development
	@echo "Starting development server..."
	@$(BIN_DIR)/$(BINARY_NAME) -http-port=:6170

run: build-dev ## Build and run with default settings
	@echo "Starting SNS Poster..."
	@$(BIN_DIR)/$(BINARY_NAME) -http-port=:6170

install: build ## Install the service (requires sudo)
	@echo "Installing SNS Poster..."
	@sudo mkdir -p $(SERVICE_DIR) $(LOG_DIR)
	@sudo cp $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(SERVICE_DIR)/$(BINARY_NAME)
	@sudo chmod +x $(SERVICE_DIR)/$(BINARY_NAME)
	@if [ -f $(SCRIPT_DIR)/sns-poster.service ]; then \
		sudo cp $(SCRIPT_DIR)/sns-poster.service /etc/systemd/system/; \
		sudo systemctl daemon-reload; \
		echo "Service installed. Use 'make start' to start"; \
	else \
		echo "Service file not found at $(SCRIPT_DIR)/sns-poster.service"; \
	fi

uninstall: ## Uninstall the service (requires sudo)
	@echo "Uninstalling SNS Poster..."
	@sudo systemctl stop sns-poster.service 2>/dev/null || true
	@sudo systemctl disable sns-poster.service 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/sns-poster.service
	@sudo rm -rf $(SERVICE_DIR)
	@sudo systemctl daemon-reload
	@echo "SNS Poster uninstalled"

start: ## Start the systemd service
	@sudo systemctl start sns-poster.service
	@sudo systemctl status sns-poster.service

stop: ## Stop the systemd service
	@sudo systemctl stop sns-poster.service

restart: ## Restart the systemd service
	@sudo systemctl restart sns-poster.service
	@sudo systemctl status sns-poster.service

status: ## Check service status
	@sudo systemctl status sns-poster.service

logs: ## View service logs
	@sudo journalctl -u sns-poster.service -f

test-api: ## Test API endpoints
	@chmod +x $(SCRIPT_DIR)/test_qr_login.sh
	@$(SCRIPT_DIR)/test_qr_login.sh

test-post: ## Test posting functionality
	@chmod +x $(SCRIPT_DIR)/quick_test_post.sh
	@$(SCRIPT_DIR)/quick_test_post.sh