# Terramate Cloud SDK for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/terramate-io/terramate-mcp-server)](https://goreportcard.com/report/github.com/terramate-io/terramate-mcp-server)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A production-ready Go SDK for interacting with the [Terramate Cloud API](https://cloud.terramate.io). This SDK provides type-safe access to stacks, drifts, deployments, and pull request integrations.

## Features

- 🔐 **Secure Authentication** - API key-based authentication with automatic retry logic
- 🌍 **Multi-Region Support** - EU and US region endpoints
- 📦 **Complete API Coverage** - Stacks, Drifts, Deployments, Review Requests, and Memberships
- 🔄 **Automatic Retries** - Built-in exponential backoff for transient failures
- ⏱️ **Context Support** - Cancellation and timeout handling
- 🧪 **Well Tested** - 82%+ test coverage with 140+ tests
- 📝 **Type Safe** - Full Go type definitions for all API resources

## Installation

```bash
go get github.com/terramate-io/terramate-mcp-server/sdk/terramate
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

func main() {
    // Create client
    client, err := terramate.NewClient("your-api-key",
        terramate.WithRegion("eu"))
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    
    // List organizations
    memberships, _, err := client.Memberships.List(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    orgUUID := memberships[0].OrgUUID
    
    // List drifted stacks
    stacks, _, err := client.Stacks.List(ctx, orgUUID, &terramate.StacksListOptions{
        DriftStatus: []string{"drifted"},
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d drifted stacks\n", len(stacks.Stacks))
}
```

## Client Configuration

### Creating a Client

```go
import "github.com/terramate-io/terramate-mcp-server/sdk/terramate"

// Basic client (EU region by default)
client, err := terramate.NewClient("your-api-key")

// With region
client, err := terramate.NewClient(
    "your-api-key",
    terramate.WithRegion("us"),  // or "eu"
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

// With custom HTTP client
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: customTransport,
}
client, err := terramate.NewClient(
    "your-api-key",
    terramate.WithHTTPClient(httpClient),
)
```

### Region Endpoints

- **EU**: `https://api.terramate.io` (default)
- **US**: `https://api.us.terramate.io`

## API Services

### Memberships API

List organization memberships for the authenticated user.

```go
// List all organizations
memberships, resp, err := client.Memberships.List(ctx)
if err != nil {
    log.Fatal(err)
}

for _, m := range memberships {
    fmt.Printf("Org: %s (%s), Role: %s, Status: %s\n",
        m.OrgDisplayName, m.OrgUUID, m.Role, m.Status)
}
```

### Stacks API

Manage and query infrastructure stacks.

```go
// List all stacks in an organization
stacks, _, err := client.Stacks.List(ctx, orgUUID, nil)

// List with filters
stacks, _, err := client.Stacks.List(ctx, orgUUID, &terramate.StacksListOptions{
    Repository:       []string{"github.com/acme/infra"},
    Target:           []string{"production"},
    DriftStatus:      []string{"drifted"},
    DeploymentStatus: []string{"failed"},
    Search:           "database",
    MetaTag:          []string{"critical", "production"},
    Page:             1,
    PerPage:          50,
})

// Get specific stack details
stack, _, err := client.Stacks.Get(ctx, orgUUID, stackID)
fmt.Printf("Stack: %s\n", stack.MetaName)
fmt.Printf("Status: %s, Drift: %s\n", stack.Status, stack.DriftStatus)
```

### Drifts API

Detect and analyze infrastructure drift.

```go
// List drift detection runs for a stack
drifts, _, err := client.Drifts.ListForStack(ctx, orgUUID, stackID,
    &terramate.DriftsListOptions{
        DriftStatus: []string{"drifted", "failed"},
        Page:        1,
        PerPage:     10,
    })

// Get drift details with terraform plan
drift, _, err := client.Drifts.Get(ctx, orgUUID, stackID, driftID)

// Access the terraform plan
if drift.DriftDetails != nil {
    fmt.Printf("Provisioner: %s\n", drift.DriftDetails.Provisioner)
    fmt.Printf("State Serial: %d\n", drift.DriftDetails.Serial)
    fmt.Println(drift.DriftDetails.ChangesetASCII)  // Full terraform plan
}
```

