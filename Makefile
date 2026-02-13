# Terramate MCP Server - Build Configuration

# Default target
.DEFAULT_GOAL := help

# Build output directory
BUILD_DIR := bin
BINARY_NAME := terramate-mcp-server

# Docker configuration
DOCKER_REGISTRY := ghcr.io
DOCKER_ORG := terramate-io
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(DOCKER_ORG)/$(BINARY_NAME)

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go build flags
LDFLAGS := -s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)
GO_BUILD_FLAGS := -ldflags="$(LDFLAGS)" -trimpath

# Go commands (use asdf if available, otherwise fall back to system go)
ASDF := $(shell command -v asdf 2> /dev/null)
ifdef ASDF
# Set GOROOT to asdf's golang installation
ASDF_GOLANG_VERSION := $(shell asdf current golang 2>/dev/null | awk '{print $$2}')
export GOROOT := $(HOME)/.asdf/installs/golang/$(ASDF_GOLANG_VERSION)/go
GOCMD := asdf exec go
else
GOCMD := go
endif
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOCLEAN := $(GOCMD) clean
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
TOOLS_BIN := $(BUILD_DIR)/tools
GOLANGCI_LINT := $(TOOLS_BIN)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.5.0
# Derive Go toolchain version from the active Go installation
# to avoid version mismatches between build tools and golangci-lint
ifdef ASDF
GO_VERSION := $(ASDF_GOLANG_VERSION)
else
GO_VERSION := $(shell go version | awk '{print $$3}' | sed 's/go//')
endif
GOTOOLCHAIN ?= go$(GO_VERSION)
GOLANGCI_LINT_TOOLCHAIN ?= go$(GO_VERSION)

# Test flags
TEST_FLAGS := -v -race -coverprofile=coverage.out -timeout=10m
# TODO: Set to a real number when we implemented tests
COVERAGE_MIN := 0

.PHONY: all build build/dev docker/build docker/push docker/login clean test test/coverage test/race \
        lint lint/fix fmt fmt/check vet check deps verify tidy/check install uninstall \
        run dev docker/run help info ci ci/lint ci/test ci/build clean/all test/short

## Build targets

all: clean build ## Build production binary

build: ## Build optimized production binary
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/terramate-mcp-server
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build/dev: ## Build debug binary (faster, with debug info)
	@echo "Building development binary..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/terramate-mcp-server
	@echo "✅ Development binary built"

docker/build: ## Build Docker image (multi-stage build)
	@echo "Building Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker build . \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		-f Dockerfile
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest
	@echo "✅ Docker image built: $(DOCKER_IMAGE):$(VERSION)"

## Test targets

test: ## Run tests
	$(GOTEST) $(TEST_FLAGS) ./...

