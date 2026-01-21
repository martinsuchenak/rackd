# Build System & Deployment

This document covers the build system, Docker deployment, and release automation.

## Makefile

```makefile
# Makefile for rackd

BINARY := rackd
BUILD_DIR := ./build
WEBUI_DIR := ./webui
ASSETS_DIR := ./internal/ui/assets
GO := go

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags="-s -w \
    -X main.version=$(VERSION) \
    -X main.commit=$(GIT_COMMIT) \
    -X main.date=$(BUILD_TIME)"

.PHONY: all build binary ui-build test clean run-server dev lint fmt help

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
	cp $(WEBUI_DIR)/src/index.html $(ASSETS_DIR)/
	cp $(WEBUI_DIR)/dist/output.css $(ASSETS_DIR)/
	cp $(WEBUI_DIR)/dist/app.js $(ASSETS_DIR)/

## test: Run all tests
test: ui-build
	$(GO) test -v -race -coverprofile=coverage.out ./...

## test-short: Run short tests only
test-short:
	$(GO) test -v -short ./...

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

## dev: Run in development mode with hot reload
dev:
	@echo "Starting development server..."
	@$(MAKE) ui-build
	$(GO) run . server --log-level debug

## lint: Run linters
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GO) fmt ./...
	gofumpt -w .

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
```

## Dockerfile

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make nodejs npm
RUN npm install -g bun

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN make build

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary
COPY --from=builder /app/build/rackd /usr/local/bin/rackd

# Create data directory
RUN mkdir -p /data

# Set environment
ENV DATA_DIR=/data
ENV LISTEN_ADDR=:8080

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/api/datacenters || exit 1

ENTRYPOINT ["rackd"]
CMD ["server"]
```

## Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  rackd:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - rackd-data:/data
    environment:
      - DATA_DIR=/data
      - LISTEN_ADDR=:8080
      - LOG_LEVEL=info
      - LOG_FORMAT=json
      - API_AUTH_TOKEN=${API_AUTH_TOKEN:-}
      - MCP_AUTH_TOKEN=${MCP_AUTH_TOKEN:-}
      - DISCOVERY_ENABLED=true
      - DISCOVERY_INTERVAL=24h
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/datacenters"]
      interval: 30s
      timeout: 3s
      retries: 3

volumes:
  rackd-data:
```

## GoReleaser

```yaml
# .goreleaser.yml
version: 2

before:
  hooks:
    - go mod tidy
    - make ui-build

builds:
  - id: rackd
    main: .
    binary: rackd
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}

archives:
  - id: default
    format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"

brews:
  - name: rackd
    repository:
      owner: martinsuchenak
      name: homebrew-tap
      token: "{{ .Env.TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/martinsuchenak/rackd"
    description: "Device inventory and IPAM management"
    license: "MIT"
    install: |
      bin.install "rackd"
    test: |
      system "#{bin}/rackd", "version"
```

## Nomad Job

```hcl
# deploy/nomad.hcl
job "rackd" {
  datacenters = ["dc1"]
  type        = "service"

  group "rackd" {
    count = 1

    network {
      port "http" {
        static = 8080
      }
    }

    volume "data" {
      type      = "host"
      source    = "rackd-data"
      read_only = false
    }

    task "rackd" {
      driver = "docker"

      config {
        image = "ghcr.io/martinsuchenak/rackd:latest"
        ports = ["http"]
      }

      volume_mount {
        volume      = "data"
        destination = "/data"
      }

      env {
        DATA_DIR          = "/data"
        LISTEN_ADDR       = ":8080"
        LOG_FORMAT        = "json"
        LOG_LEVEL         = "info"
        DISCOVERY_ENABLED = "true"
      }

      template {
        data = <<-EOF
          {{ with nomadVar "nomad/jobs/rackd" }}
          API_AUTH_TOKEN={{ .api_auth_token }}
          MCP_AUTH_TOKEN={{ .mcp_auth_token }}
          {{ end }}
        EOF
        destination = "secrets/env"
        env         = true
      }

      resources {
        cpu    = 256
        memory = 256
      }

      service {
        name = "rackd"
        port = "http"

        check {
          type     = "http"
          path     = "/api/datacenters"
          interval = "30s"
          timeout  = "5s"
        }

        tags = [
          "traefik.enable=true",
          "traefik.http.routers.rackd.rule=Host(`rackd.example.com`)",
        ]
      }
    }
  }
}
```