### Review Requests API

Work with pull requests and merge requests.

```go
// List open PRs
reviewRequests, _, err := client.ReviewRequests.List(ctx, orgUUID,
    &terramate.ReviewRequestsListOptions{
        Repository: []string{"github.com/acme/infra"},
        Status:     []string{"open"},
        Search:     "database",
    })

// Get PR with stack previews (terraform plans)
details, _, err := client.ReviewRequests.Get(ctx, orgUUID, reviewRequestID, nil)

// Access terraform plans for each affected stack
for _, sp := range details.StackPreviews {
    fmt.Printf("Stack: %s\n", sp.Stack.Path)
    fmt.Printf("Status: %s\n", sp.Status)
    
    if sp.ChangesetDetails != nil {
        fmt.Println(sp.ChangesetDetails.ChangesetASCII)  // Terraform plan
    }
    
    // See change counts
    if sp.ResourceChanges != nil {
        fmt.Printf("Creates: %d, Updates: %d, Deletes: %d\n",
            sp.ResourceChanges.ActionsSummary.CreateCount,
            sp.ResourceChanges.ActionsSummary.UpdateCount,
            sp.ResourceChanges.ActionsSummary.DeleteCount)
    }
}

// Exclude stack previews for faster response
details, _, err := client.ReviewRequests.Get(ctx, orgUUID, reviewRequestID,
    &terramate.ReviewRequestGetOptions{
        ExcludeStackPreviews: true,
    })
```

### Deployments API

Monitor and analyze CI/CD deployments.

```go
// List recent deployments
deployments, _, err := client.Deployments.List(ctx, orgUUID,
    &terramate.DeploymentsListOptions{
        Repository: []string{"github.com/acme/infra"},
        Status:     []string{"failed"},
        Page:       1,
        PerPage:    20,
    })

for _, d := range deployments.Deployments {
    fmt.Printf("Deployment #%d: %s\n", d.ID, d.CommitTitle)
    fmt.Printf("Status: %s (%d ok, %d failed)\n", 
        d.Status, d.OkCount, d.FailedCount)
}

// Get workflow deployment details
workflow, _, err := client.Deployments.GetWorkflow(ctx, orgUUID, workflowID)
fmt.Printf("Workflow: %s\n", workflow.CommitTitle)
fmt.Printf("Stacks: %d total (%d ok, %d failed)\n",
    workflow.StackDeploymentTotalCount,
    workflow.OkCount,
    workflow.FailedCount)

// List stack deployments in a workflow
stackDeployments, _, err := client.Deployments.ListForWorkflow(ctx, orgUUID, workflowID, nil)

// Get specific stack deployment with terraform apply output
deployment, _, err := client.Deployments.GetStackDeployment(ctx, orgUUID, stackDeploymentID)

if deployment.ChangesetDetails != nil {
    fmt.Println(deployment.ChangesetDetails.ChangesetASCII)  // Terraform apply output
}

// List all stack deployments across organization
allDeployments, _, err := client.Deployments.ListStackDeployments(ctx, orgUUID,
    &terramate.StackDeploymentsListOptions{
        Status: []string{"failed"},
    })
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
stacks, _, err := client.Stacks.List(ctx, orgUUID, nil)
if err != nil {
    if apiErr, ok := err.(*terramate.APIError); ok {
        switch {
        case apiErr.IsUnauthorized():
            // Handle 401 - check API key
            fmt.Println("Authentication failed")
        case apiErr.IsNotFound():
            // Handle 404
            fmt.Println("Resource not found")
        case apiErr.IsForbidden():
            // Handle 403
            fmt.Println("Access denied")
        case apiErr.IsBadRequest():
            // Handle 400
            fmt.Println("Invalid request:", apiErr.Message)
        case apiErr.IsServerError():
            // Handle 5xx
            fmt.Println("Server error, will retry")
        default:
            fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
        }
        return
    }
    log.Fatal(err)
}
```

## Context and Timeouts

The SDK respects context cancellation and timeouts:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

stacks, _, err := client.Stacks.List(ctx, orgUUID, nil)

// With cancellation
ctx, cancel := context.WithCancel(context.Background())

