# Terramate MCP Server

[![CI](https://github.com/terramate-io/terramate-mcp-server/actions/workflows/ci.yml/badge.svg)](https://github.com/terramate-io/terramate-mcp-server/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/terramate-io/terramate-mcp-server)](https://goreportcard.com/report/github.com/terramate-io/terramate-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

The Terramate MCP Server integrates [Terramate CLI](https://github.com/terramate-io/terramate) and [Terramate Cloud](https://terramate.io) with AI assistants like ChatGPT, Claude, Cursor, and any LLM that supports the [Model Context Protocol (MCP)](https://github.com/mark3labs/mcp-go).

This server enables natural language interactions with your Terramate Cloud organization, allowing you to query deployments, stacks, drifts, and manage Infrastructure as Code (IaC) workflows directly from your AI assistant.

## Features

- 📚 **Stack Management** - List, filter, and query stacks with powerful search capabilities
- 🔍 **Drift Detection** - View drift runs and retrieve terraform plan outputs for AI-assisted reconciliation
- 🔀 **Pull Request Integration** - Review terraform plans for all stacks in PRs/MRs before merging
- 🚢 **Deployment Tracking** - Monitor CI/CD deployments, view terraform apply output, debug failures
- 🛠️ **MCP Tools** - 13 production-ready tools for Terramate Cloud operations
- 📦 **Stack Resources** - List and filter resources per stack (plan/state) by status, type, provider, and more

## Installation

### Prerequisites

- Go 1.25.0 or later
- A [Terramate Cloud](https://cloud.terramate.io) account
- Authentication credentials:
  - **Recommended**: Run `terramate cloud login` (self-service, no admin required)
  - **Alternative**: Organization API key ([requires admin to generate](https://cloud.terramate.io/o/YOUR_ORG/settings/api-keys))

### From Source

```bash
git clone https://github.com/terramate-io/terramate-mcp-server.git
cd terramate-mcp-server
make build
```

The binary will be available at `bin/terramate-mcp-server`.

### Using Docker

Pull the pre-built image from GitHub Container Registry:

```bash
# Pull the latest version
docker pull ghcr.io/terramate-io/terramate-mcp-server:latest

# Run with JWT authentication (recommended)
# First: terramate cloud login
docker run --rm -it \
  -v ~/.terramate.d:/root/.terramate.d:ro \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest

# Or with API key (issuing an organization API key requires admin privileges)
docker run --rm -it \
  -e TERRAMATE_API_KEY="your-api-key" \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest
```

> **Apple Silicon (M1/M2/M3/M4):** The Docker image is built for `linux/amd64`. On Apple Silicon Macs you must pass `--platform linux/amd64` to `docker run` (and `docker pull`). Docker Desktop will run the image via Rosetta emulation automatically. See the [Claude Desktop](#claude-desktop) and [Cursor](#cursor) integration examples below for ready-to-use configurations.

Or build locally:

```bash
# Build with default version info
docker build -t terramate-mcp-server .

# Build with custom version information
docker build . \
  --build-arg VERSION=1.0.0 \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S') \
  -t terramate-mcp-server:1.0.0

# Run with JWT authentication (recommended)
docker run --rm -it \
  -v ~/.terramate.d:/root/.terramate.d:ro \
  -e TERRAMATE_REGION="eu" \
  terramate-mcp-server:latest

# Or with API key (deprecated)
docker run --rm -it \
  -e TERRAMATE_API_KEY="your-api-key" \
  -e TERRAMATE_REGION="eu" \
  terramate-mcp-server:latest
```

**Docker Build Arguments:**

| Argument     | Description                    | Default   |
| ------------ | ------------------------------ | --------- |
| `VERSION`    | Version to embed in the binary | `dev`     |
| `GIT_COMMIT` | Git commit SHA to embed        | `unknown` |
| `BUILD_TIME` | Build timestamp to embed       | `unknown` |

## Authentication

The MCP server supports two authentication methods: **JWT Token** (recommended) and **API Key** (requires admin privileges).

### JWT Token Authentication (Recommended)

JWT tokens provide user-level authentication using your Terramate Cloud credentials obtained via `terramate cloud login`.

**Why JWT is Preferred:**
- ✅ **Self-service**: Any user can authenticate themselves without admin intervention
- ✅ **No admin required**: Unlike organization API keys which require admin privileges to create
- ✅ **User-level permissions**: Actions are tracked per user for better audit trails
- ✅ **Multiple providers**: Google, GitHub, GitLab, SSO support
- ✅ **Automatic token refresh**: Tokens refresh transparently when expired - zero maintenance
- ✅ **No manual credential management**: Simple `terramate cloud login` command

**Benefits:**
- User-level permissions and audit trails
- Support for multiple authentication providers (Google, GitHub, GitLab, SSO)
- Automatic token management via Terramate CLI
- No manual credential management
- **No organization admin required** - users can self-authenticate

**Setup:**

1. **Login via Terramate CLI:**
   ```bash
   terramate cloud login
   ```
   This opens your browser and stores JWT credentials in `~/.terramate.d/credentials.tmrc.json`

2. **Run the MCP server** (auto-detects credentials):
   ```bash
   ./bin/terramate-mcp-server --region eu
   ```

3. **Or specify custom credential file location:**
   ```bash
   ./bin/terramate-mcp-server --credential-file /path/to/credentials.tmrc.json --region eu
   ```

**Credential File Location:**
- Default: `~/.terramate.d/credentials.tmrc.json`
- Custom: Set via `--credential-file` flag or `TERRAMATE_CREDENTIAL_FILE` environment variable

**Supported Providers:**
- Google OAuth
- GitHub OAuth  
- GitLab OAuth
- SSO

**Token Expiration:**
JWT tokens typically expire after 1 hour. The MCP server handles this automatically:
- **Automatic Refresh**: When a token expires, the server automatically refreshes it using the refresh token
- **CLI-Compatible IDP Key**: Refresh uses the same Firebase IDP key as Terramate CLI by default, so tokens issued by `terramate cloud login` can be refreshed correctly
- **Optional Override**: Set `TMC_API_IDP_KEY` to override the default IDP key (advanced/debug use)
- **File Watching**: The server watches the credential file and automatically reloads tokens when the Terramate CLI updates them
- **Zero Downtime**: Token refresh happens transparently - no need to restart the server
- **Shared Credentials**: Both MCP server and Terramate CLI can safely use and update the same credential file

**How it works:**
1. Initial setup: Run `terramate cloud login` once
2. MCP server starts and loads the JWT token
3. Server automatically watches the credential file for changes
4. When you use Terramate CLI, it may refresh the token
5. MCP server detects the file change and reloads the new token
6. If MCP server gets a 401 error, it refreshes the token itself
7. Everything happens automatically - zero maintenance!

### API Key Authentication

API keys provide organization-level authentication.


**⚠️ Requires Admin Privileges:** Organization API keys can only be created and managed by organization administrators in Terramate Cloud. Regular users cannot generate API keys, making JWT authentication the preferred method for individual developers.

**Setup:**

```bash
./bin/terramate-mcp-server --api-key "your-api-key" --region eu
```

Or with environment variable:

```bash
export TERRAMATE_API_KEY="your-api-key"
export TERRAMATE_REGION="eu"
./bin/terramate-mcp-server
```

**Obtain API Key (Requires Admin):**
Organization administrators can generate API keys from [Terramate Cloud Settings](https://cloud.terramate.io/o/YOUR_ORG/settings/api-keys)

### Authentication Priority

When both authentication methods are available, the MCP server uses this precedence:

1. **API Key** (if `--api-key` flag or `TERRAMATE_API_KEY` env var is set)
2. **JWT Token** from credential file

This ensures backward compatibility while allowing migration to JWT authentication.

## Configuration

The server accepts configuration via command-line flags or environment variables:

| Flag                 | Environment Variable        | Required | Default                                           | Description                                                        |
| -------------------- | --------------------------- | -------- | ------------------------------------------------- | ------------------------------------------------------------------ |
| `--api-key`          | `TERRAMATE_API_KEY`         | ❌       | -                                                 | Terramate Cloud API key (deprecated, prefer JWT authentication)   |
| `--credential-file`  | `TERRAMATE_CREDENTIAL_FILE` | ❌       | `~/.terramate.d/credentials.tmrc.json`            | Path to JWT credentials file                                       |
| `--region`           | `TERRAMATE_REGION`          | ⚠️\*     | -                                                 | Terramate Cloud region (`eu` or `us`)                              |
| `--base-url`         | `TERRAMATE_BASE_URL`        | ❌       | `https://api.terramate.io`                        | Custom API base URL                                                |

\* Required when using the default base URL. Optional if `--base-url` is specified.

### Region Endpoints

- **EU**: `https://api.terramate.io` (default)
- **US**: `https://api.us.terramate.io`

When using `--region eu`, the server automatically uses the EU endpoint. When using `--region us`, it uses the US endpoint.

## Usage

### Running the Server

#### Standalone Mode

**With JWT Authentication (Recommended):**

```bash
# First, login via Terramate CLI
terramate cloud login

# Run MCP server (auto-loads credentials)
./bin/terramate-mcp-server --region eu

# Or specify custom credential file
./bin/terramate-mcp-server --credential-file /path/to/credentials.tmrc.json --region eu

# Custom base URL (bypasses region)
./bin/terramate-mcp-server --base-url="https://custom.api.example.com"
```

**With API Key:**

```bash
# Using environment variables
export TERRAMATE_API_KEY="your-api-key"
export TERRAMATE_REGION="eu"
./bin/terramate-mcp-server

# Using command-line flags
./bin/terramate-mcp-server --api-key="your-api-key" --region="eu"
```

#### With Docker

> **Apple Silicon:** Add `--platform linux/amd64` to all `docker run` commands below.

**With JWT Authentication:**

```bash
# Mount credential file from host
docker run --rm -it \
  --platform linux/amd64 \
  -v ~/.terramate.d:/root/.terramate.d:ro \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest
```

**With API Key:**

```bash
docker run --rm -it \
  --platform linux/amd64 \
  -e TERRAMATE_API_KEY="your-api-key" \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest
```

### Integrating with AI Assistants

The server communicates via stdio using the Model Context Protocol. Configure your AI assistant to use this server:

#### Claude Desktop

**With JWT Authentication (Recommended):**

**Option 1: Direct Binary**

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "/path/to/bin/terramate-mcp-server",
      "args": ["--region", "eu"]
    }
  }
}
```

**Option 2: With Pre-Start Refresh (Recommended for reliability)**

Ensures token is fresh before server starts:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "bash",
      "args": [
        "-c",
        "terramate cloud info -v >/dev/null 2>&1 || true; /path/to/bin/terramate-mcp-server --region eu"
      ]
    }
  }
}
```

**Note:** Claude Desktop runs in your user context, so it automatically has access to `~/.terramate.d/credentials.tmrc.json`. The MCP server now includes automatic token refresh, so the pre-start command is optional but recommended for extra reliability.

**Option 3: Docker**

Runs the MCP server via Docker instead of a local binary. On Apple Silicon Macs, the `--platform linux/amd64` flag is required:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "--platform", "linux/amd64",
        "-v", "~/.terramate.d:/root/.terramate.d:ro",
        "-e", "TERRAMATE_REGION=eu",
        "ghcr.io/terramate-io/terramate-mcp-server:latest"
      ]
    }
  }
}
```

> **Note:** On Intel-based Macs and Linux x86_64 hosts the `--platform linux/amd64` flag is optional but harmless to include.

**With API Key:**

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

**With JWT Authentication (Recommended):**

**Option 1: Direct Binary**

Add to your Cursor MCP settings:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "/path/to/bin/terramate-mcp-server",
      "args": ["--region", "eu"]
    }
  }
}
```

**Option 2: Docker**

Runs the MCP server via Docker. On Apple Silicon Macs, the `--platform linux/amd64` flag is required:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "--platform", "linux/amd64",
        "-v", "~/.terramate.d:/root/.terramate.d:ro",
        "-e", "TERRAMATE_REGION=eu",
        "ghcr.io/terramate-io/terramate-mcp-server:latest"
      ]
    }
  }
}
```

> **Note:** On Intel-based Macs and Linux x86_64 hosts the `--platform linux/amd64` flag is optional but harmless to include.

**Option 3: Docker with Pre-Start Refresh (Optional)**

Ensures token is fresh before server starts:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "bash",
      "args": [
        "-c",
        "terramate cloud info -v >/dev/null 2>&1 || true; docker run -i --rm --platform linux/amd64 -v ~/.terramate.d:/root/.terramate.d:ro -e TERRAMATE_REGION=eu ghcr.io/terramate-io/terramate-mcp-server:latest"
      ]
    }
  }
}
```

