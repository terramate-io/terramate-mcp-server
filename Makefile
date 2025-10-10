# Terramate MCP Server - Build Configuration

.PHONY: all build clean test lint

# Build output directory
BUILD_DIR := bin

# Go build flags
GO_BUILD_FLAGS := -ldflags="-s -w" -trimpath

# Default target
all: build

# Build all binaries
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GO_BUILD_FLAGS) -o $(BUILD_DIR)/terramate-mcp-server ./cmd/terramate-mcp-server

build-docker:
	docker build . -t terramate-mcp-server:latest -f Dockerfile

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Run tests
test:
	go test -v ./...

# Run linter
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...


# Run standalone server
run:
	./$(BUILD_DIR)/terramate-mcp-server

# Display help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  all                    - Build all binaries"
	@echo "  build                  - Build binary"
	@echo "  build-docker           - Build Docker image"
	@echo ""
	@echo "Test targets:"
	@echo "  test                   - Run tests"
	@echo "  lint                   - Run linter"
	@echo "  fmt                    - Format code"
	@echo ""
	@echo "Run targets:"
	@echo "  run                    - Run MCP server"
	@echo ""
	@echo "Utility targets:"
	@echo "  clean                  - Clean build artifacts"
	@echo "  deps                   - Install dependencies"
	@echo "  help                   - Show this help"
