# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Add JWT token authentication support for user-level credentials
- Add `--credential-file` flag to specify custom JWT credential file location
- Add `LoadJWTFromFile()` function to load credentials from `~/.terramate.d/credentials.tmrc.json`
- Add `Credential` interface for authentication abstraction
- Add `JWTCredential` implementation with automatic expiration checking
- Add `NewClientWithJWT()` convenience constructor for JWT-based authentication
- Add `NewClientWithAPIKey()` convenience constructor for API key authentication
- Add comprehensive test coverage for JWT authentication flows
- Add integration tests for MCP server with JWT credentials
- Add auto-refresh pattern for JWT tokens in AI assistant configurations (runs `terramate cloud info` before starting server)

### Changed
- Change `NewClient()` to accept `Credential` interface instead of raw API key string
- Change MCP server to auto-load JWT credentials from default location when no API key provided
- Update all tests to use new `NewClientWithAPIKey()` constructor
- Update Makefile `run`, `dev`, and `docker/run` targets to support both JWT and API key authentication
- Update Dockerfile with documentation for both authentication methods

### Security
- Add automatic JWT token expiration validation before each API request
- Add helpful error messages guiding users to refresh expired credentials

## [0.0.1] - 2025-11-01

### Added

- Initial release of Terramate MCP Server
- Model Context Protocol (MCP) server implementation for Terramate Cloud integration
- Terramate Cloud SDK with comprehensive API client
  - Stacks service for stack management
  - Drifts service for drift detection
  - Deployments service for CI/CD tracking
  - Review Requests service for PR/MR integration
  - Previews service for terraform plan debugging
  - Memberships service for organization management
- MCP Tools for AI assistants:
  - `tmc_authenticate` - Authentication and organization membership
  - `tmc_list_stacks` - List and filter stacks with pagination
  - `tmc_get_stack` - Get detailed stack information
  - `tmc_list_drifts` - List drift detection runs
  - `tmc_get_drift` - Get drift details with terraform plans
  - `tmc_list_review_requests` - List pull/merge requests
  - `tmc_get_review_request` - Get PR details with terraform plans
  - `tmc_get_stack_preview_logs` - Debug failed terraform plans
  - `tmc_list_deployments` - List CI/CD deployments
  - `tmc_get_stack_deployment` - Get deployment details
  - `tmc_get_deployment_logs` - Debug failed deployments
- HTTP client with automatic retry logic and exponential backoff
- Graceful shutdown handling for SIGINT and SIGTERM signals
- Configuration via environment variables and CLI flags
- Support for EU and US regions
- Docker support with multi-stage builds
- Comprehensive test suite with race detector and coverage
- GitHub Actions CI/CD pipeline
- Automated Docker image publishing to GitHub Container Registry
- Documentation and usage examples

[unreleased]: https://github.com/terramate-io/terramate-mcp-server/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/terramate-io/terramate-mcp-server/releases/tag/v0.0.1

