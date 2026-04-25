.PHONY: build test docker-build up down verify clean help

# Variables
BINARY_NAME=oidc-server
DOCKER_IMAGE=democryst/go-oidc:latest

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

build: ## Build the OIDC provider binary
	@echo "Building binary..."
	go build -o $(BINARY_NAME) ./cmd/server/main.go

test: ## Run all unit tests
	@echo "Running tests..."
	go test -v ./...

docker-build: ## Build the hardened Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

up: ## Start the full stack locally via Docker Compose
	@echo "Starting local environment..."
	docker-compose up -d

down: ## Stop local environment
	@echo "Stopping local environment..."
	docker-compose down

verify: build test ## Verify build and run tests
	@echo "Verification complete."

clean: ## Clean build artifacts
	@echo "Cleaning artifacts..."
	rm -f $(BINARY_NAME)
	rm -f *.test
