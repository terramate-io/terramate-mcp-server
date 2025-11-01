# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Deprecated

### Removed

### Fixed

### Security

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

