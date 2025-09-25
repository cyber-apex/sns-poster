# SNS Notify Makefile
# A multi-platform social media publishing tool

# Variables
BINARY_NAME = sns-notify
PACKAGE = sns-notify
MODULE = sns-notify
BUILD_DIR = ./cmd/sns-notify
GOOS = linux
GOARCH = amd64

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
GOVET = $(GOCMD) vet
GOLINT = golangci-lint

# Build flags
BUILD_FLAGS = -ldflags="-s -w"
BUILD_TAGS = 

# Directories
SCRIPT_DIR = ./scripts
LOG_DIR = /var/logs/sns-notify
SERVICE_DIR = /opt/sns-notify

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

.PHONY: all build clean test coverage deps help dev run install uninstall docker

# Default target
all: clean deps test build

# Help target
help: ## Show this help message
	@echo "$(BLUE)SNS Notify - Multi-platform Social Media Publishing Tool$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the application for current platform
	@echo "$(YELLOW)Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)...$(NC)"
	@CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) $(BUILD_DIR)
	@echo "$(GREEN)✅ Build completed: $(BINARY_NAME)$(NC)"

build-linux: ## Build for Linux AMD64
	@$(MAKE) build GOOS=linux GOARCH=amd64
	@mv $(BINARY_NAME) $(BINARY_NAME)-linux-amd64
	@echo "$(GREEN)✅ Linux build completed: $(BINARY_NAME)-linux-amd64$(NC)"

build-dev: ## Build for development (current platform)
	@echo "$(YELLOW)Building for development...$(NC)"
	@$(GOBUILD) -o $(BINARY_NAME) $(BUILD_DIR)
	@echo "$(GREEN)✅ Development build completed$(NC)"

build-all: ## Build using the build script
	@echo "$(YELLOW)Running build script...$(NC)"
	@chmod +x $(SCRIPT_DIR)/build.sh
	@$(SCRIPT_DIR)/build.sh

# Test targets
test: ## Run tests
	@echo "$(YELLOW)Running tests...$(NC)"
	@$(GOTEST) -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	@$(GOTEST) -v -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Coverage report generated: coverage.html$(NC)"

test-race: ## Run tests with race detector
	@echo "$(YELLOW)Running tests with race detector...$(NC)"
	@$(GOTEST) -race -v ./...

# Quality assurance
lint: ## Run linter
	@echo "$(YELLOW)Running linter...$(NC)"
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "$(YELLOW)golangci-lint not installed, running go vet instead$(NC)"; \
		$(GOVET) ./...; \
	fi

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	@$(GOVET) ./...

fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(NC)"
	@$(GOCMD) fmt ./...

# Dependency management
deps: ## Download dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) verify

deps-update: ## Update dependencies
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@$(GOMOD) tidy
	@$(GOGET) -u ./...

deps-vendor: ## Create vendor directory
	@echo "$(YELLOW)Creating vendor directory...$(NC)"
	@$(GOMOD) vendor

# Development targets
dev: build-dev ## Build and run for development
	@echo "$(YELLOW)Starting development server...$(NC)"
	@./$(BINARY_NAME) -http-port=:6170

run: build-dev ## Build and run with default settings
	@echo "$(YELLOW)Starting SNS Notify...$(NC)"
	@./$(BINARY_NAME) -http-port=:6170 -log-file=""

run-with-logs: build-dev ## Build and run with file logging
	@echo "$(YELLOW)Starting SNS Notify with file logging...$(NC)"
	@sudo mkdir -p $(LOG_DIR)
	@sudo chown $(USER):$(USER) $(LOG_DIR)
	@./$(BINARY_NAME) -http-port=:6170 -log-file=$(LOG_DIR)/$(BINARY_NAME).log

# Installation targets
install: build-linux ## Install the service (requires sudo)
	@echo "$(YELLOW)Installing SNS Notify...$(NC)"
	@sudo mkdir -p $(SERVICE_DIR)
	@sudo mkdir -p $(LOG_DIR)
	@sudo cp $(BINARY_NAME)-linux-amd64 $(SERVICE_DIR)/$(BINARY_NAME)
	@sudo chmod +x $(SERVICE_DIR)/$(BINARY_NAME)
	@sudo chown root:root $(SERVICE_DIR)/$(BINARY_NAME)
	@if [ -f $(SCRIPT_DIR)/sns-notify.service ]; then \
		sudo cp $(SCRIPT_DIR)/sns-notify.service /etc/systemd/system/; \
		sudo systemctl daemon-reload; \
		echo "$(GREEN)✅ Service installed. Use 'make service-start' to start$(NC)"; \
	else \
		echo "$(RED)❌ Service file not found at $(SCRIPT_DIR)/sns-notify.service$(NC)"; \
	fi

