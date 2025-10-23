# Terramate MCP Server Guidelines

## Project Structure & Module Organization
- `cmd/terramate-mcp-server/` hosts the CLI entrypoint that wires flags, creates the server, and handles shutdown.
- `sdk/terramate/` contains the Terramate Cloud API client SDK with services for Stacks, Drifts, and Memberships.
- `tools/` groups MCP tool handlers; for example see `tools/tmc` for all Terramate Cloud specific tools.
- `types/` defines shared interfaces and structs used across the project.

## Build, Test, and Development Commands

### Build Commands
- `make build` - Build optimized production binary to `bin/terramate-mcp-server`
- `make build-dev` - Build debug binary (faster, with debug info)
- `make build-docker` - Build Docker image using multi-stage build
- `make clean` - Clean build artifacts and test cache
- `make clean-all` - Clean everything including Go module cache

### Test Commands
- `make test` - Run tests with race detector and coverage (timeout: 10m)
- `make test-coverage` - Run tests and display coverage report
- `make test-race` - Run tests with race detector
- `make test-short` - Run tests, skip slow tests

### Lint and Format Commands
- `make lint` - Run golangci-lint with 5m timeout
- `make lint-fix` - Run linters and auto-fix issues
- `make fmt` - Format all Go code using `gofmt -s`
- `make fmt-check` - Check if code is formatted (fails if not)
- `make vet` - Run `go vet` on all packages
- `make check` - Run all checks: format, vet, lint, and test

### Dependency Commands
- `make deps` - Download and tidy dependencies
- `make verify` - Verify dependencies integrity
- `make tidy-check` - Check if go.mod and go.sum are tidy

### Run Commands
- `make run` - Build and run the server (requires TERRAMATE_API_KEY and TERRAMATE_REGION env vars)
- `make dev` - Build and run in development mode
- `make run-docker` - Build and run in Docker container

### Docker Commands
- `make build-docker` - Build Docker image tagged with version and latest
- `make docker-push` - Push Docker image to registry (requires GITHUB_TOKEN and GITHUB_USER)
- `make docker-login` - Login to GitHub Container Registry

### Utility Commands
- `make info` - Display build and Docker information
- `make help` - Display help with all available targets
- `make ci` - Run all CI checks (lint, test, build)

## Coding Style & Naming Conventions
- Follow standard Go formatting enforced by `go fmt` and `golangci-lint`; commit only gofmt'd files.
- Keep packages lower-case nouns; exported identifiers PascalCase; wrap errors with `%w`.
- Name tests `TestXxx`/`BenchmarkXxx`; favor table-driven cases where practical.
- SDK methods should match OpenAPI operation patterns (List, Get, Create, etc.).
- Reuse common types like `ListOptions` and `PaginatedResult` across services.

## Testing Guidelines
- Default to `make test` which runs with race detector and 10m timeout.
- Tests run with `-v -race -coverprofile=coverage.out -timeout=10m` flags.
- All SDK methods should have comprehensive tests covering:
  - Response parsing
  - Query parameter handling
  - Input validation
  - API error handling
  - Authentication headers
  - Context cancellation and timeout
- Use table-driven tests for validation scenarios.
- Aim for high coverage on new code and document any gaps in the PR body.

## Commit & Pull Request Guidelines
- Use short, imperative commit subjects like `feat: add drift service to sdk` or `fix: docker build issue`.
- Rebase or squash before raising a PR to keep history linear.
- PRs should state intent, link issues, list test commands, and add evidence for UX or API changes.
- Flag configuration changes (new flags, env vars) in the PR description and alert reviewers when credentials are required.

## Security & Configuration Tips
- Never commit secrets; use environment variables for API keys and tokens.
- Required environment variables:
  - `TERRAMATE_API_KEY` - Terramate Cloud API key (for running the server)
  - `TERRAMATE_REGION` - Terramate Cloud region: `eu` or `us` (for running the server)
  - `GITHUB_TOKEN` - GitHub token with packages:write scope (for Docker push)
  - `GITHUB_USER` - GitHub username (for Docker push)
- Docker images are published to `ghcr.io/terramate-io/terramate-mcp-server`.
