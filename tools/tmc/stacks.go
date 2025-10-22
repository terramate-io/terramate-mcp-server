package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// ListStacks creates an MCP tool that lists stacks in a Terramate Cloud organization.
func ListStacks(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_list_stacks",
			Description: `List stacks in a Terramate Cloud organization with optional filtering and pagination.

This tool retrieves stacks matching the provided filters. Use tmc_authenticate first to get the organization UUID.

Supported filters:
- repository: Filter by exact repository URLs (e.g., "github.com/owner/repo")
- target: Filter by target environment
- status: Filter by status (canceled, drifted, failed, ok, unknown)
- deployment_status: Filter by deployment status
- drift_status: Filter by drift status (ok, drifted, failed, unknown)
- draft: Filter by draft status (true/false)
- is_archived: Filter by archived status (true/false)
- search: Substring search on meta_id, meta_name, meta_description, and path
- meta_id: Filter by exact meta ID
- meta_tag: Filter by tags (can specify multiple)
- deployment_uuid: Filter by deployment UUID
- policy_severity: Filter by policy check results (missing, none, passed, low, medium, high)
- page: Page number for pagination (default: 1)
- per_page: Number of items per page (default: 20)
- sort: Sort fields (can specify multiple)

Response includes:
- stacks: Array of stack objects with metadata, status, tags, and resource information
- paginated_result: Pagination info (total, page, per_page)`,
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
					"target": map[string]interface{}{
						"type":        "array",
						"description": "Filter by target environment",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by status (canceled, drifted, failed, ok, unknown)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"deployment_status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by deployment status",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"drift_status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by drift status (ok, drifted, failed, unknown)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"draft": map[string]interface{}{
						"type":        "boolean",
						"description": "Filter by draft status",
					},
					"is_archived": map[string]interface{}{
						"type":        "array",
						"description": "Filter by archived status",
						"items": map[string]interface{}{
							"type": "boolean",
						},
					},
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Substring search on meta_id, meta_name, meta_description, and path",
					},
					"meta_id": map[string]interface{}{
						"type":        "string",
						"description": "Filter by exact meta ID",
					},
					"meta_tag": map[string]interface{}{
						"type":        "array",
						"description": "Filter by tags",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"deployment_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Filter by deployment UUID",
					},
					"policy_severity": map[string]interface{}{
						"type":        "array",
						"description": "Filter by policy check results (missing, none, passed, low, medium, high)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"page": map[string]interface{}{
						"type":        "number",
						"description": "Page number for pagination",
					},
					"per_page": map[string]interface{}{
						"type":        "number",
						"description": "Number of items per page",
					},
					"sort": map[string]interface{}{
						"type":        "array",
						"description": "Sort fields",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				Required: []string{"organization_uuid"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Parse organization_uuid.
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			// Build options from request.
			opts := &terramate.StacksListOptions{}

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
			opts.Search = request.GetString("search", "")
			opts.MetaID = request.GetString("meta_id", "")
			opts.DeploymentUUID = request.GetString("deployment_uuid", "")

			// Get draft parameter (optional boolean pointer).
			if draft, draftErr := request.RequireBool("draft"); draftErr == nil {
				opts.Draft = &draft
			}

			// Get string array parameters.
			opts.Repository = request.GetStringSlice("repository", nil)
			opts.Target = request.GetStringSlice("target", nil)
			opts.Status = request.GetStringSlice("status", nil)
			opts.DeploymentStatus = request.GetStringSlice("deployment_status", nil)
			opts.DriftStatus = request.GetStringSlice("drift_status", nil)
			opts.MetaTag = request.GetStringSlice("meta_tag", nil)
			opts.PolicySeverity = request.GetStringSlice("policy_severity", nil)
			opts.Sort = request.GetStringSlice("sort", nil)

			// Get boolean array parameter.
			opts.IsArchived = request.GetBoolSlice("is_archived", nil)

			// Call the API.
			result, _, err := client.Stacks.List(ctx, orgUUID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list stacks: %v", err)), nil
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

// GetStack creates an MCP tool that retrieves a specific stack by ID.
func GetStack(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_stack",
			Description: `Get details for a specific stack in a Terramate Cloud organization.

This tool retrieves detailed information about a specific stack, including:
- Stack metadata (name, description, tags)
- Status information (deployment status, drift status)
- Related stacks (from other targets with the same repository and meta_id)
- Resource information and policy check results

Use tmc_authenticate first to get the organization UUID, and tmc_list_stacks to find stack IDs.

Response includes:
- Full stack object with all metadata fields
- related_stacks: Array of related stacks from other targets
- resources: Resource count and policy check results`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_id": map[string]interface{}{
						"type":        "number",
						"description": "Stack ID to retrieve",
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

			// Call the API.
			stack, _, err := client.Stacks.Get(ctx, orgUUID, stackID)
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
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get stack: %v", err)), nil
			}

			// Format response.
			jsonData, err := json.MarshalIndent(stack, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
