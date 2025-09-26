# SNS Notify Makefile

# Variables
BINARY_NAME = sns-notify
BUILD_DIR = ./cmd/sns-notify
SCRIPT_DIR = ./scripts
LOG_DIR = /var/logs/sns-notify
SERVICE_DIR = /opt/sns-notify

.PHONY: all build build-dev clean test deps help dev run install uninstall start stop restart status logs test-api test-post

# Default target
all: clean deps test build

help: ## Show this help message
	@echo "SNS Notify - Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build for Linux AMD64
	@echo "Building $(BINARY_NAME) for linux/amd64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME)-linux-amd64 $(BUILD_DIR)
	@echo "Build completed: $(BINARY_NAME)-linux-amd64"

build-dev: ## Build for development
	@echo "Building for development..."
	@go build -o $(BINARY_NAME) $(BUILD_DIR)
	@echo "Development build completed"

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
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	@echo "Cleanup completed"

dev: build-dev ## Build and run for development
	@echo "Starting development server..."
	@./$(BINARY_NAME) -http-port=:6170

run: build-dev ## Build and run with default settings
	@echo "Starting SNS Notify..."
	@./$(BINARY_NAME) -http-port=:6170

install: build ## Install the service (requires sudo)
	@echo "Installing SNS Notify..."
	@sudo mkdir -p $(SERVICE_DIR) $(LOG_DIR)
	@sudo cp $(BINARY_NAME)-linux-amd64 $(SERVICE_DIR)/$(BINARY_NAME)
	@sudo chmod +x $(SERVICE_DIR)/$(BINARY_NAME)
	@if [ -f $(SCRIPT_DIR)/sns-notify.service ]; then \
		sudo cp $(SCRIPT_DIR)/sns-notify.service /etc/systemd/system/; \
		sudo systemctl daemon-reload; \
		echo "Service installed. Use 'make start' to start"; \
	else \
		echo "Service file not found at $(SCRIPT_DIR)/sns-notify.service"; \
	fi

uninstall: ## Uninstall the service (requires sudo)
	@echo "Uninstalling SNS Notify..."
	@sudo systemctl stop sns-notify.service 2>/dev/null || true
	@sudo systemctl disable sns-notify.service 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/sns-notify.service
	@sudo rm -rf $(SERVICE_DIR)
	@sudo systemctl daemon-reload
	@echo "SNS Notify uninstalled"

start: ## Start the systemd service
	@sudo systemctl start sns-notify.service
	@sudo systemctl status sns-notify.service

stop: ## Stop the systemd service
	@sudo systemctl stop sns-notify.service

restart: ## Restart the systemd service
	@sudo systemctl restart sns-notify.service
	@sudo systemctl status sns-notify.service

status: ## Check service status
	@sudo systemctl status sns-notify.service

logs: ## View service logs
	@sudo journalctl -u sns-notify.service -f

test-api: ## Test API endpoints
	@chmod +x $(SCRIPT_DIR)/test_qr_login.sh
	@$(SCRIPT_DIR)/test_qr_login.sh

test-post: ## Test posting functionality
	@chmod +x $(SCRIPT_DIR)/quick_test_post.sh
	@$(SCRIPT_DIR)/quick_test_post.sh