go func() {
    // Cancel after some condition
    cancel()
}()

drift, _, err := client.Drifts.Get(ctx, orgUUID, stackID, driftID)
```

## Pagination

All list methods support pagination:

```go
opts := &terramate.StacksListOptions{
    ListOptions: terramate.ListOptions{
        Page:    1,
        PerPage: 50,
    },
    DriftStatus: []string{"drifted"},
}

result, _, err := client.Stacks.List(ctx, orgUUID, opts)

// Check pagination
fmt.Printf("Page %d of %d\n", result.PaginatedResult.Page, result.PaginatedResult.TotalPages())
fmt.Printf("Total: %d\n", result.PaginatedResult.Total)

if result.PaginatedResult.HasNextPage() {
    opts.Page++
    // Fetch next page...
}
```

## Advanced Usage Examples

### Find and Analyze All Drifted Infrastructure

```go
// 1. List drifted stacks
stacks, _, _ := client.Stacks.List(ctx, orgUUID, &terramate.StacksListOptions{
    DriftStatus: []string{"drifted"},
})

// 2. For each drifted stack, get the drift details
for _, stack := range stacks.Stacks {
    drifts, _, _ := client.Drifts.ListForStack(ctx, orgUUID, stack.StackID, 
        &terramate.DriftsListOptions{
            DriftStatus: []string{"drifted"},
            PerPage:     1,  // Just the latest
        })
    
    if len(drifts.Drifts) > 0 {
        drift, _, _ := client.Drifts.Get(ctx, orgUUID, stack.StackID, drifts.Drifts[0].ID)
        
        fmt.Printf("Stack: %s\n", stack.MetaName)
        fmt.Printf("Drift Plan:\n%s\n", drift.DriftDetails.ChangesetASCII)
    }
}
```

### Review Terraform Plans in a Pull Request

```go
// 1. Find PR by number
reviews, _, _ := client.ReviewRequests.List(ctx, orgUUID,
    &terramate.ReviewRequestsListOptions{
        Search: "245",  // PR number
    })

// 2. Get PR with all stack terraform plans
details, _, _ := client.ReviewRequests.Get(ctx, orgUUID, 
    reviews.ReviewRequests[0].ReviewRequestID, nil)

// 3. Analyze each stack's changes
for _, sp := range details.StackPreviews {
    if sp.Status == "changed" {
        fmt.Printf("\nStack: %s\n", sp.Stack.Path)
        fmt.Printf("Changes: +%d ~%d -%d\n",
            sp.ResourceChanges.ActionsSummary.CreateCount,
            sp.ResourceChanges.ActionsSummary.UpdateCount,
            sp.ResourceChanges.ActionsSummary.DeleteCount)
        fmt.Printf("Plan:\n%s\n", sp.ChangesetDetails.ChangesetASCII)
    }
}
```

### Monitor Failed Deployments

```go
// 1. List recent failed deployments
deployments, _, _ := client.Deployments.List(ctx, orgUUID,
    &terramate.DeploymentsListOptions{
        Status:  []string{"failed"},
        PerPage: 10,
    })

// 2. For each failed deployment, get details
for _, d := range deployments.Deployments {
    fmt.Printf("\nDeployment #%d: %s\n", d.ID, d.CommitTitle)
    fmt.Printf("Failed stacks: %d/%d\n", d.FailedCount, d.StackDeploymentTotalCount)
    
    // 3. Get individual stack failures
    stackDeployments, _, _ := client.Deployments.ListForWorkflow(ctx, orgUUID, d.ID, nil)
    
    for _, sd := range stackDeployments.StackDeployments {
        if sd.Status == "failed" {
            deployment, _, _ := client.Deployments.GetStackDeployment(ctx, orgUUID, sd.ID)
            fmt.Printf("  Failed: %s\n", sd.Path)
            // Analyze failure in deployment.ChangesetDetails
        }
    }
}
```

### Debug Failed Terraform Plan in Pull Request

```go
// Complete debugging workflow for failed PR terraform plans

// 1. Find PR with failures
reviews, _, _ := client.ReviewRequests.List(ctx, orgUUID,
    &terramate.ReviewRequestsListOptions{
        Search: "245",  // PR number
    })