**Note:** The MCP server now includes automatic token refresh and file watching. The pre-start `terramate cloud info` command is optional but provides extra reliability by ensuring the token is fresh before the server starts.

**With API Key (Legacy):**

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

The MCP server provides the following tools for interacting with Terramate Cloud:

### Authentication

#### `tmc_authenticate`

Authenticates with Terramate Cloud and retrieves organization membership information.

**Parameters:** None (uses configured API key)

**Returns:** Organization membership details including UUIDs needed for other tools

**Example:**

```
User: "Show me my Terramate organizations"
Assistant: *calls tmc_authenticate*
Result: List of organizations with UUIDs and roles
```

---

### Stack Management

#### `tmc_list_stacks`

Lists stacks in an organization with powerful filtering and pagination.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID from `tmc_authenticate`

**Optional Filters:**

- `repository` (array) - Filter by repository URLs
- `target` (array) - Filter by target environment
- `status` (array) - Filter by status (ok, failed, drifted, etc.)
- `deployment_status` (array) - Filter by deployment status
- `drift_status` (array) - Filter by drift status (ok, drifted, failed)
- `draft` (boolean) - Filter by draft status
- `is_archived` (array) - Filter by archived status
- `search` (string) - Substring search on name, ID, description, path
- `meta_id` (string) - Filter by exact meta ID
- `meta_tag` (array) - Filter by tags
- `deployment_uuid` (string) - Filter by deployment UUID
- `policy_severity` (array) - Filter by policy severity
- `page` (number) - Page number (default: 1)
- `per_page` (number) - Items per page (max: 100)
- `sort` (array) - Sort fields

