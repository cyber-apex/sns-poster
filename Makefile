# SNS Poster Makefile

BINARY_NAME = sns-poster
BUILD_DIR = ./cmd
BIN_DIR = ./bin
SERVICE_DIR = /opt/sns-poster

.PHONY: help build dev clean install start stop logs

help: ## Show help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build production binary
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(BUILD_DIR)

dev: ## Run with hot-reload (install air if needed)
	@command -v air > /dev/null || go install github.com/air-verse/air@latest
	@air

clean: ## Clean build artifacts
	@rm -rf $(BIN_DIR) tmp debug *.png build-errors.log

install: build ## Install service
	@sudo mkdir -p $(SERVICE_DIR) /var/logs/sns-poster
	@sudo cp $(BIN_DIR)/$(BINARY_NAME)-linux-amd64 $(SERVICE_DIR)/$(BINARY_NAME)
	@sudo chmod +x $(SERVICE_DIR)/$(BINARY_NAME)
	@sudo cp scripts/sns-poster.service /etc/systemd/system/
	@sudo systemctl daemon-reload

start: ## Start service
	@sudo systemctl start sns-poster.service

stop: ## Stop service
	@sudo systemctl stop sns-poster.service

logs: ## View logs
	@sudo journalctl -u sns-poster.service -f