// 2. Get PR with stack previews
details, _, _ := client.ReviewRequests.Get(ctx, orgUUID, reviews.ReviewRequests[0].ReviewRequestID, nil)

// 3. Find failed stack previews
for _, sp := range details.StackPreviews {
    if sp.Status == "failed" {
        fmt.Printf("\n❌ Failed: %s (Preview ID: %d)\n", sp.Stack.Path, sp.StackPreviewID)
        
        // 4. Get error logs for AI analysis
        logs, _, _ := client.Previews.GetLogs(ctx, orgUUID, sp.StackPreviewID,
            &terramate.PreviewLogsOptions{
                Channel: "stderr",  // Error messages
                PerPage: 100,
            })
        
        // 5. Display logs for AI to analyze
        fmt.Println("Error logs:")
        for _, log := range logs.StackPreviewLogLines {
            fmt.Printf("[%s] %s\n", log.Timestamp.Format("15:04:05"), log.Message)
        }
        
        // AI can now analyze these logs and suggest fixes
        // Example errors:
        // - Provider authentication issues
        // - Missing required variables
        // - Syntax errors in terraform code
        // - Resource creation failures
    }
}
```

### Debug Failed Deployment with Logs

```go
// Complete debugging workflow for failed deployments

// 1. List recent failed deployments
deployments, _, _ := client.Deployments.List(ctx, orgUUID,
    &terramate.DeploymentsListOptions{
        Status: []string{"failed"},
    })

// 2. Get workflow details
workflow, _, _ := client.Deployments.GetWorkflow(ctx, orgUUID, deployments.Deployments[0].ID)
fmt.Printf("Failed deployment: %s\n", workflow.CommitTitle)
fmt.Printf("Failed stacks: %d\n", workflow.FailedCount)

// 3. Get failed stack deployments
stackDeps, _, _ := client.Deployments.ListForWorkflow(ctx, orgUUID, workflow.ID, nil)

for _, sd := range stackDeps.StackDeployments {
    if sd.Status == "failed" {
        // 4. Get deployment logs for AI analysis
        logs, _, _ := client.Deployments.GetDeploymentLogs(ctx, orgUUID, 
            sd.Stack.StackID, sd.DeploymentUUID,
            &terramate.DeploymentLogsOptions{
                Channel: "stderr",
                PerPage: 100,
            })
        
        fmt.Printf("\n❌ Failed deployment: %s\n", sd.Path)
        fmt.Println("Error logs:")
        for _, log := range logs.DeploymentLogLines {
            fmt.Printf("[%s] %s\n", log.Timestamp.Format("15:04:05"), log.Message)
        }
        
        // AI analyzes logs and provides:
        // - Root cause identification
        // - Fix suggestions
        // - Configuration recommendations
        // - Links to relevant documentation
    }
}
```

## API Reference

### Services

- **`client.Memberships`** - Organization memberships
  - `List(ctx)` - List user's organizations

- **`client.Stacks`** - Infrastructure stacks
  - `List(ctx, orgUUID, opts)` - List/filter stacks
  - `Get(ctx, orgUUID, stackID)` - Get stack details

- **`client.Drifts`** - Drift detection
  - `ListForStack(ctx, orgUUID, stackID, opts)` - List drift runs
  - `Get(ctx, orgUUID, stackID, driftID)` - Get drift with plan

- **`client.ReviewRequests`** - Pull/merge requests
  - `List(ctx, orgUUID, opts)` - List PRs/MRs
  - `Get(ctx, orgUUID, reviewRequestID, opts)` - Get PR with stack plans

- **`client.Deployments`** - CI/CD deployments
  - `List(ctx, orgUUID, opts)` - List workflow deployments
  - `GetWorkflow(ctx, orgUUID, workflowID)` - Get workflow details
  - `ListForWorkflow(ctx, orgUUID, workflowID, opts)` - List stacks in workflow
  - `ListStackDeployments(ctx, orgUUID, opts)` - List all stack deployments
  - `GetStackDeployment(ctx, orgUUID, deploymentID)` - Get deployment with plan
  - `GetDeploymentLogs(ctx, orgUUID, stackID, deploymentUUID, opts)` - Get terraform apply logs

- **`client.Previews`** - Stack preview debugging
  - `Get(ctx, orgUUID, stackPreviewID)` - Get preview details
  - `GetLogs(ctx, orgUUID, stackPreviewID, opts)` - Get terraform plan logs
  - `ExplainErrors(ctx, orgUUID, stackPreviewID, force)` - Get AI error explanation

## Type Reference

### Common Types

```go
// Pagination
type ListOptions struct {
    Page    int
    PerPage int
}

