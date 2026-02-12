package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// ListResources creates an MCP tool that lists resources in a Terramate Cloud organization.
func ListResources(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_list_resources",
			Description: `List resources (stack-level plan/state resources) in a Terramate Cloud organization with optional filtering.

This tool retrieves resources matching the provided filters. Use tmc_authenticate first to get the organization UUID.

Supported filters:
- stack_id: Filter by stack ID (list resources for a single stack)
- status: Filter by resource status (ok, drifted, pending)
- technology: Filter by technology (e.g. terraform, opentofu)
- provider: Filter by provider (e.g. aws, gcloud)
- resource_type: Filter by resource type (e.g. vpc, loadbalancer)
- repository: Filter by repository URLs
- target: Filter by deployment target
- extracted_account: Filter by extracted account
- is_archived: Filter by stack archived status (true/false)
- policy_severity: Filter by policy check (missing, none, passed, low, medium, high)
- search: Search in stack title/description/path and resource extracted name/id/address
- page, per_page: Pagination
- sort: Sort (e.g. updated_at,desc or path,asc)

Response includes:
- resources: Array of resource objects with descriptor, stack, status, drifted, pending
- paginated_result: Pagination info`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"stack_id": map[string]interface{}{
						"type":        "number",
						"description": "Filter by stack ID (list resources for this stack only)",
					},
					"status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by resource status (ok, drifted, pending)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"technology": map[string]interface{}{
						"type":        "array",
						"description": "Filter by technology (e.g. terraform, opentofu)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"provider": map[string]interface{}{
						"type":        "array",
						"description": "Filter by provider (e.g. aws, gcloud)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"resource_type": map[string]interface{}{
						"type":        "array",
						"description": "Filter by resource type (e.g. vpc, loadbalancer)",
						"items": map[string]interface{}{
							"type": "string",
						},
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
						"description": "Filter by deployment target",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"extracted_account": map[string]interface{}{
						"type":        "array",
						"description": "Filter by extracted account",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"is_archived": map[string]interface{}{
						"type":        "array",
						"description": "Filter by stack archived status",
						"items": map[string]interface{}{
							"type": "boolean",
						},
					},
					"policy_severity": map[string]interface{}{
						"type":        "array",
						"description": "Filter by policy check (missing, none, passed, low, medium, high)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Search in stack title/description/path and resource name/id/address",
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
						"description": "Sort fields (e.g. updated_at,desc or path,asc)",
						"items": map[string]interface{}{
							"type": "string",
						},
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

			opts := &terramate.ResourcesListOptions{}

			if page := request.GetInt("page", 0); page > 0 {
				opts.Page = page
			}
			if perPage := request.GetInt("per_page", 0); perPage > 0 {
				if perPage > 100 {
					return mcp.NewToolResultError("Per page value must not exceed 100."), nil
				}
				opts.PerPage = perPage
			}

			opts.StackID = request.GetInt("stack_id", 0)
			opts.Search = request.GetString("search", "")
			opts.Status = request.GetStringSlice("status", nil)
			opts.Technology = request.GetStringSlice("technology", nil)
			opts.Provider = request.GetStringSlice("provider", nil)
			opts.Type = request.GetStringSlice("resource_type", nil)
			opts.Repository = request.GetStringSlice("repository", nil)
			opts.Target = request.GetStringSlice("target", nil)
			opts.ExtractedAccount = request.GetStringSlice("extracted_account", nil)
			opts.IsArchived = request.GetBoolSlice("is_archived", nil)
			opts.PolicySeverity = request.GetStringSlice("policy_severity", nil)
			opts.Sort = request.GetStringSlice("sort", nil)

			result, _, err := client.Resources.List(ctx, orgUUID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list resources: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}

// GetResource creates an MCP tool that retrieves a specific resource by UUID.
func GetResource(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_resource",
			Description: `Get details for a specific resource in a Terramate Cloud organization.

Returns the resource with optional details (e.g. values state when available).
Use tmc_authenticate for organization UUID and tmc_list_resources to find resource_uuid.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"resource_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Resource UUID (from tmc_list_resources)",
					},
				},
				Required: []string{"organization_uuid", "resource_uuid"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			resourceUUID, err := request.RequireString("resource_uuid")
			if err != nil {
				return mcp.NewToolResultError("Resource UUID is required and must be a string."), nil
			}

			resource, _, err := client.Resources.Get(ctx, orgUUID, resourceUUID)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Resource %s not found.", resourceUUID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get resource: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(resource, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
