package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// ListDrifts creates an MCP tool that lists drift detection runs for a specific stack.
func ListDrifts(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_list_drifts",
			Description: `List all drift detection runs for a specific stack in Terramate Cloud.

This tool retrieves the history of drift detection runs for a stack. Each drift run represents
a point-in-time check for infrastructure drift. Use this to see all drift runs before fetching
detailed plan output with tmc_get_drift.

Workflow:
1. Use tmc_list_stacks with drift_status=["drifted"] to find drifted stacks
2. Use tmc_list_drifts to see all drift runs for a specific stack
3. Use tmc_get_drift to get the full terraform plan for a specific drift run

Supported filters:
- drift_status: Filter by drift status (ok, drifted, failed)
- grouping_key: Filter by CI/CD grouping key
- page: Page number for pagination (default: 1)
- per_page: Number of items per page (default: 10, max: 100)

Response includes:
- drifts: Array of drift run objects with status, timestamps, and metadata
- paginated_result: Pagination info (total, page, per_page)

Note: The drift_details field (with ASCII plan) is NOT included in list responses.
Use tmc_get_drift to retrieve the full plan output.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_id": map[string]interface{}{
						"type":        "number",
						"description": "Stack ID to get drift runs for",
					},
					"drift_status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by drift status (ok, drifted, failed)",
						"items": map[string]interface{}{
							"type": "string",
							"enum": []string{"ok", "drifted", "failed"},
						},
					},
					"grouping_key": map[string]interface{}{
						"type":        "string",
						"description": "Filter by CI/CD grouping key",
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
				Required: []string{"organization_uuid", "stack_id"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse organization_uuid.
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			// Parse stack_id.
			stackID, err := request.RequireInt("stack_id")
			if err != nil {
				return mcp.NewToolResultError("Stack ID is required and must be a number."), nil
			}
			if stackID <= 0 {
				return mcp.NewToolResultError("Stack ID must be positive."), nil
			}

			// Build options from request.
			opts := &terramate.DriftsListOptions{}

			// Get pagination parameters with validation.
			if page := request.GetInt("page", 0); page > 0 {
				opts.Page = page
			}
			if perPage := request.GetInt("per_page", 0); perPage > 0 {
				if perPage > 100 {
					return mcp.NewToolResultError("Per page value must not exceed 100."), nil
				}
				opts.PerPage = perPage
			}

			// Get string parameters.
			opts.GroupingKey = request.GetString("grouping_key", "")

			// Get string array parameters.
			opts.DriftStatus = request.GetStringSlice("drift_status", nil)

			// Call the API.
			result, _, err := client.Drifts.ListForStack(ctx, orgUUID, stackID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Stack with ID %d not found.", stackID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list drifts: %v", err)), nil
			}

			// Format response.
			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}

// GetDrift creates an MCP tool that retrieves detailed drift information including the terraform plan.
func GetDrift(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_drift",
			Description: `Get detailed drift information including the terraform plan (ASCII output).

This tool retrieves the complete drift detection details for a specific drift run, including:
- drift_details.changeset_ascii: The terraform plan output in ASCII format (up to 4MB)
- drift_details.changeset_json: The terraform plan in JSON format (up to 16MB)
- drift_details.provisioner: Tool used (terraform or opentofu)
- drift_details.serial: Terraform state serial number
- stack: Full stack object with metadata
- Drift metadata (status, timestamps, auth info, command executed)

Use this to pull the terraform plan into context for AI-assisted drift reconciliation.

Workflow:
1. Use tmc_list_stacks with drift_status=["drifted"] to find drifted stacks
2. Use tmc_list_drifts to see drift runs and get a drift_id
3. Use tmc_get_drift to retrieve the full plan for analysis

Response includes the complete Drift object with all fields populated.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_id": map[string]interface{}{
						"type":        "number",
						"description": "Stack ID",
					},
					"drift_id": map[string]interface{}{
						"type":        "number",
						"description": "Drift ID (get from tmc_list_drifts)",
					},
				},
				Required: []string{"organization_uuid", "stack_id", "drift_id"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse organization_uuid.
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			// Parse stack_id.
			stackID, err := request.RequireInt("stack_id")
			if err != nil {
				return mcp.NewToolResultError("Stack ID is required and must be a number."), nil
			}
			if stackID <= 0 {
				return mcp.NewToolResultError("Stack ID must be positive."), nil
			}

			// Parse drift_id.
			driftID, err := request.RequireInt("drift_id")
			if err != nil {
				return mcp.NewToolResultError("Drift ID is required and must be a number."), nil
			}
			if driftID <= 0 {
				return mcp.NewToolResultError("Drift ID must be positive."), nil
			}

			// Call the API.
			drift, _, err := client.Drifts.Get(ctx, orgUUID, stackID, driftID)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Drift with ID %d not found for stack %d.", driftID, stackID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get drift: %v", err)), nil
			}

			// Format response.
			jsonData, err := json.MarshalIndent(drift, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
