# Terramate MCP Server Guidelines for Claude

This file provides guidance to Claude (claude.ai) when working with code in this repository.

## Project Overview

This is the Terramate MCP Server - a Model Context Protocol server that integrates Terramate Cloud with AI assistants like Claude, ChatGPT, and Cursor. It enables natural language interactions with Terramate Cloud for managing Infrastructure as Code workflows.

## Project Structure & Module Organization

- `cmd/terramate-mcp-server/` hosts the CLI entrypoint that wires flags, creates the server, and handles shutdown.
- `sdk/terramate/` contains the Terramate Cloud API client SDK with services for Stacks, Drifts, Deployments, ReviewRequests, Previews, and Memberships.
- `tools/` groups MCP tool handlers; for example see `tools/tmc` for all Terramate Cloud specific tools.
- `types/` defines shared interfaces and structs used across the project.

## Build, Test, and Development Commands

### Build Commands

- `make build` - Build optimized production binary to `bin/terramate-mcp-server`
- `make build/dev` - Build debug binary (faster, with debug info)
- `make docker/build` - Build Docker image using multi-stage build
- `make clean` - Clean build artifacts and test cache
- `make clean/all` - Clean everything including Go module cache

### Test Commands

- `make test` - Run tests with race detector and coverage (timeout: 10m)
- `make test/coverage` - Run tests and display coverage report
- `make test/race` - Run tests with race detector
- `make test/short` - Run tests, skip slow tests

### Lint and Format Commands

- `make lint` - Run golangci-lint with 5m timeout
- `make lint/fix` - Run linters and auto-fix issues
- `make fmt` - Format all Go code using `gofmt -s`
- `make fmt/check` - Check if code is formatted (fails if not)
- `make vet` - Run `go vet` on all packages
- `make check` - Run all checks: format, vet, lint, and test

### Dependency Commands

- `make deps` - Download and tidy dependencies
- `make verify` - Verify dependencies integrity
- `make tidy/check` - Check if go.mod and go.sum are tidy

### Run Commands

- `make run` - Build and run the server (supports JWT or API key auth, requires TERRAMATE_REGION env var)
- `make dev` - Build and run in development mode (supports JWT or API key auth)
- `make docker/run` - Build and run in Docker container (supports JWT or API key auth)

### Docker Commands

- `make docker/build` - Build Docker image tagged with version and latest
- `make docker/push` - Push Docker image to registry (requires GITHUB_TOKEN and GITHUB_USER)
- `make docker/login` - Login to GitHub Container Registry

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
- Use generic query builder helpers from `client.go` (addPagination, addStringSlice, etc.) to reduce code duplication.

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

### Conventional Commits
Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification for all commit messages:

**Format:**
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:**
- `feat:` - A new feature (triggers MINOR version bump)
- `fix:` - A bug fix (triggers PATCH version bump)
- `docs:` - Documentation only changes
- `style:` - Code style changes (formatting, missing semicolons, etc.)
- `refactor:` - Code changes that neither fix a bug nor add a feature
- `perf:` - Performance improvements
- `test:` - Adding or updating tests
- `build:` - Changes to build system or dependencies
- `ci:` - Changes to CI configuration files and scripts
- `chore:` - Other changes that don't modify src or test files
- `revert:` - Reverts a previous commit

**Scope (optional):**
- `sdk` - Changes to the SDK (`sdk/terramate/`)
- `tools` - Changes to MCP tools (`tools/`)
- `cmd` - Changes to CLI (`cmd/terramate-mcp-server/`)
- `docker` - Docker-related changes
- `deps` - Dependency updates

**Breaking Changes:**
- Add `!` after type/scope: `feat!:` or `feat(sdk)!:`
- Add `BREAKING CHANGE:` footer with description (triggers MAJOR version bump)

**Examples:**
```
feat(sdk): add deployments service with log streaming

Add new Deployments service to SDK with support for listing
workflows, stack deployments, and streaming logs.

Closes #123
```

```
fix(tools): correct pagination handling in list_stacks

The per_page parameter was not being passed correctly to the API,
causing pagination to fail for large stack lists.

Fixes #456
```

```
feat(sdk)!: change authentication to use API key instead of OAuth

BREAKING CHANGE: The Client constructor now requires an API key
instead of OAuth credentials. Update all client initialization code.

Migration guide:
- Old: terramate.NewClient(oauth)
- New: terramate.NewClient(apiKey)
```

```
docs: update README with new deployment tools examples
```

**Pull Request Guidelines:**
- Rebase or squash before raising a PR to keep history linear.
- PRs should state intent, link issues, list test commands, and add evidence for UX or API changes.
- Flag configuration changes (new flags, env vars) in the PR description and alert reviewers when credentials are required.

### Changelog Maintenance