**Example:**

```
User: "Show me all drifted stacks in production"
Assistant: *calls tmc_list_stacks with drift_status=["drifted"], target=["production"]*
```

#### `tmc_get_stack`

Retrieves detailed information about a specific stack.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID
- `stack_id` (number) - Stack ID

**Returns:** Complete stack details including related stacks and resource information

**Example:**

```
User: "Get details for stack ID 123"
Assistant: *calls tmc_get_stack*
Result: Full stack metadata, related stacks, resource counts, policy checks
```

---

### Drift Management

#### `tmc_list_drifts`

Lists all drift detection runs for a specific stack.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID
- `stack_id` (number) - Stack ID

**Optional Filters:**

- `drift_status` (array) - Filter by status (ok, drifted, failed)
- `grouping_key` (string) - Filter by CI/CD grouping key
- `page` (number) - Page number (default: 1)
- `per_page` (number) - Items per page (max: 100)

**Returns:** Array of drift runs with metadata (does NOT include terraform plan details)

**Example:**

```
User: "Show me drift detection runs for stack 456"
Assistant: *calls tmc_list_drifts*
Result: List of drift runs with IDs, statuses, and timestamps
```

#### `tmc_get_drift`

Retrieves complete drift details including the Terraform plan output.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID
- `stack_id` (number) - Stack ID
- `drift_id` (number) - Drift ID from `tmc_list_drifts`

