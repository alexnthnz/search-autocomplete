# Search Autocomplete Service Makefile

.PHONY: help build run test clean docker-build docker-run deps fmt lint

# Variables
APP_NAME = search-autocomplete
BINARY_NAME = autocomplete-server
DOCKER_IMAGE = $(APP_NAME):latest
CONFIG_FILE = configs/config.env

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development commands
deps: ## Install dependencies
	go mod download
	go mod tidy

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

lint: ## Run linter
	golangci-lint run || echo "golangci-lint not installed, run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin v1.54.2"

build: deps fmt ## Build the application
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) cmd/server/main.go
	@echo "Binary built: bin/$(BINARY_NAME)"

run: build ## Run the application locally
	@echo "Starting $(APP_NAME)..."
	@if [ -f $(CONFIG_FILE) ]; then \
		echo "Loading configuration from $(CONFIG_FILE)"; \
		export $$(cat $(CONFIG_FILE) | grep -v '^#' | xargs) && ./bin/$(BINARY_NAME); \
	else \
		echo "No config file found, using defaults"; \
		./bin/$(BINARY_NAME); \
	fi

dev: ## Run in development mode with auto-reload
	@echo "Running in development mode..."
	@if [ -f $(CONFIG_FILE) ]; then \
		export $$(cat $(CONFIG_FILE) | grep -v '^#' | xargs) && LOG_LEVEL=debug go run cmd/server/main.go; \
	else \
		LOG_LEVEL=debug go run cmd/server/main.go; \
	fi

test: ## Run tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Production commands
build-prod: ## Build for production
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/$(BINARY_NAME) cmd/server/main.go

# Docker commands
docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE) .

docker-run: docker-build ## Run in Docker container
	docker run -p 8080:8080 --env-file $(CONFIG_FILE) $(DOCKER_IMAGE)

docker-compose-up: ## Start with docker-compose (includes Redis)
	docker-compose up --build

docker-compose-down: ## Stop docker-compose services
	docker-compose down

# Utility commands
clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

install: build ## Install binary to GOPATH/bin
	go install cmd/server/main.go

# Sample data and testing
load-sample-data: ## Load sample suggestions via API
	@echo "Loading sample data..."
	@curl -X POST http://localhost:8080/api/v1/admin/suggestions/batch \
		-H "Content-Type: application/json" \
		-H "X-API-Key: your-secret-api-key-here" \
		-d '[
			{"term": "machine learning", "frequency": 1500, "score": 1500, "category": "tech"},
			{"term": "artificial intelligence", "frequency": 1200, "score": 1200, "category": "tech"},
			{"term": "data science", "frequency": 1000, "score": 1000, "category": "tech"},
			{"term": "react native", "frequency": 800, "score": 800, "category": "tech"},
			{"term": "vue.js", "frequency": 600, "score": 600, "category": "tech"},
			{"term": "typescript", "frequency": 900, "score": 900, "category": "tech"},
			{"term": "kubernetes", "frequency": 700, "score": 700, "category": "tech"},
			{"term": "docker containers", "frequency": 650, "score": 650, "category": "tech"},
			{"term": "microservices", "frequency": 550, "score": 550, "category": "tech"},
			{"term": "cloud computing", "frequency": 1100, "score": 1100, "category": "tech"}
		]'

test-api: ## Test API endpoints
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/api/v1/health | jq .
	@echo "\nTesting autocomplete endpoint..."
	@curl -s "http://localhost:8080/api/v1/autocomplete?q=app&limit=5" | jq .
	@echo "\nTesting stats endpoint..."
	@curl -s http://localhost:8080/api/v1/stats | jq .

# Performance testing
stress-test: ## Run basic stress test
	@echo "Running stress test..."
	@for i in {1..100}; do \
		curl -s "http://localhost:8080/api/v1/autocomplete?q=app" > /dev/null & \
	done
	@wait
	@echo "Stress test completed"

# Setup commands
setup: deps ## Initial setup for development
	@echo "Setting up development environment..."
	@mkdir -p bin logs data
	@cp configs/config.env .env.example
	@echo "Setup complete! Copy .env.example to .env and configure as needed."

# Database/Redis helpers (when using Redis)
redis-start: ## Start Redis server locally
	redis-server --daemonize yes --port 6379

redis-stop: ## Stop Redis server
	redis-cli shutdown

redis-cli: ## Open Redis CLI
	redis-cli

# Monitoring and logs
logs: ## Show application logs (when running with systemd or similar)
	journalctl -u $(APP_NAME) -f

stats: ## Show current service statistics
	@curl -s http://localhost:8080/api/v1/stats | jq .

# Release commands
version: ## Show version info
	@echo "App: $(APP_NAME)"
	@echo "Go version: $$(go version)"
	@echo "Build time: $$(date)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Environment-specific runs
run-dev: ## Run with development settings
	LOG_LEVEL=debug ENABLE_CORS=true go run cmd/server/main.go

run-prod: ## Run with production settings  
	LOG_LEVEL=warn REDIS_ENABLED=true CACHE_TTL=15m ./bin/$(BINARY_NAME)

# Backup and restore (for persistent data scenarios)
backup: ## Backup suggestions data
	@echo "Creating backup..."
	@curl -s http://localhost:8080/api/v1/stats > backups/stats_$$(date +%Y%m%d_%H%M%S).json

# Security
security-scan: ## Run security scan on dependencies
	@echo "Scanning for security vulnerabilities..."
	@go list -json -m all | nancy sleuth || echo "nancy not installed, run: go install github.com/sonatypecommunity/nancy@latest" 