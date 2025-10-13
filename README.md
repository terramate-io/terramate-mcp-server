# Terramate MCP Server

[![Go Report Card](https://goreportcard.com/badge/github.com/terramate-io/terramate-mcp-server)](https://goreportcard.com/report/github.com/terramate-io/terramate-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The Terramate MCP Server integrates [Terramate CLI](https://github.com/terramate-io/terramate) and [Terramate Cloud](https://terramate.io) with AI assistants like ChatGPT, Claude, Cursor, and any LLM that supports the [Model Context Protocol (MCP)](https://github.com/mark3labs/mcp-go).

This server enables natural language interactions with your Terramate Cloud organization, allowing you to query deployments, stacks, drifts, and manage Infrastructure as Code (IaC) workflows directly from your AI assistant.

## Features

- 🔐 **Secure Authentication** - API key-based authentication with Terramate Cloud
- 🌍 **Multi-Region Support** - EU and US region endpoints
- 🛠️ **MCP Tools** - Extensible tool system for Terramate operations
- 🔄 **Automatic Retries** - Built-in retry logic for transient failures
- 📊 **Comprehensive Testing** - 88%+ test coverage
- 🚀 **Production Ready** - Graceful shutdown, timeouts, and error handling

## Installation

### Prerequisites

- Go 1.25.0 or later
- A [Terramate Cloud](https://cloud.terramate.io) account
- A Terramate Cloud API key ([generate one here](https://cloud.terramate.io/o/YOUR_ORG/settings/api-keys))

### From Source

```bash
git clone https://github.com/terramate-io/terramate-mcp-server.git
cd terramate-mcp-server
make build
```

The binary will be available at `bin/terramate-mcp-server`.

### Using Docker

```bash
docker build -t terramate-mcp-server .
docker run --rm -e TERRAMATE_API_KEY=your-key -e TERRAMATE_REGION=eu terramate-mcp-server
```

## Configuration

The server accepts configuration via command-line flags or environment variables:

| Flag         | Environment Variable | Required | Default                    | Description                           |
| ------------ | -------------------- | -------- | -------------------------- | ------------------------------------- |
| `--api-key`  | `TERRAMATE_API_KEY`  | ✅       | -                          | Your Terramate Cloud API key          |
| `--region`   | `TERRAMATE_REGION`   | ⚠️\*     | -                          | Terramate Cloud region (`eu` or `us`) |
| `--base-url` | `TERRAMATE_BASE_URL` | ❌       | `https://api.terramate.io` | Custom API base URL                   |

\* Required when using the default base URL. Optional if `--base-url` is specified.

### Region Endpoints

- **EU**: `https://api.terramate.io` (default)
- **US**: `https://api.us.terramate.io`

When using `--region eu`, the server automatically uses the EU endpoint. When using `--region us`, it uses the US endpoint.

## Usage

### Running the Server

#### Standalone Mode

```bash
# Using environment variables
export TERRAMATE_API_KEY="your-api-key"
export TERRAMATE_REGION="eu"
./bin/terramate-mcp-server

# Using command-line flags
./bin/terramate-mcp-server --api-key="your-api-key" --region="eu"

# Custom base URL (bypasses region)
./bin/terramate-mcp-server --api-key="your-api-key" --base-url="https://custom.api.example.com"
```

#### With Docker

```bash
docker run --rm -it \
  -e TERRAMATE_API_KEY="your-api-key" \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest
```

### Integrating with AI Assistants

The server communicates via stdio using the Model Context Protocol. Configure your AI assistant to use this server:

#### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "/path/to/terramate-mcp-server",
      "env": {
        "TERRAMATE_API_KEY": "your-api-key",
        "TERRAMATE_REGION": "eu"
      }
    }
  }
}
```

#### Cursor

Add to your Cursor MCP settings:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "/path/to/bin/terramate-mcp-server",
      "args": ["--api-key", "your-api-key", "--region", "eu"]
    }
  }
}
```

## Available Tools

### `tmc_authenticate`

Authenticates with Terramate Cloud and retrieves organization membership information.

**Parameters:** None

**Returns:**

```json
{
  "authenticated": true,
  "organization_uuid": "org-uuid",
  "organization_name": "my-org",
  "organization_display_name": "My Organization",
  "role": "admin",
  "status": "active",
  "memberships": [...]
}
```

**Example Usage:**

```
User: "Authenticate with Terramate Cloud"
Assistant: *calls tmc_authenticate*
```

### Future Tools

The following tools are planned:

- `tmc_list_stacks` - List all stacks in your organization
- `tmc_get_stack` - Get detailed information about a specific stack
- `tmc_list_deployments` - List recent deployments
- `tmc_list_drifts` - List detected configuration drifts
- `tmc_list_alerts` - List active alerts

## Development

### Building

```bash
# Build production binary
make build

# Build debug binary (faster, includes debug symbols)
make build-dev

# Build Docker image
make build-docker
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test -v ./sdk/terramate/...
```

Current test coverage:

- **sdk/terramate**: 88.6%
- **tools**: 100%
- **tools/tmc**: 91.3%

### Linting

```bash
# Run linters
make lint

# Auto-fix linting issues
make lint-fix

# Format code
make fmt
```

### Project Structure

```
.
├── cmd/
│   └── terramate-mcp-server/    # Main server entry point
│       ├── main.go              # CLI setup and configuration
│       └── server.go            # MCP server implementation
├── sdk/
│   └── terramate/               # Terramate Cloud API client
│       ├── client.go            # HTTP client with retries
│       ├── errors.go            # Error types
│       ├── memberships.go       # Memberships API
│       └── types.go             # API data models
├── tools/
│   ├── handlers.go              # Tool registration
│   └── tmc/
│       └── auth.go              # Authentication tool
├── internal/
│   └── version/                 # Version and user agent
└── Makefile                     # Build automation
```

## Architecture

### HTTP Client

The SDK includes a production-ready HTTP client with:

- **Automatic retries** on 429 (rate limit) and 5xx errors for idempotent requests (GET, HEAD, OPTIONS)
- **Exponential backoff** with context cancellation support
- **Request body size limits** (10 MiB) to prevent memory exhaustion
- **Content-type aware** JSON parsing
- **Context propagation** for timeout and cancellation

### Error Handling

All API errors are wrapped in `APIError` with helper methods:

```go
if apiErr, ok := err.(*terramate.APIError); ok {
    if apiErr.IsUnauthorized() {
        // Handle 401
    }
    if apiErr.IsServerError() {
        // Handle 5xx
    }
}
```

### Graceful Shutdown

The server handles `SIGINT` and `SIGTERM` signals gracefully:

1. Stops accepting new requests
2. Waits up to 30 seconds for in-flight requests to complete
3. Logs shutdown status

## API Documentation

### Terramate Cloud SDK

#### Creating a Client

```go
import "github.com/terramate-io/terramate-mcp-server/sdk/terramate"

// Basic client (EU region by default)
client, err := terramate.NewClient("your-api-key")

// With region
client, err := terramate.NewClient(
    "your-api-key",
    terramate.WithRegion("us"),
)

// With custom base URL
client, err := terramate.NewClient(
    "your-api-key",
    terramate.WithBaseURL("https://custom.api.example.com"),
)

// With custom timeout
client, err := terramate.NewClient(
    "your-api-key",
    terramate.WithTimeout(60 * time.Second),
)
```

#### Memberships API

```go
// List organization memberships
memberships, resp, err := client.Memberships.List(ctx)
if err != nil {
    log.Fatal(err)
}

for _, m := range memberships {
    fmt.Printf("Org: %s (%s), Role: %s\n",
        m.OrgDisplayName, m.OrgUUID, m.Role)
}
```

## Troubleshooting

### Authentication Failures

**Problem:** `Authentication failed: Invalid API key`

**Solution:**

- Verify your API key is correct
- Ensure the API key has not expired
- Check that you're using the correct region
- Regenerate the API key if necessary

### Region Errors

**Problem:** `invalid region: xyz (must be 'eu' or 'us')`

**Solution:**

- Use only `eu` or `us` as the region value
- If using a custom base URL, omit the `--region` flag

### Connection Timeouts

**Problem:** Requests time out or fail intermittently

**Solution:**

- Check your network connectivity
- Verify the API endpoint is reachable
- Increase timeout: `--base-url` with `WithTimeout()` option in code
- Check Terramate Cloud status page

### Rate Limiting

The client automatically retries on 429 responses with exponential backoff. If you consistently hit rate limits:

- Reduce request frequency
- Batch operations where possible
- Contact support for higher rate limits

## Contributing

Contributions are welcome! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make your changes and add tests
4. Run tests and linters: `make check`
5. Commit with descriptive messages
6. Push and create a pull request

### Code Standards

- Follow Go best practices and idioms
- Maintain test coverage above 80%
- Use `make fmt` before committing
- Ensure `make lint` passes
- Add godoc comments for exported types and functions

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Terramate Documentation](https://terramate.io/docs)
- 💬 [Community Discord](https://terramate.io/discord)
- 🐛 [Issue Tracker](https://github.com/terramate-io/terramate-mcp-server/issues)
- 📧 [Email Support](mailto:support@terramate.io)

## Related Projects

- [Terramate CLI](https://github.com/terramate-io/terramate) - Infrastructure as Code orchestration
- [MCP Go](https://github.com/mark3labs/mcp-go) - Model Context Protocol implementation for Go
- [Terramate Cloud](https://cloud.terramate.io) - Collaborative IaC platform

---

Built with ❤️ by the [Terramate Team](https://terramate.io)
