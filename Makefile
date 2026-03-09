# Makefile for rackd

BINARY := rackd
BUILD_DIR := ./build
WEBUI_DIR := ./webui
ASSETS_DIR := ./internal/ui/assets
GO := GOTOOLCHAIN=go1.26.0 go

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags="-s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(GIT_COMMIT) \
	-X main.date=$(BUILD_TIME)"

.PHONY: all build binary ui-build test clean run-server dev lint fmt help validate

## Default target
all: build

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build complete application (UI + binary)
build: ui-build binary

## binary: Build Go binary only
binary:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

## ui-build: Build web UI assets
ui-build:
	@echo "Building UI..."
	@mkdir -p $(ASSETS_DIR)
	cd $(WEBUI_DIR) && bun install
	cd $(WEBUI_DIR) && bun run build
	cp $(WEBUI_DIR)/dist/index.html $(ASSETS_DIR)/
	cp $(WEBUI_DIR)/dist/output.css $(ASSETS_DIR)/
	cp $(WEBUI_DIR)/dist/app.js $(ASSETS_DIR)/

## test: Run all tests
test: ui-build
	$(GO) test -v -coverprofile=coverage.out ./...

## test-short: Run short tests only
test-short:
	$(GO) test -v -short ./...

## test-race: Run tests with race detector
test-race: ui-build
	$(GO) test -v -race ./...

## test-coverage: Show test coverage
test-coverage: test
	$(GO) tool cover -func=coverage.out

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(ASSETS_DIR)
	rm -f coverage.out

## run-server: Build and run server
run-server: build
	$(BUILD_DIR)/$(BINARY) server

run-dev: build
	$(BUILD_DIR)/$(BINARY) server --dev-mode --log-level debug

## dev: Run in development mode with hot reload
dev:
	@echo "Starting development server..."
	@$(MAKE) ui-build
	$(GO) run . server --log-level debug

## lint: Run linters
lint:
	golangci-lint run ./...

## security: Run security scanner (gosec)
security:
	gosec ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...
	gofumpt -w .

## validate: Run all validations (build, test, vet, lint)
validate:
	@echo "=== Building ==="
	go build ./...
	@echo "=== Running tests ==="
	go test ./... -v
	@echo "=== Running vet ==="
	go vet ./...
	@echo "=== Running lint ==="
	golangci-lint run || true
	@echo "=== Validation complete ==="

## build-linux: Build for Linux
build-linux: ui-build
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .

## build-darwin: Build for macOS
build-darwin: ui-build
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 .

## build-windows: Build for Windows
build-windows: ui-build
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe .

## docker: Build Docker image
docker:
	docker build -t $(BINARY):$(VERSION) .

## docker-run: Run Docker container
docker-run:
	docker run -p 8080:8080 -v $(PWD)/data:/data $(BINARY):$(VERSION)

## push-tag: Create and push a git tag (usage: make push-tag TAG=v1.0.0)
push-tag:
ifndef TAG
	$(error TAG is undefined. Usage: make push-tag TAG=v1.0.0)
endif
	git tag $(TAG)
	git push origin $(TAG)