**Returns:** Full drift object including:

- `drift_details.changeset_ascii` - Terraform plan in ASCII format (up to 4MB)
- `drift_details.changeset_json` - Terraform plan in JSON format (up to 16MB)
- `drift_details.provisioner` - Tool used (terraform/opentofu)
- `drift_details.serial` - Terraform state serial number
- `stack` - Complete stack object
- Metadata, timestamps, and authentication info

**Example:**

```
User: "Show me the terraform plan for drift ID 100 in stack 456"
Assistant: *calls tmc_get_drift*
Result: Full terraform plan output ready for AI analysis
```

---

### Review Request (Pull/Merge Request) Management

#### `tmc_list_review_requests`

Lists pull requests and merge requests tracked in Terramate Cloud.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID

**Optional Filters:**

- `status` (array) - Filter by PR status (open, merged, closed, approved, changes_requested, review_required)
- `repository` (array) - Filter by repository URLs
- `search` (string) - Search PR number, title, commit SHA, branch names
- `draft` (boolean) - Filter by draft status
- `page` (number) - Page number (default: 1)
- `per_page` (number) - Items per page (max: 100)

**Returns:** Array of review requests with preview summaries (counts only, not plans)

**Example:**

```
User: "Show me all open PRs with terraform plan changes"
Assistant: *calls tmc_list_review_requests with status=["open"]*
Result: List of PRs with preview.changed_count > 0
```

#### `tmc_get_review_request`

Retrieves complete PR details including terraform plans for ALL affected stacks.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID
- `review_request_id` (number) - Review Request ID from list

**Optional Parameters:**

- `exclude_stack_previews` (boolean) - Exclude terraform plans (default: false)

**Returns:** Full PR details including:

- `review_request` - PR metadata (title, branch, status, checks, reviews)
- `stack_previews[]` - Array of per-stack terraform plans with:
  - `stack` - Full stack object (stack_id, path, meta_id)
  - `changeset_details.changeset_ascii` - Terraform plan (up to 4MB)
  - `resource_changes` - Summary of creates/updates/deletes
  - `status` - changed, unchanged, failed, etc.

**Example:**

```
User: "Show me terraform plans for PR #245"
Assistant: *finds review_request_id, calls tmc_get_review_request*
Result: All stack plans with full terraform output for AI analysis
```

---

### Deployment Management

#### `tmc_list_deployments`

Lists CI/CD workflow deployments in an organization.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID

**Optional Filters:**

- `repository` (array) - Filter by repository URLs
- `status` (array) - Filter by status (ok, failed, processing)
- `search` (string) - Search commit SHA, title, or branch
- `page` (number) - Page number (default: 1)
- `per_page` (number) - Items per page (max: 100)

**Returns:** Array of workflow deployments with:

- Status counts (ok_count, failed_count, pending_count, running_count, canceled_count)
- Commit information
- Timestamps
- Optional review_request (if deployed from a PR)

**Example:**

```
User: "Show me recent failed deployments"
Assistant: *calls tmc_list_deployments with status=["failed"]*
Result: List of failed CI/CD runs with stack counts
```

#### `tmc_get_stack_deployment`

Retrieves detailed deployment information including terraform apply output.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID
- `stack_deployment_id` (number) - Stack Deployment ID

**Returns:** Complete deployment details including:

- `changeset_details.changeset_ascii` - Terraform apply plan (up to 4MB)
- `stack` - Full stack object
- `cmd` - Command executed
- `status` - Deployment status
- Timestamps (created_at, started_at, finished_at)

**Example:**

```
User: "Show me what was deployed for stack deployment 200"
Assistant: *calls tmc_get_stack_deployment*
Result: Full terraform apply output and deployment details
```

---

### Stack Resources

#### `tmc_list_resources`

Lists resources (stack-level plan/state resources, e.g. Terraform resources) in an organization with optional filtering.

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID from `tmc_authenticate`

**Optional Filters:**

- `stack_id` (number) - Filter by stack ID (list resources for a single stack)
- `status` (array) - Filter by resource status (ok, drifted, pending)
- `technology` (array) - Filter by technology (e.g. terraform, opentofu)
- `provider` (array) - Filter by provider (e.g. aws, gcloud)
- `resource_type` (array) - Filter by resource type (e.g. aws_vpc, loadbalancer)
- `repository` (array) - Filter by repository URLs
- `target` (array) - Filter by deployment target
- `extracted_account` (array) - Filter by extracted account
- `is_archived` (array) - Filter by stack archived status
- `policy_severity` (array) - Filter by policy check (missing, none, passed, low, medium, high)
- `search` (string) - Search in stack title/description/path and resource name/id/address
- `page` (number) - Page number (default: 1)
- `per_page` (number) - Items per page (max: 100)
- `sort` (array) - Sort fields (e.g. updated_at,desc or path,asc)