type PaginatedResult struct {
    Total   int
    Page    int
    PerPage int
}

// Methods
func (p *PaginatedResult) HasNextPage() bool
func (p *PaginatedResult) HasPrevPage() bool
func (p *PaginatedResult) TotalPages() int
```

### Stack Types

```go
type Stack struct {
    StackID          int
    Repository       string
    Path             string
    MetaID           string
    MetaName         string
    Status           string  // ok, failed, drifted, etc.
    DriftStatus      string  // ok, drifted, failed
    DeploymentStatus string  // ok, failed, pending, etc.
    CreatedAt        time.Time
    UpdatedAt        time.Time
    // ... more fields
}
```

### Drift Types

```go
type Drift struct {
    ID           int
    StackID      int
    Status       string  // ok, drifted, failed
    DriftDetails *ChangesetDetails  // Contains terraform plan
    StartedAt    *time.Time
    FinishedAt   *time.Time
    // ... more fields
}

type ChangesetDetails struct {
    Provisioner    string  // terraform, opentofu
    Serial         int64
    ChangesetASCII string  // Terraform plan (up to 4MB)
    ChangesetJSON  string  // JSON plan (up to 16MB)
}
```

### Deployment Types

```go
type WorkflowDeploymentGroup struct {
    ID                        int
    Status                    string  // ok, failed, processing
    CommitTitle               string
    Repository                string
    OkCount                   int
    FailedCount               int
    PendingCount              int
    StackDeploymentTotalCount int
    CreatedAt                 time.Time
    // ... more fields
}

type StackDeployment struct {
    ID               int
    DeploymentUUID   string
    Path             string
    Cmd              []string
    Status           string  // ok, failed, pending, etc.
    ChangesetDetails *ChangesetDetails  // Terraform apply output
    Stack            *Stack
    CreatedAt        time.Time
    // ... more fields
}
```

### Review Request Types

```go
type ReviewRequest struct {
    ReviewRequestID   int
    Platform          string  // github, gitlab, bitbucket
    Number            int     // PR/MR number
    Title             string
    Status            string  // open, merged, closed
    Branch            string
    BaseBranch        string
    Preview           *Preview  // Summary of latest preview
    // ... more fields
}

type StackPreview struct {
    StackPreviewID   int
    Status           string  // changed, unchanged, failed
    Stack            *Stack
    ChangesetDetails *ChangesetDetails  // Terraform plan
    ResourceChanges  *ResourceChanges   // Change summary
    // ... more fields
}
```

## Testing

Run the SDK tests:

```bash
go test ./sdk/terramate -v
go test ./sdk/terramate -race
go test ./sdk/terramate -cover
```

Current test coverage: **89.2%** with **160+ tests**

All SDK tests include:
- ✅ Response parsing validation
- ✅ Query parameter handling
- ✅ Input validation
- ✅ API error handling
- ✅ Authentication headers
- ✅ Context cancellation and timeout
- ✅ Race condition detection (`-race` flag)

## Contributing

See the main [Contributing Guide](../../CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

## Support

- 📖 [Terramate Documentation](https://terramate.io/docs)
- 💬 [Community Discord](https://terramate.io/discord)
- 🐛 [Issue Tracker](https://github.com/terramate-io/terramate-mcp-server/issues)
- 📧 [Email Support](mailto:support@terramate.io)

## Related

- [Terramate MCP Server](../../README.md) - Main MCP server documentation
- [Terramate Cloud](https://cloud.terramate.io) - Web UI and collaboration platform
- [Terramate CLI](https://github.com/terramate-io/terramate) - IaC orchestration tool

