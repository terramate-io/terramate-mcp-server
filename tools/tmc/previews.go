package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// GetStackPreviewLogs creates an MCP tool that retrieves terraform command logs for AI analysis.
func GetStackPreviewLogs(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_stack_preview_logs",
			Description: `Get terraform command logs for analyzing failed or running stack previews.

This tool retrieves the raw terraform command output (stdout/stderr) which can then
be analyzed by AI to understand what went wrong and how to fix it.

Use this to:
- Debug terraform plan failures in pull requests
- Analyze provider error messages
- Understand validation and syntax errors
- Get detailed stack traces and error context

Workflow for debugging failed PR:
1. tmc_get_review_request to find failed stack_preview (status: "failed")
2. tmc_get_stack_preview_logs to fetch raw terraform logs
3. AI analyzes logs in context to explain the issue and suggest fixes

Logs are paginated and can be filtered by channel:
- stderr: Error messages and warnings (most useful for debugging)
- stdout: Standard terraform output

Tip: For failed previews, fetch stderr channel first for error messages.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_preview_id": map[string]interface{}{
						"type":        "number",
						"description": "Stack Preview ID (from tmc_get_review_request)",
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
				Required: []string{"organization_uuid", "stack_preview_id"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			stackPreviewID, err := request.RequireInt("stack_preview_id")
			if err != nil {
				return mcp.NewToolResultError("Stack Preview ID is required and must be a number."), nil
			}
			if stackPreviewID <= 0 {
				return mcp.NewToolResultError("Stack Preview ID must be positive."), nil
			}

			opts := &terramate.PreviewLogsOptions{}
			if page := request.GetInt("page", 0); page > 0 {
				opts.Page = page
			}
			if perPage := request.GetInt("per_page", 0); perPage > 0 {
				opts.PerPage = perPage
			}
			opts.Channel = request.GetString("channel", "")

			logs, _, err := client.Previews.GetLogs(ctx, orgUUID, stackPreviewID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Stack Preview with ID %d not found.", stackPreviewID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get logs: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(logs, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
