package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// ListDeployments creates an MCP tool that lists workflow deployments (CI/CD runs) in an organization.
func ListDeployments(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_list_deployments",
			Description: `List workflow deployments (CI/CD runs) in a Terramate Cloud organization.

This tool retrieves CI/CD workflow runs from GitHub Actions, GitLab CI, or other platforms.
Each deployment shows status counts for all stacks deployed in that run.

Use this to:
- Monitor recent deployment activity
- Find failed deployments
- Track CI/CD performance
- See deployment history

Supported filters:
- repository: Filter by repository URLs
- status: Filter by deployment status (ok, failed, processing)
- search: Search commit SHA, title, branch
- page, per_page: Pagination (max: 100)

Response includes:
- deployments: Array of workflow deployment groups
- Each deployment shows:
  * Status counts (ok_count, failed_count, pending_count, etc.)
  * Commit info (title, SHA, branch)
  * Timestamps (created_at, started_at, finished_at)
  * Optional review_request (if from a PR)`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"repository": map[string]interface{}{
						"type":        "array",
						"description": "Filter by repository URLs",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by status (ok, failed, processing)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Search commit SHA, title, or branch",
					},
					"page": map[string]interface{}{
						"type":        "number",
						"description": "Page number for pagination",
					},
					"per_page": map[string]interface{}{
						"type":        "number",
						"description": "Number of items per page (max: 100)",
					},
				},
				Required: []string{"organization_uuid"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			opts := &terramate.DeploymentsListOptions{}

			if page := request.GetInt("page", 0); page > 0 {
				opts.Page = page
			}
			if perPage := request.GetInt("per_page", 0); perPage > 0 {
				if perPage > 100 {
					return mcp.NewToolResultError("Per page value must not exceed 100."), nil
				}
				opts.PerPage = perPage
			}

			opts.Search = request.GetString("search", "")
			opts.Repository = request.GetStringSlice("repository", nil)
			opts.Status = request.GetStringSlice("status", nil)

			result, _, err := client.Deployments.List(ctx, orgUUID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list deployments: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}

// GetStackDeployment creates an MCP tool that retrieves detailed stack deployment information including terraform plan.
func GetStackDeployment(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_stack_deployment",
			Description: `Get detailed information about a specific stack deployment including the terraform plan.

This tool retrieves complete deployment details for a single stack including:
- changeset_details: Terraform apply plan output (ASCII format, up to 4MB)
- Stack metadata
- Command executed
- Timestamps (created, started, finished)
- Deployment status

Use this to:
- See what was actually deployed for a stack
- Review terraform apply output
- Debug failed deployments
- Audit deployment history

The changeset_details field contains the terraform plan that was applied,
which is useful for understanding what infrastructure changes were made.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_deployment_id": map[string]interface{}{
						"type":        "number",
						"description": "Stack Deployment ID",
					},
				},
				Required: []string{"organization_uuid", "stack_deployment_id"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			stackDeploymentID, err := request.RequireInt("stack_deployment_id")
			if err != nil {
				return mcp.NewToolResultError("Stack Deployment ID is required and must be a number."), nil
			}
			if stackDeploymentID <= 0 {
				return mcp.NewToolResultError("Stack Deployment ID must be positive."), nil
			}

			deployment, _, err := client.Deployments.GetStackDeployment(ctx, orgUUID, stackDeploymentID)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Stack Deployment with ID %d not found.", stackDeploymentID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get stack deployment: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(deployment, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}

// GetDeploymentLogs creates an MCP tool that retrieves terraform deployment logs for AI analysis.
func GetDeploymentLogs(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_deployment_logs",
			Description: `Get terraform deployment logs for analyzing failed or running deployments.

This tool retrieves the raw terraform apply/destroy command output (stdout/stderr)
which can then be analyzed by AI to understand deployment failures and suggest fixes.

Use this to:
- Debug terraform apply failures in CI/CD
- Analyze provider errors during deployment
- Understand why resources failed to create/update/destroy
- Get detailed error context and stack traces

Workflow for debugging failed deployment:
1. tmc_list_deployments to find failed deployments
2. Use SDK to get workflow details and find failed stack deployment
3. tmc_get_deployment_logs to fetch raw terraform apply logs
4. AI analyzes logs to explain the issue and suggest remediation

Logs are paginated and can be filtered by channel:
- stderr: Error messages and warnings (most useful for debugging)
- stdout: Standard terraform apply output

Note: Requires stack_id and deployment_uuid from the deployment object.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_id": map[string]interface{}{
						"type":        "number",
						"description": "Stack ID from the deployment",
					},
					"deployment_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Deployment UUID from stack deployment object",
					},
					"channel": map[string]interface{}{
						"type":        "string",
						"description": "Filter by channel (stdout or stderr)",
					},
					"page": map[string]interface{}{
						"type":        "number",
						"description": "Page number for pagination",
					},
					"per_page": map[string]interface{}{
						"type":        "number",
						"description": "Number of items per page",
					},
				},
				Required: []string{"organization_uuid", "stack_id", "deployment_uuid"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			stackID, err := request.RequireInt("stack_id")
			if err != nil {
				return mcp.NewToolResultError("Stack ID is required and must be a number."), nil
			}
			if stackID <= 0 {
				return mcp.NewToolResultError("Stack ID must be positive."), nil
			}

			deploymentUUID, err := request.RequireString("deployment_uuid")
			if err != nil {
				return mcp.NewToolResultError("Deployment UUID is required and must be a string."), nil
			}

			opts := &terramate.DeploymentLogsOptions{}
			if page := request.GetInt("page", 0); page > 0 {
				opts.Page = page
			}
			if perPage := request.GetInt("per_page", 0); perPage > 0 {
				opts.PerPage = perPage
			}
			opts.Channel = request.GetString("channel", "")

			logs, _, err := client.Deployments.GetDeploymentLogs(ctx, orgUUID, stackID, deploymentUUID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Deployment logs not found for stack %d and deployment %s.", stackID, deploymentUUID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get deployment logs: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(logs, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