uninstall: ## Uninstall the service (requires sudo)
	@echo "$(YELLOW)Uninstalling SNS Notify...$(NC)"
	@sudo systemctl stop sns-notify.service 2>/dev/null || true
	@sudo systemctl disable sns-notify.service 2>/dev/null || true
	@sudo rm -f /etc/systemd/system/sns-notify.service
	@sudo rm -rf $(SERVICE_DIR)
	@sudo systemctl daemon-reload
	@echo "$(GREEN)✅ SNS Notify uninstalled$(NC)"

# Service management
service-start: ## Start the systemd service
	@echo "$(YELLOW)Starting SNS Notify service...$(NC)"
	@sudo systemctl start sns-notify.service
	@sudo systemctl status sns-notify.service

service-stop: ## Stop the systemd service
	@echo "$(YELLOW)Stopping SNS Notify service...$(NC)"
	@sudo systemctl stop sns-notify.service

service-restart: ## Restart the systemd service
	@echo "$(YELLOW)Restarting SNS Notify service...$(NC)"
	@sudo systemctl restart sns-notify.service
	@sudo systemctl status sns-notify.service

service-status: ## Check service status
	@sudo systemctl status sns-notify.service

service-logs: ## View service logs
	@sudo journalctl -u sns-notify.service -f

service-enable: ## Enable service to start on boot
	@sudo systemctl enable sns-notify.service
	@echo "$(GREEN)✅ Service enabled for auto-start$(NC)"

service-disable: ## Disable service auto-start
	@sudo systemctl disable sns-notify.service
	@echo "$(GREEN)✅ Service auto-start disabled$(NC)"

# Testing and validation
test-api: ## Test API endpoints
	@echo "$(YELLOW)Testing API endpoints...$(NC)"
	@chmod +x $(SCRIPT_DIR)/test_qr_login.sh
	@$(SCRIPT_DIR)/test_qr_login.sh

test-post: ## Test posting functionality
	@echo "$(YELLOW)Testing post functionality...$(NC)"
	@chmod +x $(SCRIPT_DIR)/quick_test_post.sh
	@$(SCRIPT_DIR)/quick_test_post.sh

health-check: ## Check application health
	@echo "$(YELLOW)Checking application health...$(NC)"
	@curl -s http://localhost:6170/health | jq . 2>/dev/null || curl -s http://localhost:6170/health

# Cleanup targets
clean: ## Clean build artifacts
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(BINARY_NAME)-*
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)✅ Cleanup completed$(NC)"

clean-logs: ## Clean log files (requires sudo)
	@echo "$(YELLOW)Cleaning log files...$(NC)"
	@sudo rm -f $(LOG_DIR)/*.log
	@echo "$(GREEN)✅ Log files cleaned$(NC)"

clean-all: clean clean-logs ## Clean everything

# Docker targets (for future use)
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(NC)"
	@docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "$(YELLOW)Running Docker container...$(NC)"
	@docker run -p 6170:6170 $(BINARY_NAME):latest

# Release targets
release: clean deps test lint build-linux ## Prepare release build
	@echo "$(GREEN)✅ Release build completed$(NC)"
	@ls -la $(BINARY_NAME)-linux-amd64

# Quick commands
quick-start: build-dev run ## Quick build and start

# Show project info
info: ## Show project information
	@echo "$(BLUE)SNS Notify - Project Information$(NC)"
	@echo "$(YELLOW)Project:$(NC) $(PACKAGE)"
	@echo "$(YELLOW)Module:$(NC) $(MODULE)"
	@echo "$(YELLOW)Binary:$(NC) $(BINARY_NAME)"
	@echo "$(YELLOW)Target:$(NC) $(GOOS)/$(GOARCH)"
	@echo "$(YELLOW)Build Dir:$(NC) $(BUILD_DIR)"
	@echo "$(YELLOW)Service Dir:$(NC) $(SERVICE_DIR)"
	@echo "$(YELLOW)Log Dir:$(NC) $(LOG_DIR)"
	@echo ""
	@echo "$(YELLOW)Go Version:$(NC)"
	@$(GOCMD) version
