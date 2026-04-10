.PHONY: build dev test clean docker docker-up docker-down lint vet web-build go-build

# Variables
BINARY=agent-forge
GO_CMD=go
NPM_CMD=npm
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default target
all: build

# ==================== Build ====================

## build: Compile Go binary + build React frontend
build: go-build web-build
	@echo "✅ Build complete"

## go-build: Compile Go binary
go-build:
	@echo "🔨 Building Go binary..."
	$(GO_CMD) build -ldflags="-s -w -X main.version=$(VERSION)" -o ./bin/$(BINARY) ./cmd/agent-forge/

## web-build: Build React frontend
web-build:
	@echo "🎨 Building React frontend..."
	cd web && $(NPM_CMD) run build

## install: Install binary to $GOPATH/bin
install:
	$(GO_CMD) install -ldflags="-s -w -X main.version=$(VERSION)" ./cmd/agent-forge/

# ==================== Development ====================

## dev: Start development mode (backend + frontend)
dev:
	@echo "🚀 Starting development mode..."
	@echo "   Backend:  go run ./cmd/agent-forge/ serve"
	@echo "   Frontend: cd web && npm run dev"
	@echo ""
	@echo "Run in separate terminals:"
	@echo "  Terminal 1: make dev-backend"
	@echo "  Terminal 2: make dev-frontend"

## dev-backend: Start Go backend in dev mode
dev-backend:
	$(GO_CMD) run ./cmd/agent-forge/ serve --port 8080

## dev-frontend: Start Vite dev server with proxy
dev-frontend:
	cd web && $(NPM_CMD) run dev

# ==================== Testing ====================

## test: Run all tests (Go + TypeScript)
test: test-go test-web
	@echo "✅ All tests passed"

## test-go: Run Go tests
test-go:
	@echo "🧪 Running Go tests..."
	$(GO_CMD) test ./... -count=1

## test-web: Run TypeScript type check
test-web:
	@echo "🧪 Checking TypeScript..."
	cd web && npx tsc --noEmit

## test-verbose: Run Go tests with verbose output
test-verbose:
	$(GO_CMD) test ./... -v -count=1

## coverage: Run Go tests with coverage
coverage:
	$(GO_CMD) test ./... -coverprofile=coverage.out
	$(GO_CMD) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# ==================== Code Quality ====================

## vet: Run go vet
vet:
	@echo "🔍 Running go vet..."
	$(GO_CMD) vet ./...

## lint: Run linters
lint: vet
	@echo "🔍 Running linters..."
	@if command -v golangci-lint > /dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "   golangci-lint not installed, skipping"; \
	fi

# ==================== Docker ====================

## docker: Build Docker image
docker:
	@echo "🐳 Building Docker image..."
	docker build -t agentforge:$(VERSION) -t agentforge:latest .

## docker-up: Start with docker-compose
docker-up:
	docker-compose up -d
	@echo "🐳 AgentForge running at http://localhost:$${AF_PORT:-8080}"

## docker-down: Stop docker-compose
docker-down:
	docker-compose down

## docker-logs: View Docker logs
docker-logs:
	docker-compose logs -f

# ==================== Cleanup ====================

## clean: Remove build artifacts
clean:
	@echo "🗑️  Cleaning..."
	rm -rf bin/
	rm -rf web/dist/
	rm -f coverage.out coverage.html
	$(GO_CMD) clean -cache

# ==================== Help ====================

## help: Show this help
help:
	@echo "AgentForge - Makefile targets:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | sort
