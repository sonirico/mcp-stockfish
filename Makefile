.PHONY: clean build install uninstall run fmt test lint help dev-tools docker-build docker-run docker-shell release version

BINARY_NAME=mcp-stockfish
BUILD_DIR=bin
IMAGE_NAME=mcp-stockfish
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT_HASH?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

help:
	@echo "mcp-stockfish development commands:"
	@echo "Version: $(VERSION)"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Time: $(BUILD_TIME)"

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build \
		-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME) -s -w -extldflags '-static'" \
		-a -installsuffix cgo \
		-o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	go clean
	docker rmi $(IMAGE_NAME):$(VERSION) 2>/dev/null || true
	docker rmi $(IMAGE_NAME):latest 2>/dev/null || true
	@echo "Cleaned build artifacts and Docker images"

deps: ## Install dependencies
	go mod tidy
	@echo "Dependencies installed"

install: build ## Install binary to /usr/local/bin
	install -c $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed $(BINARY_NAME) to /usr/local/bin/"

uninstall: ## Uninstall binary from /usr/local/bin
	rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstalled $(BINARY_NAME) from /usr/local/bin/"

run: ## Run the server (stdio mode)
	CGO_ENABLED=0 go run -ldflags "-X main.version=$(VERSION) -s -w" .

run-http: ## Run the server in HTTP mode
	CGO_ENABLED=0 MCP_STOCKFISH_SERVER_MODE=http go run -ldflags "-X main.version=$(VERSION) -s -w" .

run-http-debug: ## Run the server in HTTP mode with debug logging
	CGO_ENABLED=0 MCP_STOCKFISH_SERVER_MODE=http MCP_STOCKFISH_LOG_LEVEL=debug go run -ldflags "-X main.version=$(VERSION) -s -w" .

fmt: ## Format code
	gofmt -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not found, install with: make dev-tools"
	@command -v golines >/dev/null 2>&1 && golines -w . || echo "golines not found, install with: make dev-tools"
	@echo "Code formatted"

test: ## Run tests
	go test -v ./...

lint: ## Run linter
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not found, install from https://golangci-lint.run/"

dev-tools: ## Install development tools
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/segmentio/golines@latest
	@echo "Development tools installed"

docker-build: ## Build Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT_HASH=$(COMMIT_HASH) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest .
	@echo "Docker image built: $(IMAGE_NAME):$(VERSION)"

docker-run: ## Run Docker container
	docker run -it --rm \
		-v /usr/local/bin/stockfish:/usr/local/bin/stockfish:ro \
		$(IMAGE_NAME):$(VERSION)

docker-run-http: ## Run Docker container in HTTP mode
	docker run -it --rm \
		-p 8080:8080 \
		-e MCP_STOCKFISH_SERVER_MODE=http \
		-e MCP_STOCKFISH_HTTP_HOST=0.0.0.0 \
		$(IMAGE_NAME):$(VERSION)

docker-shell: ## Open shell in Docker container
	docker run -it --rm \
		-v /usr/local/bin/stockfish:/usr/local/bin/stockfish:ro \
		--entrypoint /bin/bash \
		$(IMAGE_NAME):$(VERSION)

release: clean build docker-build ## Build release
	@echo "Release build complete:"
	@echo "  Binary: $(BUILD_DIR)/$(BINARY_NAME) ($(VERSION))"
	@echo "  Docker: $(IMAGE_NAME):$(VERSION)"