**Returns:** Array of resources with descriptor (address, type, provider), stack, status, drifted, pending, and pagination info.

**Example:**

```
User: "List all resources for stack 42"
Assistant: *calls tmc_list_resources with stack_id=42*
Result: Resources with address, type, provider, status
```

#### `tmc_get_resource`

Retrieves a specific resource by UUID, including optional details (e.g. values state when available).

**Required Parameters:**

- `organization_uuid` (string) - Organization UUID
- `resource_uuid` (string) - Resource UUID from `tmc_list_resources`

**Returns:** Full resource object including descriptor, stack, status, and optionally `details` (values, sensitive_values).

**Example:**

```
User: "Get details for resource f1c9ecfe-1a45-499b-ab6d-1aa0a8ea2f95"
Assistant: *calls tmc_get_resource*
Result: Resource with descriptor and optional state details
```

---

## Use Cases

### 1. Find and Analyze Drifted Infrastructure

```
User: "Show me all drifted stacks in my production environment"

Assistant workflow:
1. Calls tmc_authenticate to get organization_uuid
2. Calls tmc_list_stacks with:
   - drift_status: ["drifted"]
   - target: ["production"]
3. Displays drifted stacks with IDs and paths

User: "Get the terraform plan for the VPC stack drift"

Assistant workflow:
1. Calls tmc_list_drifts for the VPC stack_id
2. Gets the most recent drift_id
3. Calls tmc_get_drift to retrieve the full plan
4. Presents the changeset_ascii to user
5. Can now help reconcile the drift using AI analysis
```

### 2. Monitor Deployment Status Across Repositories

```
User: "Show me all stacks in github.com/acme/infrastructure with deployment issues"

Assistant workflow:
1. Calls tmc_authenticate
2. Calls tmc_list_stacks with:
   - repository: ["github.com/acme/infrastructure"]
   - deployment_status: ["failed"]
3. Shows problematic stacks with details
```

### 3. Search and Filter Stacks by Tags

```
User: "Find all production database stacks with policy violations"

Assistant workflow:
1. Calls tmc_authenticate
2. Calls tmc_list_stacks with:
   - meta_tag: ["production", "database"]
   - policy_severity: ["high", "medium"]
3. Lists matching stacks with policy check details
```

### 4. Drill Down into Specific Stack Details

```
User: "Get complete details for stack ID 789"

Assistant workflow:
1. Calls tmc_authenticate
2. Calls tmc_get_stack with stack_id: 789
3. Shows:
   - Stack metadata (name, description, tags)
   - Status information
   - Related stacks across targets
   - Resource counts
   - Policy check results
```

### 5. AI-Assisted Drift Reconciliation

```
User: "Help me fix the drift in stack 456"

Assistant workflow:
1. Calls tmc_list_drifts for stack 456
2. Identifies most recent drifted run
3. Calls tmc_get_drift to get terraform plan
4. Analyzes the plan:
   - Identifies changed resources
   - Explains what drifted
   - Suggests remediation steps
5. Can propose terraform code changes to reconcile

Example drift plan analysis:
"The drift shows that the security group description changed from
'Old desc' to 'New desc' outside of Terraform. You have two options:
1. Update your Terraform code to match the current state
2. Apply your Terraform to revert the manual change"
```

### 6. Search Across All Stacks

```
User: "Find any stacks with 'database' in their name or path"

Assistant workflow:
1. Calls tmc_authenticate
2. Calls tmc_list_stacks with:
   - search: "database"
3. Returns all matching stacks (searches name, ID, description, and path)
```

### 7. Paginate Through Large Stack Lists

```
User: "Show me the first 10 stacks, then the next 10"

Assistant workflow:
1. Calls tmc_list_stacks with page: 1, per_page: 10
2. User asks for more
3. Calls tmc_list_stacks with page: 2, per_page: 10
4. Uses paginated_result to show "Page 2 of 15"
```

### 8. Complete Drift Investigation Workflow

```
User: "I need to understand what's drifted in my infrastructure"

Assistant workflow:
1. Authenticate and get org UUID
2. List all drifted stacks:
   tmc_list_stacks(drift_status: ["drifted"])

3. For each drifted stack:
   a. Get drift run history: tmc_list_drifts(stack_id)
   b. Get latest drift details: tmc_get_drift(drift_id)
   c. Analyze the terraform plan

4. Provide summary:
   - Total drifted stacks: 5
   - Most common changes: security group modifications
   - Recommended actions for each drift

5. Assist with remediation planning
```

### 9. Review Pull Request Terraform Plans

```
User: "Show me what infrastructure changes are in PR #245"

Assistant workflow:
1. Search for the PR:
   tmc_list_review_requests(repository: ["github.com/acme/infra"], search: "245")

2. Get PR details with stack previews:
   tmc_get_review_request(review_request_id: 42)

3. Analyze each stack preview:
   - Stack: /stacks/vpc (changed)
     * Creates: 0, Updates: 1, Deletes: 0
     * Plan: VPC CIDR changing from 10.0.0.0/16 to 10.1.0.0/16

   - Stack: /stacks/database (unchanged)
     * No changes

4. Provide assessment:
   "This PR updates the VPC CIDR block. This is a destructive change
    that will require downtime. Recommend reviewing with team lead."
```