The project uses [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format. The changelog must be updated with every notable change.

**When to Update:**
- Update `CHANGELOG.md` in the SAME PR that introduces the change
- Add entries to the `[Unreleased]` section under the appropriate category
- Do NOT create version sections - maintainers do this during release

**Categories:**
- `Added` - New features, tools, or capabilities
- `Changed` - Changes in existing functionality
- `Deprecated` - Soon-to-be removed features
- `Removed` - Removed features or APIs
- `Fixed` - Bug fixes
- `Security` - Security fixes or improvements

**Entry Format:**
- Use present tense, active voice: "Add drift detection" not "Added drift detection"
- Start with a verb when possible
- Be specific and concise
- Reference PR/issue numbers when applicable

**Examples:**
```markdown
## [Unreleased]

### Added
- Add `tmc_get_deployment_logs` tool for debugging failed deployments (#123)
- Add retry logic with exponential backoff to HTTP client
- Add support for custom base URL configuration

### Changed
- Change default timeout from 30s to 60s for API requests (#456)
- Improve error messages for authentication failures

### Fixed
- Fix pagination bug in `tmc_list_stacks` when per_page > 100 (#789)
- Fix race condition in graceful shutdown handler
```

**Release Process (Maintainers Only):**
When creating a release:
1. Update version in `internal/version/version.go` (e.g., `0.0.2`)
2. Move all `[Unreleased]` items to a new version section in `CHANGELOG.md`
3. Add the release date: `## [0.0.2] - 2025-11-12`
4. Update the comparison links at the bottom of `CHANGELOG.md`
5. Create a new empty `[Unreleased]` section
6. Commit with message: `chore: release v0.0.2`
7. Push to main branch
8. Create GitHub release with tag `v0.0.2` via GitHub UI
9. GitHub Actions will automatically build and publish Docker image

**What NOT to Include:**
- Internal refactorings that don't affect users
- Test-only changes (unless they add new testing capabilities)
- Minor typo fixes in code comments
- Dependency updates (unless they fix a security issue or add new functionality)

## Authentication

### JWT Token Authentication (Preferred)

**Why JWT is Preferred:**
- **No admin required**: Users can authenticate themselves via `terramate cloud login`
- **User-level permissions**: Actions are tracked per user for better audit trails
- **Self-service**: No need to request API keys from organization admins
- **Multiple providers**: Google, GitHub, GitLab, SSO support
- **Automatic token refresh**: Tokens refresh transparently when expired - zero maintenance

**Organization API Keys Require Admin:**
Organization API keys can only be created and managed by organization administrators in Terramate Cloud. This makes JWT authentication the preferred method for individual developers.

### Configuration

**JWT Authentication (Recommended):**
```bash
# 1. Login once
terramate cloud login

# 2. Run server (auto-loads from ~/.terramate.d/credentials.tmrc.json)
make run TERRAMATE_REGION=eu

# Or run directly
./bin/terramate-mcp-server --region eu
```

**API Key Authentication (Requires Admin):**
```bash
# Set environment variables
export TERRAMATE_API_KEY="your-api-key"  # Requires org admin to generate
export TERRAMATE_REGION="eu"

# Run server
make run
```

## Authentication Architecture

### Credential Abstraction

The project uses a credential abstraction layer to support multiple authentication methods:

**Credential Interface** (`sdk/terramate/credential.go`):
```go
type Credential interface {
    ApplyCredentials(req *http.Request) error
    Name() string
}
```

**Implementations:**
- `JWTCredential` - JWT tokens from `~/.terramate.d/credentials.tmrc.json`
- `APIKeyCredential` - Organization API keys (requires admin to issue an API key)

### JWT Token Authentication (Preferred)

**Why JWT is Preferred:**
- **Self-service**: Users authenticate via `terramate cloud login` without admin intervention
- **Admin requirement**: Organization API keys can ONLY be created by organization administrators
- **User-level permissions**: Actions tracked per user for audit trails
- **Multiple providers**: Google, GitHub, GitLab, SSO support

**JWT Credential File:**
- **Location**: `~/.terramate.d/credentials.tmrc.json`
- **Format**:
  ```json
  {
    "provider": "Google",
    "id_token": "eyJhbGc...",
    "refresh_token": "1//0g..."
  }
  ```
- **Managed by**: Terramate CLI (`terramate cloud login`)

**Implementation Details:**
- JWT tokens are parsed ONLY to extract provider information (for display purposes)
- **Automatic token refresh**: When API returns 401 Unauthorized, the server automatically refreshes the token
- **File watching**: The server watches the credential file and reloads tokens when Terramate CLI updates them
- **Thread-safe**: All credential operations use mutex protection for concurrent access
- **Atomic file updates**: Credential file updates are atomic to prevent corruption
- Uses `Authorization: Bearer <token>` header
- **Zero maintenance**: No manual token refresh or server restarts needed

**Automatic Token Refresh:**
The MCP server implements a hybrid approach for seamless token management:
1. **Reactive Refresh**: When API returns 401, server refreshes token and retries request
2. **File Watching**: Server watches `~/.terramate.d/credentials.tmrc.json` for external updates
3. **Shared Credentials**: Both MCP server and Terramate CLI safely share the same credential file
4. **Atomic Updates**: File updates use atomic operations to prevent race conditions

**Security Note:**
The client does NOT validate JWT expiration locally. This is intentional and follows security best practices:
- Client-side parsing uses `ParseUnverified()` which doesn't verify signatures
- Making security decisions based on unverified data would be unsafe
- The API server is the authoritative source for token validation
- 401 errors trigger automatic token refresh - transparent to users

### API Key Authentication (issuing an organization API key requires admin privileges)

**⚠️ Requires Admin Privileges:**
Organization API keys can only be created and managed by organization administrators. This creates a bottleneck for individual developers and is why JWT authentication is strongly preferred.

**Usage:**
- Uses HTTP Basic Auth with API key as username, empty password
- Never expires from client perspective
- Environment variable: `TERRAMATE_API_KEY`
- CLI flag: `--api-key`

### Authentication Priority (in code)

When initializing the MCP server:
1. Check for `TERRAMATE_API_KEY` environment variable or `--api-key` flag
2. If API key present → use `APIKeyCredential` (show deprecation warning)
3. If no API key → load JWT from credential file
4. If neither → return helpful error message

### SDK Client Constructors

**Main constructor** (accepts any Credential):
```go
client, err := terramate.NewClient(credential, opts...)
```

**Convenience constructors:**
```go
// For API key (backward compatible)
client, err := terramate.NewClientWithAPIKey(apiKey, opts...)

// For JWT token
client, err := terramate.NewClientWithJWT(jwtToken, opts...)

// Load JWT from file
cred, err := terramate.LoadJWTFromFile("~/.terramate.d/credentials.tmrc.json")
client, err := terramate.NewClient(cred, opts...)
```

### When Adding New SDK Methods

Always use the client's credential for authentication:
```go
func (s *Service) SomeMethod(ctx context.Context, ...) error {
    // The client's credential is automatically applied via newRequest()
    req, err := s.client.newRequest(ctx, "GET", path, nil)
    // ...
}
```

**Do NOT:**
- Manually set Authorization headers
- Assume API key is always available
- Perform client-side JWT validation or expiration checking

**DO:**
- Let the credential interface handle authentication
- Trust the client's newRequest() method
- Let the API server validate credentials (it's the source of truth)
- Write tests for both JWT and API key scenarios

## Security & Configuration Tips

### Authentication
- **Never commit secrets**: Use environment variables or credential files
- **JWT credentials**: Stored in `~/.terramate.d/credentials.tmrc.json` with `0600` permissions
- **Prefer JWT over API key**: JWT enables self-service and better audit trails
- **API keys require admin**: Only organization administrators can create API keys

### Environment Variables

**For Running Server:**
- `TERRAMATE_REGION` - Terramate Cloud region: `eu` or `us` (required)
- `TERRAMATE_API_KEY` - Organization API key (deprecated, for backward compatibility)
- `TERRAMATE_CREDENTIAL_FILE` - Custom JWT credential file path (optional, defaults to `~/.terramate.d/credentials.tmrc.json`)

**For Docker Push:**
- `GITHUB_TOKEN` - GitHub token with packages:write scope
- `GITHUB_USER` - GitHub username

### Docker
- Docker images are published to `ghcr.io/terramate-io/terramate-mcp-server`
- For JWT auth in Docker: mount `~/.terramate.d` directory as read-only volume
- Example: `docker run -v ~/.terramate.d:/root/.terramate.d:ro ...`

### Credential File Security
- File permissions MUST be `0600` (read/write owner only) - enforced by the SDK
- The SDK will refuse to load credential files with insecure permissions
- Never commit credential files to git
- `.terramate.d/` should be in `.gitignore`
- MCP server watches the credential file for changes and automatically reloads tokens

## Security Best Practices

### 🔒 Preventing Token Leakage

**CRITICAL: Never expose tokens, API keys, or credentials in:**
- Error messages
- Log messages
- Debug output
- HTTP response bodies in error messages
- Stack traces
- Test output (unless sanitized)

**When handling errors:**
```go
// ❌ BAD: Leaks token in error message
return fmt.Errorf("refresh failed: %s", string(responseBody))

// ✅ GOOD: Parse JSON safely, extract only safe fields
var errResp struct {
    Error string `json:"error"`
}
if err := json.Unmarshal(body, &errResp); err == nil {
    return fmt.Errorf("refresh failed: %s", errResp.Error)
}
return fmt.Errorf("refresh failed (status %d)", statusCode)
```

**When logging:**
```go
// ❌ BAD: Logs token value
log.Printf("Token: %s", token)

// ✅ GOOD: Generic log message
log.Printf("JWT token refreshed successfully")

// ✅ GOOD: Log error without token
log.Printf("Warning: failed to reload credential: %v", err)
```

**When handling HTTP responses:**
```go
// ❌ BAD: Includes raw body in error (may contain tokens)
apiErr := &APIError{Message: string(body)}

// ✅ GOOD: Parse JSON safely, extract only error fields
apiErr := &APIError{Message: "API request failed"}
if isJSONContentType(resp.Header.Get("Content-Type")) {
    var errResp ErrorResponse
    if err := json.Unmarshal(body, &errResp); err == nil {
        apiErr.Message = errResp.Error // Only safe parsed field
    }
}
```

### Security Checklist for New Code

When adding or modifying code that handles credentials:

- [ ] **Error Messages**: Never include tokens, API keys, or raw HTTP response bodies
- [ ] **Logging**: Use generic messages, never log credential values
- [ ] **JSON Parsing**: Parse error responses safely, extract only known safe fields
- [ ] **HTTP Bodies**: Never convert response bodies to strings for error messages without parsing
- [ ] **Test Output**: In tests, only log token prefixes (e.g., `token[:20]+"..."`) if needed
- [ ] **File Permissions**: Always validate credential file permissions (`0600`)
- [ ] **Thread Safety**: Use mutexes for concurrent credential access
- [ ] **Input Validation**: Validate all inputs before processing
- [ ] **HTTPS Only**: Never use HTTP for credential transmission
- [ ] **Context Timeouts**: Use context timeouts for all network operations

### Common Security Anti-Patterns to Avoid

**1. Token Leakage in Errors:**
```go
// ❌ BAD
return fmt.Errorf("failed: %s", string(httpResponseBody))

// ✅ GOOD
return fmt.Errorf("failed: %s", parseSafeError(httpResponseBody))
```

**2. Logging Credentials:**
```go
// ❌ BAD
log.Printf("Using token: %s", token)

// ✅ GOOD
log.Printf("Using JWT authentication")
```

**3. Including Raw Bodies:**
```go
// ❌ BAD
err := fmt.Errorf("API error: %s", string(body))

// ✅ GOOD
err := parseAPIError(resp, body) // Safely parses JSON
```

**4. Debug Output:**
```go
// ❌ BAD
fmt.Printf("Token: %v\n", credential)

// ✅ GOOD
fmt.Printf("Credential type: %s\n", credential.Name())
```

### Security Review Process

Before committing code that handles credentials:

1. **Search for token leakage:**
   ```bash
   grep -r "fmt.*token\|log.*token\|string(body)" --include="*.go"
   ```

2. **Verify error handling:**
   - Check all `fmt.Errorf()` calls with `%s` or `%v` formatting
   - Ensure HTTP response bodies are parsed, not converted to strings
   - Verify error messages don't include credential values

3. **Check logging:**
   - Search for `log.Printf` or `log.Println` with credential variables
   - Ensure all log messages are generic

4. **Test security:**
   - Run tests and verify no tokens appear in output
   - Check error messages don't expose sensitive data
   - Verify file permissions are enforced

5. **Review HTTP handling:**
   - Ensure all API calls use HTTPS
   - Verify response bodies are parsed safely
   - Check error handling doesn't leak response bodies

## SDK Development

When adding new SDK endpoints:

1. Check the OpenAPI spec (`openapi.yml`) for exact endpoint definition
2. Add types to `sdk/terramate/types.go` with proper JSON tags
3. Create service file (e.g., `sdk/terramate/newservice.go`)
4. Use generic query builders from `client.go` (don't duplicate code)
5. Add comprehensive tests in `sdk/terramate/newservice_test.go`
6. Update `sdk/terramate/client.go` to initialize the service
7. Verify against OpenAPI spec before committing

## MCP Tools Development

When adding new MCP tools:

1. Implement SDK endpoint first (see above)
2. Create tool in `tools/tmc/toolname.go` following existing patterns
3. Provide clear descriptions in the tool schema
4. Document workflow in the description
5. Add comprehensive tests in `tools/tmc/toolname_test.go`
6. Register tool in `tools/handlers.go`
7. Document use cases in `README.md`

## Documentation

- Main `README.md` - MCP server usage, tool descriptions, use cases
- `sdk/terramate/README.md` - Complete SDK documentation and API reference
- `AGENTS.md` - Guidelines for AI coding agents
- Keep documentation in sync with implementation
