# Makefile for innominatus
# Score-based Platform Orchestration

.DEFAULT_GOAL := help

# Colors for output
GREEN  := \033[0;32m
YELLOW := \033[0;33m
BLUE   := \033[0;34m
RED    := \033[0;31m
NC     := \033[0m # No Color

# Variables
GO_CMD := go
NPM_CMD := npm
SERVER_BINARY := innominatus
CLI_BINARY := innominatus-ctl
WEB_UI_DIR := web-ui
COVERAGE_FILE := coverage.out

# Go environment
export GO111MODULE=on
export CGO_ENABLED=0

# Database configuration - PostgreSQL by default
export DB_HOST ?= localhost
export DB_PORT ?= 5432
export DB_USER ?= postgres
export DB_PASSWORD ?= postgres
export DB_NAME ?= idp_orchestrator2
export DB_SSLMODE ?= disable

##@ Help

.PHONY: help
help: ## Display this help message
	@echo ""
	@echo "$(BLUE)innominatus - Score-based Platform Orchestration$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(YELLOW)<target>$(NC)\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n$(BLUE)%s$(NC)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""

##@ Development

.PHONY: install
install: ## Install all dependencies (Go + npm)
	@echo "$(GREEN)Installing Go dependencies...$(NC)"
	@$(GO_CMD) mod download
	@echo "$(GREEN)Installing web-ui dependencies...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) install
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

.PHONY: build
build: build-server build-cli build-mcp build-ui ## Build all components (server, CLI, MCP, web UI)

.PHONY: prepare-embed
prepare-embed: ## Prepare static files for Go embed (internal target)
	@./scripts/prepare-embed.sh

.PHONY: build-server
build-server: prepare-embed ## Build the server binary
	@echo "$(GREEN)Building server...$(NC)"
	@$(GO_CMD) build -o $(SERVER_BINARY) cmd/server/main.go
	@echo "$(GREEN)✓ Server built: ./$(SERVER_BINARY)$(NC)"

.PHONY: build-cli
build-cli: ## Build the CLI binary
	@echo "$(GREEN)Building CLI...$(NC)"
	@$(GO_CMD) build -o $(CLI_BINARY) ./cmd/cli
	@echo "$(GREEN)✓ CLI built: ./$(CLI_BINARY)$(NC)"

.PHONY: build-mcp
build-mcp: ## Build the MCP server binary (for Claude Desktop integration)
	@echo "$(GREEN)Building MCP server...$(NC)"
	@$(GO_CMD) build -o innominatus-mcp ./cmd/mcp-server
	@echo "$(GREEN)✓ MCP server built: ./innominatus-mcp$(NC)"
	@echo "$(YELLOW)See docs/MCP_SERVER_GO.md for installation instructions$(NC)"

.PHONY: build-ui
build-ui: ## Build the web UI
	@echo "$(GREEN)Building web UI...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run build
	@echo "$(GREEN)✓ Web UI built$(NC)"

.PHONY: run
run: build-ui prepare-embed ## Run the server (build web-ui first)
	@echo "$(GREEN)Starting server...$(NC)"
	@$(GO_CMD) run cmd/server/main.go

.PHONY: run-server
run-server: ## Run the server only (assumes files already prepared)
	@echo "$(GREEN)Starting server...$(NC)"
	@$(GO_CMD) run cmd/server/main.go

.PHONY: run-cli
run-cli: ## Run the CLI
	@echo "$(GREEN)Running CLI...$(NC)"
	@$(GO_CMD) run cmd/cli/main.go

.PHONY: dev
dev: ## Start server + web UI in development mode (parallel)
	@echo "$(GREEN)Starting development environment...$(NC)"
	@echo "$(YELLOW)Server: http://localhost:8081$(NC)"
	@echo "$(YELLOW)Web UI: http://localhost:3000$(NC)"
	@$(MAKE) -j2 dev-server dev-ui

.PHONY: dev-server
dev-server: build-ui prepare-embed ## Start server in development mode (with embedded files)
	@echo "$(GREEN)Starting server with embedded web-ui...$(NC)"
	@$(GO_CMD) run cmd/server/main.go

.PHONY: dev-ui
dev-ui: ## Start web UI in development mode
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run dev

##@ Testing

.PHONY: test
test: ## Run all local tests (unit + e2e, no Kubernetes)
	@echo "$(GREEN)Running all local tests...$(NC)"
	@$(MAKE) test-unit
	@$(MAKE) test-e2e
	@$(MAKE) test-ui

.PHONY: test-unit
test-unit: prepare-embed ## Run Go unit tests with race detection
	@echo "$(GREEN)Running Go unit tests...$(NC)"
	@$(GO_CMD) test ./... -v -race -coverprofile=$(COVERAGE_FILE) -short

.PHONY: test-e2e
test-e2e: ## Run Go E2E tests (skip Kubernetes tests)
	@echo "$(GREEN)Running Go E2E tests (no Kubernetes)...$(NC)"
	@export SKIP_DEMO_TESTS=1 && \
	export SKIP_INTEGRATION_TESTS=1 && \
	$(GO_CMD) test ./tests/e2e -v -run "TestValidate|TestAnalyze|TestGoldenPaths|TestDeployment"

.PHONY: test-e2e-k8s
test-e2e-k8s: ## Run full E2E tests including Kubernetes demo tests
	@echo "$(GREEN)Running full E2E tests (including Kubernetes)...$(NC)"
	@echo "$(YELLOW)Note: Requires Docker Desktop with Kubernetes enabled$(NC)"
	@$(GO_CMD) test ./tests/e2e -v -timeout 30m

.PHONY: test-e2e-gitops
test-e2e-gitops: ## Run GitOps E2E integration tests (requires Gitea + ArgoCD)
	@echo "$(GREEN)Running GitOps E2E integration tests...$(NC)"
	@echo "$(YELLOW)Prerequisites: Gitea, ArgoCD, GITEA_TOKEN environment variable$(NC)"
	@./scripts/run-e2e-tests.sh

.PHONY: test-ui
test-ui: ## Run Web UI Playwright tests
	@echo "$(GREEN)Running Web UI E2E tests...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run test:e2e

.PHONY: test-ui-ui
test-ui-ui: ## Run Web UI tests in UI mode
	@echo "$(GREEN)Running Web UI tests in UI mode...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run test:e2e:ui

.PHONY: test-ui-debug
test-ui-debug: ## Run Web UI tests in debug mode
	@echo "$(GREEN)Running Web UI tests in debug mode...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run test:e2e:debug

.PHONY: test-all
test-all: ## Run complete test suite (including K8s tests)
	@echo "$(GREEN)Running complete test suite...$(NC)"
	@$(MAKE) test-unit
	@$(MAKE) test-e2e-k8s
	@$(MAKE) test-ui

.PHONY: test-ci
test-ci: ## Simulate CI test run (matches GitHub Actions)
	@echo "$(GREEN)Simulating CI test run...$(NC)"
	@echo "$(YELLOW)1/3 Go unit tests...$(NC)"
	@$(GO_CMD) test ./... -v -race -coverprofile=$(COVERAGE_FILE) -short
	@echo "$(YELLOW)2/3 Go E2E tests...$(NC)"
	@export SKIP_DEMO_TESTS=1 && \
	export SKIP_INTEGRATION_TESTS=1 && \
	$(GO_CMD) test ./tests/e2e -v
	@echo "$(YELLOW)3/3 Web UI tests...$(NC)"
	@cd $(WEB_UI_DIR) && CI=true $(NPM_CMD) run test:e2e

##@ Code Quality

.PHONY: lint
lint: lint-go lint-ui ## Run all linters

.PHONY: lint-go
lint-go: ## Lint Go code
	@echo "$(GREEN)Linting Go code...$(NC)"
	@$(GO_CMD) vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint not installed, using go vet only$(NC)"; \
	fi

.PHONY: lint-ui
lint-ui: ## Lint TypeScript/React code
	@echo "$(GREEN)Linting web UI...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run lint

.PHONY: fmt
fmt: fmt-go fmt-ui ## Format all code

.PHONY: fmt-go
fmt-go: ## Format Go code
	@echo "$(GREEN)Formatting Go code...$(NC)"
	@$(GO_CMD) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

.PHONY: fmt-ui
fmt-ui: ## Format TypeScript/React code
	@echo "$(GREEN)Formatting web UI code...$(NC)"
	@cd $(WEB_UI_DIR) && $(NPM_CMD) run format

.PHONY: vet
vet: ## Run go vet
	@echo "$(GREEN)Running go vet...$(NC)"
	@$(GO_CMD) vet ./...

.PHONY: coverage
coverage: ## Generate and view coverage report
	@echo "$(GREEN)Generating coverage report...$(NC)"
	@$(GO_CMD) test ./... -coverprofile=$(COVERAGE_FILE) -covermode=atomic
	@$(GO_CMD) tool cover -func=$(COVERAGE_FILE)
	@echo ""
	@echo "$(GREEN)Opening HTML coverage report...$(NC)"
	@$(GO_CMD) tool cover -html=$(COVERAGE_FILE)

.PHONY: coverage-summary
coverage-summary: ## Show coverage summary
	@echo "$(GREEN)Coverage Summary:$(NC)"
	@$(GO_CMD) test ./... -coverprofile=$(COVERAGE_FILE) -covermode=atomic >/dev/null 2>&1
	@$(GO_CMD) tool cover -func=$(COVERAGE_FILE) | tail -20

##@ Utilities

.PHONY: setup-playwright
setup-playwright: ## Install Playwright browsers
	@echo "$(GREEN)Installing Playwright browsers...$(NC)"
	@cd $(WEB_UI_DIR) && npx playwright install --with-deps

.PHONY: clean
clean: ## Remove build artifacts and caches
	@echo "$(GREEN)Cleaning build artifacts...$(NC)"
	@rm -f $(SERVER_BINARY) $(CLI_BINARY)
	@rm -f $(COVERAGE_FILE)
	@rm -rf $(WEB_UI_DIR)/out
	@rm -rf $(WEB_UI_DIR)/.next
	@rm -rf $(WEB_UI_DIR)/playwright-report
	@rm -rf $(WEB_UI_DIR)/test-results
	@echo "$(GREEN)✓ Clean complete$(NC)"

.PHONY: clean-all
clean-all: clean ## Remove all generated files including dependencies
	@echo "$(GREEN)Removing dependencies...$(NC)"
	@rm -rf $(WEB_UI_DIR)/node_modules
	@$(GO_CMD) clean -modcache
	@echo "$(GREEN)✓ Deep clean complete$(NC)"

.PHONY: demo-time
demo-time: ## Install demo environment (requires K8s)
	@echo "$(GREEN)Installing demo environment...$(NC)"
	@./$(CLI_BINARY) demo-time

.PHONY: demo-status
demo-status: ## Check demo environment status
	@./$(CLI_BINARY) demo-status

.PHONY: demo-nuke
demo-nuke: ## Remove demo environment
	@echo "$(YELLOW)Removing demo environment...$(NC)"
	@./$(CLI_BINARY) demo-nuke

##@ Git Hooks

.PHONY: install-hooks
install-hooks: ## Install git pre-commit hooks
	@echo "$(GREEN)Installing git hooks...$(NC)"
	@echo '#!/bin/sh\nmake test-unit\nmake lint' > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "$(GREEN)✓ Git hooks installed$(NC)"

##@ Info

.PHONY: version
version: ## Show version information
	@echo "$(BLUE)innominatus Version Info:$(NC)"
	@echo "  Go version: $$($(GO_CMD) version)"
	@echo "  Node version: $$(node --version 2>/dev/null || echo 'not installed')"
	@echo "  npm version: $$($(NPM_CMD) --version 2>/dev/null || echo 'not installed')"
	@echo "  Playwright: $$(cd $(WEB_UI_DIR) && npx playwright --version 2>/dev/null || echo 'not installed')"

.PHONY: db-status
db-status: ## Check PostgreSQL database connection
	@echo "$(BLUE)Database Configuration:$(NC)"
	@echo "  Host: $(DB_HOST)"
	@echo "  Port: $(DB_PORT)"
	@echo "  Database: $(DB_NAME)"
	@echo "  User: $(DB_USER)"
	@echo ""
	@echo "$(GREEN)Testing database connection...$(NC)"
	@psql -h $(DB_HOST) -p $(DB_PORT) -U $(DB_USER) -d $(DB_NAME) -c "SELECT version();" 2>&1 | head -1 || \
		echo "$(RED)✗ Cannot connect to PostgreSQL. Ensure PostgreSQL is running.$(NC)"

.PHONY: deps-check
deps-check: ## Check if all dependencies are installed
	@echo "$(GREEN)Checking dependencies...$(NC)"
	@command -v go >/dev/null 2>&1 && echo "$(GREEN)✓ Go installed$(NC)" || echo "$(RED)✗ Go not found$(NC)"
	@command -v node >/dev/null 2>&1 && echo "$(GREEN)✓ Node.js installed$(NC)" || echo "$(RED)✗ Node.js not found$(NC)"
	@command -v npm >/dev/null 2>&1 && echo "$(GREEN)✓ npm installed$(NC)" || echo "$(RED)✗ npm not found$(NC)"
	@command -v kubectl >/dev/null 2>&1 && echo "$(GREEN)✓ kubectl installed$(NC)" || echo "$(YELLOW)⚠ kubectl not found (needed for demo)$(NC)"
	@command -v helm >/dev/null 2>&1 && echo "$(GREEN)✓ helm installed$(NC)" || echo "$(YELLOW)⚠ helm not found (needed for demo)$(NC)"
	@command -v docker >/dev/null 2>&1 && echo "$(GREEN)✓ docker installed$(NC)" || echo "$(YELLOW)⚠ docker not found (needed for demo)$(NC)"

# Mark all targets as phony
.PHONY: all $(MAKECMDGOALS)