### 10. Find Terraform Plan for Specific Stack in a PR

```
User: "What changes will PR #123 make to the VPC stack?"

Assistant workflow:
1. Find the PR:
   tmc_list_review_requests(search: "123")
   → review_request_id: 42

2. Get PR with stack previews:
   tmc_get_review_request(review_request_id: 42)

3. Find VPC stack in stack_previews:
   for preview in stack_previews:
     if preview.stack.path == "/stacks/vpc":
       terraform_plan = preview.changeset_details.changeset_ascii

4. Display plan:
   "The VPC stack will have these changes:
   - Security group description updated
   - No resources created or destroyed"
```

### 11. Review All Failed Terraform Plans in Open PRs

```
User: "Show me all PRs with failed terraform plans"

Assistant workflow:
1. List open PRs:
   tmc_list_review_requests(status: ["open"])

2. Filter PRs with failures using preview.failed_count:
   failed_prs = [pr for pr in review_requests if pr.preview.failed_count > 0]

3. For each failed PR, get details:
   tmc_get_review_request(review_request_id)

4. Find failed stacks:
   for stack_preview in stack_previews:
     if stack_preview.status == "failed":
       - Analyze the error in changeset_ascii
       - Suggest fixes

5. Provide summary:
   "PR #245: VPC stack failed - missing required variable
    PR #246: Database stack failed - syntax error in main.tf"
```

### 12. AI-Assisted PR Review Workflow

```
User: "Help me review the infrastructure changes in PR #200"

Assistant workflow:
1. Get PR with all stack plans:
   tmc_get_review_request(review_request_id: 200)

2. Analyze the PR:
   - Title: "feat: Add production database"
   - Branch: feature/prod-db
   - Status: open, awaiting review
   - Checks: 5/5 passing
   - Preview: 3 stacks changed, 0 failed

3. Review each stack plan:
   Stack 1: /stacks/database
   - Creates: 1 RDS instance
   - Security group allows 0.0.0.0/0 ⚠️ SECURITY RISK

   Stack 2: /stacks/vpc
   - Updates: Security group rules
   - Looks good ✓

4. Provide review feedback:
   "⚠️ Security concern: Database security group allows public access.
    Recommend restricting to VPC CIDR only.

    Suggested fix:
    - Change ingress_cidr_blocks from ['0.0.0.0/0'] to ['10.0.0.0/16']"
```

### 13. Find Which PRs Affect a Specific Stack

```
User: "Which open PRs will change the production VPC stack?"

Assistant workflow:
1. List open PRs in the repository:
   tmc_list_review_requests(
     repository: ["github.com/acme/infra"],
     status: ["open"]
   )

2. For each PR, check if it affects the VPC stack:
   for pr in review_requests:
     details = tmc_get_review_request(review_request_id: pr.review_request_id)
     for stack_preview in details.stack_previews:
       if stack_preview.stack.path == "/stacks/vpc" and stack_preview.status != "unchanged":
         # Found a PR affecting VPC!

3. Report findings:
   "2 PRs will affect the VPC stack:
   - PR #245: Updates CIDR block (2 resources changed)
   - PR #250: Adds NAT gateway (3 resources created)"
```

### 14. Monitor CI/CD Deployment Activity

```
User: "Show me recent deployment activity in my infrastructure repo"

Assistant workflow:
1. List recent deployments:
   tmc_list_deployments(
     repository: ["github.com/acme/infrastructure"]
   )

2. Display summary:
   "Recent deployments:
   - Deployment #100 (2 hours ago): ✅ 5/5 stacks succeeded
   - Deployment #99 (5 hours ago): ❌ 3/5 stacks failed
   - Deployment #98 (1 day ago): ✅ 8/8 stacks succeeded"

3. User can drill down on failures
```

### 15. Debug Failed Deployment

```
User: "Why did deployment #99 fail?"

Assistant workflow:
1. Get workflow details:
   (Using SDK: client.Deployments.GetWorkflow(ctx, orgUUID, 99))
   Result: Shows 3 stacks failed out of 5

2. List stack deployments in workflow:
   (Using SDK: client.Deployments.ListForWorkflow(ctx, orgUUID, 99, nil))

3. For each failed stack deployment, get details:
   tmc_get_stack_deployment(stack_deployment_id: 200)

4. Analyze the terraform output:
   "Stack /stacks/database failed during apply:
   Error: Resource 'aws_db_instance.main' failed to create
   - Insufficient instance capacity in availability zone

   Recommendation: Change instance type or try different AZ"
```

### 16. Track Deployment History for a Stack

```
User: "Show me deployment history for the VPC stack"

Assistant workflow:
1. Find the stack:
   tmc_list_stacks(search: "vpc")
   → stack_id: 456

2. Get deployment history:
   (Using SDK: client.Deployments.ListStackDeployments with stack filter)
   Or via Stacks service if that endpoint is available

3. Display timeline:
   "VPC Stack Deployment History:
   - Jan 15, 10:00: ✅ Deployed successfully (terraform apply)
   - Jan 14, 15:30: ✅ Deployed successfully (terraform apply)
   - Jan 13, 09:15: ❌ Failed - timeout waiting for VPC
   - Jan 12, 14:20: ✅ Deployed successfully (terraform apply)"
```

