# Set default target
.DEFAULT_GOAL := help

# Version variables
# Check if we're on a tagged commit (release)
IS_TAGGED := $(shell git describe --tags --exact-match > /dev/null 2>&1 && echo "1" || echo "0")

ifeq ($(IS_TAGGED),1)
    # On a release tag - use clean version (e.g., 1.2.0)
    APP_VERSION := $(shell git describe --tags --abbrev=0 | sed 's/^v//')
else
    # Development build - include commit info (e.g., 1.2.0-67-gb3c7af9-dirty)
    APP_VERSION := $(shell git describe --tags --always | sed 's/^v//')
endif

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Auto-detect host architecture
HOST_ARCH := $(shell uname -m)
ifeq ($(HOST_ARCH),x86_64)
    PLATFORM := linux/amd64
else ifeq ($(HOST_ARCH),arm64)
    PLATFORM := linux/arm64
else
    PLATFORM := linux/amd64
endif

# Docker configuration
COMPOSE_FILE := docker-compose.yml
ENV_FILE := .env.docker
IMAGE_NAME := medicaments-api
IMAGE_TAG := $(IMAGE_NAME):$(APP_VERSION)
OBS_DIR := observability

# Colors for output
CYAN := \033[36m
GREEN := \033[32m
RESET := \033[0m

##@ Help

.PHONY: help
help: ## Display this help message
	@echo "=========================================="
	@echo "  Medicaments API - Make Commands"
	@echo "=========================================="
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make $(CYAN)<target>$(RESET)\n\nTargets:\n"} \
		/^[a-zA-Z][a-zA-Z0-9_-]*:.*##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 } \
		/^##@/ { printf "\n%s\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""

##@ Docker Operations

.PHONY: validate-secrets
validate-secrets: ## Validate required secrets files exist
	@if [ ! -f ./observability/secrets/grafana_password.txt ]; then \
		echo "❌ Error: observability/secrets/grafana_password.txt not found"; \
		echo ""; \
		echo "Required secrets are missing. Please run:"; \
		echo "  make -C observability setup"; \
		echo ""; \
		exit 1; \
	fi
	@if [ ! -r ./observability/secrets/grafana_password.txt ]; then \
		echo "❌ Error: observability/secrets/grafana_password.txt is not readable"; \
		echo "Run: chmod 644 observability/secrets/grafana_password.txt"; \
		exit 1; \
	fi
	@echo "✓ Secrets validated successfully"

.PHONY: build
build: ## Build Docker image (auto-detects host arch)
	@echo "Building $(IMAGE_TAG) ($(GIT_COMMIT)) for $(PLATFORM)..."
	@GIT_COMMIT=$(GIT_COMMIT) \
		BUILD_DATE=$(BUILD_DATE) \
		APP_VERSION=$(APP_VERSION) \
		docker compose --env-file $(ENV_FILE) build
	@echo "$(GREEN)✓ Build complete: $(IMAGE_TAG)$(RESET)"

.PHONY: build-amd64
build-amd64: ## Force amd64 build
	@echo "Building $(IMAGE_TAG) ($(GIT_COMMIT)) for linux/amd64..."
	@GIT_COMMIT=$(GIT_COMMIT) \
		BUILD_DATE=$(BUILD_DATE) \
		APP_VERSION=$(APP_VERSION) \
		docker buildx build --platform linux/amd64 -t $(IMAGE_NAME):amd64 --load .
	@echo "$(GREEN)✓ Build complete: $(IMAGE_NAME):amd64$(RESET)"

.PHONY: build-arm64
build-arm64: ## Force arm64 build
	@echo "Building $(IMAGE_TAG) ($(GIT_COMMIT)) for linux/arm64..."
	@GIT_COMMIT=$(GIT_COMMIT) \
		BUILD_DATE=$(BUILD_DATE) \
		APP_VERSION=$(APP_VERSION) \
		docker buildx build --platform linux/arm64 -t $(IMAGE_NAME):arm64 --load .
	@echo "$(GREEN)✓ Build complete: $(IMAGE_NAME):arm64$(RESET)"

.PHONY: up
up: obs-up ## Start all services (obs stack + app)
	@echo "Starting services..."
	@APP_VERSION=$(APP_VERSION) \
		docker compose --env-file $(ENV_FILE) up -d
	@echo "$(GREEN)✓ Services started!$(RESET)"
	@echo ""
	@echo "API:        http://localhost:8030"

.PHONY: down
down: ## Stop all services (app + obs stack)
	@echo "Stopping services..."
	@docker compose --env-file $(ENV_FILE) down
	@$(MAKE) obs-down
	@echo "$(GREEN)✓ Services stopped$(RESET)"

.PHONY: restart
restart: down up ## Restart all services (app + obs stack)

.PHONY: rebuild
rebuild: ## Rebuild Docker images (without cleanup)
	@echo "Rebuilding images..."
	@docker compose --env-file $(ENV_FILE) build --no-cache
	@echo "$(GREEN)✓ Rebuild complete$(RESET)"

##@ Monitoring

.PHONY: logs
logs: ## View logs (use SERVICE=name to filter)
	@docker compose --env-file $(ENV_FILE) logs -f $(SERVICE)

.PHONY: ps
ps: ## Show service status
	@docker compose --env-file $(ENV_FILE) ps

.PHONY: stats
stats: ## Show resource usage
	@docker stats --no-stream

##@ Observability Submodule

.PHONY: obs-up obs-down obs-update obs-logs obs-status obs-init

obs-init: ## Initialize observability submodule
	@git submodule update --init --recursive $(OBS_DIR)
	@$(MAKE) -C $(OBS_DIR) setup
	@echo "$(GREEN)✓ Observability submodule initialized$(RESET)"

obs-up: ## Start observability stack (via submodule)
	@echo "Starting observability stack..."
	@$(MAKE) -C $(OBS_DIR) up

obs-down: ## Stop observability stack (via submodule)
	@$(MAKE) -C $(OBS_DIR) down

obs-update: ## Update observability submodule
	@echo "Updating observability submodule..."
	@git submodule update --remote $(OBS_DIR)
	@echo "$(GREEN)✓ Observability submodule updated$(RESET)"

obs-logs: ## View observability stack logs
	@$(MAKE) -C $(OBS_DIR) logs

obs-status: ## Show observability stack status
	@$(MAKE) -C $(OBS_DIR) status

##@ Maintenance

.PHONY: clean
clean: ## Remove containers, networks, volumes, and images
	@echo "Removing all Docker resources..."
	@docker compose --env-file $(ENV_FILE) down --volumes --rmi all
	@echo "$(GREEN)✓ All Docker resources removed$(RESET)"

.PHONY: export
export: ## Export Docker image as tar file (optional: IMAGE=tag)
	@IMAGE=$${IMAGE:-$(IMAGE_TAG)}; \
	FILENAME=$$(echo $$IMAGE | tr ':/' '-').tar; \
	echo "Exporting $$IMAGE to $$FILENAME..."; \
	docker save $$IMAGE -o $$FILENAME; \
	echo "$(GREEN)✓ Export complete: $$FILENAME$(RESET)"

.PHONY: import
import: ## Import Docker image from tar file (optional: FILE=tarfile)
	@if [ -z "$(FILE)" ]; then \
		echo "Available Docker image tar files:"; \
		echo ""; \
		ls -1t *.tar 2>/dev/null | head -10 | nl -v 0; \
		echo ""; \
		read -p "Enter file number (or 0 to cancel): " num; \
		if [ "$$num" = "0" ]; then \
			echo "Import cancelled"; \
			exit 1; \
		fi; \
		FILE=$$(ls -1t *.tar 2>/dev/null | sed -n "$$((num+1))p"); \
		if [ -z "$$FILE" ]; then \
			echo "Error: Invalid selection"; \
			exit 1; \
		fi; \
	fi
	@echo "Importing $(FILE)..."; \
	docker load -i $(FILE); \
	echo "$(GREEN)✓ Import complete: $(FILE)$(RESET)"

##@ Testing

.PHONY: test
test: ## Run all tests
	@echo "Running all tests..."
	@go test -v ./...

.PHONY: test-short
test-short: ## Run unit tests only
	@echo "Running unit tests..."
	@go test -short -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	@go test -race -v ./...

.PHONY: test-smoke
test-smoke: ## Run smoke tests
	@echo "Running smoke tests..."
	@go test ./tests -run .*Smoke.* -v

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test ./tests -run TestIntegration -v

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out -v ./...
	@echo "$(GREEN)✓ Coverage report generated: coverage.out$(RESET)"

##@ Benchmarking

.PHONY: bench
bench: ## Run all benchmarks (handlers + tests)
	@echo "Running all benchmarks..."
	@echo "=== Handler benchmarks ==="
	@go test ./handlers -bench=. -benchmem -v
	@echo ""
	@echo "=== Integration benchmarks ==="
	@go test ./tests -bench=. -benchmem -run=^$ -v

.PHONY: bench-handlers
bench-handlers: ## Run handler benchmarks only
	@echo "Running handler benchmarks..."
	@go test ./handlers -bench=. -benchmem -v

.PHONY: bench-tests
bench-tests: ## Run integration benchmarks only
	@echo "Running integration benchmarks..."
	@go test ./tests -bench=. -benchmem -run=^$ -v

.PHONY: bench-algorithmic
bench-algorithmic: ## Run algorithmic performance benchmarks
	@echo "Running algorithmic benchmarks..."
	@go test ./tests -bench=BenchmarkAlgorithmicPerformance -benchmem -v

.PHONY: bench-http
bench-http: ## Run HTTP performance benchmarks
	@echo "Running HTTP benchmarks..."
	@go test ./tests -bench=BenchmarkHTTPPerformance -benchmem -v

.PHONY: bench-realworld
bench-realworld: ## Run real-world search benchmarks
	@echo "Running real-world search benchmarks..."
	@go test ./tests -bench=BenchmarkRealWorldSearch -benchmem -v

.PHONY: bench-sustained
bench-sustained: ## Run sustained performance benchmarks
	@echo "Running sustained performance benchmarks..."
	@go test ./tests -bench=BenchmarkSustainedPerformance -benchmem -v

.PHONY: bench-profile
bench-profile: ## Run benchmarks with CPU profiling
	@echo "Running benchmarks with CPU profiling..."
	@go test ./handlers -bench=. -benchmem -cpuprofile=cpu.prof -v
	@echo "$(GREEN)✓ CPU profile generated: cpu.prof$(RESET)"

##@ Code Quality

.PHONY: lint
lint: ## Run code linting
	@echo "Running linters..."
	@go vet ./...
	@gofmt -l . | [ $$(gofmt -l . | wc -l) -eq 0 ] || (echo "Files need formatting:" && gofmt -d . && exit 1)
	@echo "$(GREEN)✓ Linting complete$(RESET)"

##@ Information

.PHONY: version
version: ## Show version information
	@echo "App Version: $(APP_VERSION)"
	@echo "Git Commit:  $(GIT_COMMIT)"
	@echo "Build Date:  $(BUILD_DATE)"
	@echo "Platform:     $(PLATFORM)"
	@echo "Image Tag:    $(IMAGE_TAG)"