test/coverage: test ## Run tests and show coverage report
	@$(GOCMD) tool cover -func=coverage.out
	@echo ""
	@total=$$($(GOCMD) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$total%"; \
	if [ -n "$$total" ] && [ $$(echo "$$total < $(COVERAGE_MIN)" | bc) -eq 1 ]; then \
		echo "❌ Coverage $$total% is below minimum $(COVERAGE_MIN)%"; \
		exit 1; \
	fi

test/race: ## Run tests with race detector
	$(GOTEST) -race ./...

test/short: ## Run tests (skip slow tests)
	$(GOTEST) -short ./...

$(GOLANGCI_LINT): ## Install golangci-lint locally via go install
	@echo "Installing golangci-lint..."
	@mkdir -p $(TOOLS_BIN)
	@GOTOOLCHAIN=$(GOLANGCI_LINT_TOOLCHAIN) GOBIN=$(abspath $(TOOLS_BIN)) $(GOCMD) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "✅ golangci-lint installed at $(GOLANGCI_LINT)"

## Lint and format targets

lint: $(GOLANGCI_LINT) ## Run linters
	@echo "Running golangci-lint..."
	@env -u GOROOT GOTOOLCHAIN=local $(GOLANGCI_LINT) run --config .golangci.yml --timeout=5m

lint/fix: $(GOLANGCI_LINT) ## Run linters and auto-fix issues
	@echo "Running golangci-lint with auto-fix..."
	@env -u GOROOT GOTOOLCHAIN=local $(GOLANGCI_LINT) run --config .golangci.yml --fix --timeout=5m

fmt: ## Format code
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@echo "✅ Code formatted"

fmt/check: ## Check if code is formatted
	@echo "Checking code formatting..."
	@unformatted=$$($(GOFMT) -s -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "❌ The following files are not formatted:"; \
		echo "$$unformatted"; \
		echo "Run 'make fmt' to fix"; \
		exit 1; \
	fi
	@echo "✅ All files are properly formatted"

vet: ## Run go vet (kept for local use)
	@echo "Running go vet..."
	@$(GOCMD) vet ./...
	@echo "✅ No issues found"

check: fmt/check vet lint test ## Run all checks (format, vet, lint, test)

## Dependency targets

deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "✅ Dependencies updated"

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GOMOD) verify
	@echo "✅ Dependencies verified"

tidy/check: ## Check if go.mod and go.sum are tidy
	@echo "Checking if go.mod and go.sum are tidy..."
	@$(GOMOD) tidy
	@git diff --exit-code go.mod go.sum || (echo "❌ go.mod or go.sum not tidy. Run 'make deps'" && exit 1)
	@echo "✅ go.mod and go.sum are tidy"

## Install targets

install: build ## Install binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✅ Installed to $(GOPATH)/bin/$(BINARY_NAME)"

uninstall: ## Uninstall binary from $GOPATH/bin
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "✅ Uninstalled"

## Run targets

run: build ## Build and run the MCP server (supports JWT or API key auth)
	@echo "Starting MCP server..."
	@if [ -n "${TERRAMATE_API_KEY}" ]; then \
		echo "Using API key authentication"; \
		./$(BUILD_DIR)/$(BINARY_NAME) --api-key=${TERRAMATE_API_KEY} --region=${TERRAMATE_REGION}; \
	else \
		echo "Using JWT authentication from credential file"; \
		./$(BUILD_DIR)/$(BINARY_NAME) --region=${TERRAMATE_REGION}; \
	fi

dev: build/dev ## Build and run in development mode (supports JWT or API key auth)
	@echo "Starting MCP server (development mode)..."
	@if [ -n "${TERRAMATE_API_KEY}" ]; then \
		echo "Using API key authentication"; \
		./$(BUILD_DIR)/$(BINARY_NAME) --api-key=${TERRAMATE_API_KEY} --region=${TERRAMATE_REGION}; \
	else \
		echo "Using JWT authentication from credential file"; \
		./$(BUILD_DIR)/$(BINARY_NAME) --region=${TERRAMATE_REGION}; \
	fi

docker/run: docker/build ## Build and run Docker container (supports JWT or API key auth)
	@echo "Starting MCP server in Docker..."
	@if [ -n "${TERRAMATE_API_KEY}" ]; then \
		echo "Using API key authentication"; \
		docker run --rm -it \
			-e TERRAMATE_API_KEY=${TERRAMATE_API_KEY} \
			-e TERRAMATE_REGION=${TERRAMATE_REGION} \
			$(DOCKER_IMAGE):latest; \
	else \
		echo "Using JWT authentication (mounting credential directory)"; \
		docker run --rm -it \
			-v ${HOME}/.terramate.d:/root/.terramate.d:ro \
			-e TERRAMATE_REGION=${TERRAMATE_REGION} \
			$(DOCKER_IMAGE):latest; \
	fi

docker/push: docker/build ## Push Docker image to registry
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest
	@echo "✅ Docker images pushed"

docker/login: ## Login to GitHub Container Registry
	@echo "Logging in to $(DOCKER_REGISTRY)..."
	@echo ${GITHUB_TOKEN} | docker login $(DOCKER_REGISTRY) -u ${GITHUB_USER} --password-stdin
	@echo "✅ Logged in to $(DOCKER_REGISTRY)"

## Utility targets

clean: ## Clean build artifacts and test cache
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@$(GOCLEAN) -cache -testcache
	@echo "✅ Cleaned"

clean/all: clean ## Clean everything including dependencies
	@echo "Cleaning dependencies..."
	@$(GOCLEAN) -modcache
	@echo "✅ All cleaned"

info: ## Display build information
	@echo "Build Information:"
	@echo "  Version:      $(VERSION)"
	@echo "  Git Commit:   $(GIT_COMMIT)"
	@echo "  Build Time:   $(BUILD_TIME)"
	@echo "  Go Version:   $$($(GOCMD) version)"
	@echo "  Build Dir:    $(BUILD_DIR)"
	@echo "  Binary:       $(BINARY_NAME)"
	@echo ""
	@echo "Docker Information:"
	@echo "  Registry:     $(DOCKER_REGISTRY)"
	@echo "  Organization: $(DOCKER_ORG)"
	@echo "  Image:        $(DOCKER_IMAGE)"
	@echo "  Tags:         $(VERSION), latest"

## CI targets (used by GitHub Actions)

ci/lint: fmt/check lint ## Run all lint checks (CI)

ci/test: test/coverage ## Run tests with coverage (CI)

ci/build: build ## Build for CI

ci: ci/lint ci/test ci/build ## Run all CI checks

help: ## Display this help message
	@echo "Terramate MCP Server - Makefile"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^[a-zA-Z/_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment Variables:"
	@echo "  TERRAMATE_API_KEY  - Terramate Cloud API key (required for run)"
	@echo "  TERRAMATE_REGION   - Terramate Cloud region: eu or us (required for run)"
	@echo "  VERSION            - Version to embed (default: git describe)"
	@echo "  GITHUB_USER        - GitHub username (required for docker-login)"
	@echo "  GITHUB_TOKEN       - GitHub token with packages:write (required for docker-push)"