### 17. Compare Deployment vs Drift

```
User: "What's the difference between the last deployment and current drift?"

Assistant workflow:
1. Get stack:
   tmc_get_stack(stack_id: 456)

2. Get latest deployment:
   (Using SDK: Get deployment history, take most recent)
   tmc_get_stack_deployment(stack_deployment_id: 200)
   Deployment plan: "Deploys VPC with CIDR 10.0.0.0/16"

3. Get latest drift:
   tmc_list_drifts(stack_id: 456)
   tmc_get_drift(drift_id: 100)
   Drift plan: "VPC CIDR changed to 10.1.0.0/16"

4. Compare:
   "The deployment created VPC with 10.0.0.0/16.
    Current drift shows someone manually changed it to 10.1.0.0/16.
    This is an unauthorized change that should be reviewed."
```

### 18. Debug Failed Terraform Plan in Pull Request

````
User: "The database stack terraform plan failed in PR #245. Help me fix it."

AI-Assisted Debugging Workflow:
1. Find the PR:
   tmc_list_review_requests(search: "245")
   → review_request_id: 42

2. Get PR with stack previews:
   tmc_get_review_request(review_request_id: 42)
   → Find stack_preview where status == "failed"
   → stack_preview_id: 300, stack.path: "/stacks/database"

3. Fetch raw terraform error logs:
   tmc_get_stack_preview_logs(
     organization_uuid: "org-123",
     stack_preview_id: 300,
     channel: "stderr"  ← Error messages only
   )

4. AI analyzes the logs:
   Log output shows:
   "Error: error configuring Terraform AWS Provider: no valid credential sources
    Error: Failed to query available provider packages"

5. AI provides diagnosis and fix:
   "The terraform plan failed because AWS credentials are not configured in your
    GitHub Actions workflow.

    Root Cause:
    - Missing AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables
    - The AWS provider cannot authenticate

    To Fix - Add to your workflow (.github/workflows/terraform.yml):
    ```yaml
    - name: Configure AWS Credentials
      uses: aws-actions/configure-aws-credentials@v4
      with:
        role-to-assume: arn:aws:iam::123456789:role/github-actions
        aws-region: us-east-1
    ```

    Or add secrets:
    ```yaml
    env:
      AWS_ACCESS_KEY_ID: \${{ secrets.AWS_ACCESS_KEY_ID }}
      AWS_SECRET_ACCESS_KEY: \${{ secrets.AWS_SECRET_ACCESS_KEY }}
    ```"
````

### 19. Debug Failed Deployment in CI/CD

```
User: "Deployment #99 failed. What went wrong?"

AI-Assisted Deployment Debugging:
1. Get workflow details:
   (Using SDK: client.Deployments.GetWorkflow(ctx, orgUUID, 99))
   Result: Shows 3 stacks failed: /stacks/database, /stacks/cache, /stacks/queue

2. Get failed stack deployments:
   (Using SDK: client.Deployments.ListForWorkflow(ctx, orgUUID, 99, nil))
   Filter for status == "failed"

3. For the database stack failure, get deployment logs:
   tmc_get_deployment_logs(
     organization_uuid: "org-123",
     stack_id: 789,
     deployment_uuid: "deploy-uuid-abc",
     channel: "stderr"
   )

4. AI analyzes terraform apply errors:
   Logs show:
   "Error: creating RDS DB Instance: InvalidParameterValue:
    The parameter MasterUserPassword is not a valid password"

5. AI provides fix:
   "The deployment failed because the RDS master password doesn't meet
    AWS requirements.

    Issue:
    - Password must be 8-41 characters
    - Must contain uppercase, lowercase, numbers, and special characters
    - Cannot contain certain special characters: @, \", '

    Fix:
    1. Update your password in the secrets manager or tfvars
    2. Ensure it meets AWS RDS password requirements
    3. Common issue: passwords with @ or quotes need to be escaped

    Example valid password: MyP@ssw0rd123!
    Re-run the deployment after updating the password."
```

## Development

### Building

```bash
# Build production binary
make build

# Build debug binary (faster, includes debug symbols)
make build/dev

# Build Docker image
make docker/build
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test/coverage

# Run specific package tests
go test -v ./sdk/terramate/...
```

### Linting

```bash
# Run linters
make lint

# Auto-fix linting issues
make lint/fix

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
│       ├── stacks.go            # Stacks API
│       ├── drifts.go            # Drifts API
│       ├── reviewrequests.go    # Review Requests (PR/MR) API
│       ├── deployments.go       # Deployments API
│       ├── previews.go          # Previews API
│       ├── resources.go         # Stack resources API
│       └── types.go             # API data models
├── tools/
│   ├── handlers.go              # Tool registration
│   └── tmc/                     # Terramate Cloud MCP tools
│       ├── auth.go              # Authentication tool
│       ├── stacks.go            # Stack management tools
│       ├── drifts.go            # Drift detection tools
│       ├── reviewrequests.go    # Pull/merge request tools
│       ├── deployments.go       # Deployment tracking tools
│       ├── previews.go          # Stack preview logs tool
│       └── resources.go         # Stack resources tools
├── internal/
│   └── version/                 # Version and user agent
└── Makefile                     # Build automation
```

## Architecture

### Graceful Shutdown

The MCP server handles `SIGINT` and `SIGTERM` signals gracefully:

1. Stops accepting new requests
2. Waits up to 30 seconds for in-flight requests to complete
3. Logs shutdown status

## SDK Documentation

For programmatic access to the Terramate Cloud API, see the [SDK documentation](sdk/terramate/README.md).

The SDK provides type-safe Go clients for all Terramate Cloud APIs:

- **Stacks** - Manage infrastructure stacks
- **Drifts** - Detect and analyze drift
- **Deployments** - Monitor CI/CD deployments with logs
- **Review Requests** - Integrate with PRs/MRs
- **Previews** - Debug failed terraform plans with logs
- **Resources** - List and get stack resources (plan/state)
- **Memberships** - Organization management

```go
import "github.com/terramate-io/terramate-mcp-server/sdk/terramate"

// With JWT (recommended)
credPath, _ := terramate.GetDefaultCredentialPath()
credential, _ := terramate.LoadJWTFromFile(credPath)
client, _ := terramate.NewClient(credential, terramate.WithRegion("eu"))

// Or with API key
client, _ := terramate.NewClientWithAPIKey("your-api-key", terramate.WithRegion("eu"))

stacks, _, _ := client.Stacks.List(ctx, orgUUID, nil)
```

See [sdk/terramate/README.md](sdk/terramate/README.md) for complete documentation, examples, and API reference.

## Troubleshooting

### Docker on Apple Silicon (M1/M2/M3/M4)

**Problem:** The Docker container crashes on startup or prints `exec format error`

**Solution:**

- The published Docker image is built for `linux/amd64`. On Apple Silicon Macs you need to explicitly request that platform:
  ```bash
  docker run --rm -it --platform linux/amd64 \
    -v ~/.terramate.d:/root/.terramate.d:ro \
    -e TERRAMATE_REGION="eu" \
    ghcr.io/terramate-io/terramate-mcp-server:latest
  ```
- Make sure Docker Desktop has Rosetta emulation enabled: **Docker Desktop → Settings → General → Use Rosetta for x86_64/amd64 emulation on Apple Silicon** (recommended for better performance)
- Alternatively, build the image locally on your Mac to get a native `arm64` image:
  ```bash
  docker build -t terramate-mcp-server .
  ```

### Authentication Failures

**Problem:** `Authentication failed: credentials are invalid or expired`

**Solution:**

- If using JWT credentials:
  - Run `terramate cloud login` again to refresh credentials
  - Ensure `~/.terramate.d/credentials.tmrc.json` contains both `id_token` and `refresh_token`
  - If using custom identity provider setup, set `TMC_API_IDP_KEY` to match your Terramate CLI/IDP configuration
- If using API key authentication:
  - Verify your API key is correct
  - Ensure the API key has not expired
  - Regenerate the API key if necessary
- Check that you're using the correct region

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

## CI/CD & Releases

### Continuous Integration

The project uses GitHub Actions for CI/CD. On every push and pull request to `main`, the following checks run:

- **Tests** - Unit tests with race detector and coverage reporting
- **Linting** - Code quality checks using golangci-lint
- **Formatting** - Ensures all code is properly formatted
- **Build** - Verifies the binary builds successfully
- **Docker** - Tests Docker image builds

### Release Process

When a new release is published on GitHub:

1. A Docker image is automatically built with version information
2. The image is tagged with multiple tags:
   - `ghcr.io/terramate-io/terramate-mcp-server:latest`
   - `ghcr.io/terramate-io/terramate-mcp-server:1.2.3` (exact version)
   - `ghcr.io/terramate-io/terramate-mcp-server:1.2` (minor version)
   - `ghcr.io/terramate-io/terramate-mcp-server:1` (major version)
3. The image is pushed to GitHub Container Registry

### Creating a Release

To create a new release:

```bash
# Tag the release (use semantic versioning)
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3

# Or create a release via GitHub UI
# The release workflow will automatically build and push the Docker image
```

### Using Released Docker Images

```bash
# Pull the latest version
docker pull ghcr.io/terramate-io/terramate-mcp-server:latest

# Pull a specific version
docker pull ghcr.io/terramate-io/terramate-mcp-server:1.2.3

# Run with JWT authentication (recommended)
# First: terramate cloud login
# Note: Add --platform linux/amd64 on Apple Silicon Macs
docker run --rm -it \
  --platform linux/amd64 \
  -v ~/.terramate.d:/root/.terramate.d:ro \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest

# Or run with API key (deprecated)
docker run --rm -it \
  --platform linux/amd64 \
  -e TERRAMATE_API_KEY="your-api-key" \
  -e TERRAMATE_REGION="eu" \
  ghcr.io/terramate-io/terramate-mcp-server:latest
```

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
