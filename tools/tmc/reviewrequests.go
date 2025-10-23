package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// ListReviewRequests creates an MCP tool that lists review requests (pull/merge requests) in an organization.
func ListReviewRequests(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_list_review_requests",
			Description: `List review requests (pull requests/merge requests) in a Terramate Cloud organization.

This tool retrieves pull/merge requests from GitHub, GitLab, or Bitbucket that are tracked
in Terramate Cloud. Each review request shows associated terraform previews and deployment status.

Workflow to find stack previews:
1. Use tmc_list_review_requests to find PRs (filter by repository, status, etc.)
2. Use tmc_get_review_request with a review_request_id to see stack previews
3. Each stack preview includes the full terraform plan output

Supported filters:
- status: Filter by PR status (open, merged, closed, approved, changes_requested, review_required)
- repository: Filter by repository URLs
- search: Search PR number, title, commit SHA, branch names
- draft: Filter by draft status
- collaborator_id: Filter by collaborator
- author_uuid: Filter by author user UUID
- page, per_page: Pagination (default: page 1, per_page 10)
- sort: Sort fields (last_updated_at, status, repository)

Response includes:
- review_requests: Array of PR objects with metadata, preview summaries, and collaborators
- paginated_result: Pagination info

Each review_request includes a 'preview' field with summary counts (changed, failed, etc.)
but NOT the actual terraform plans. Use tmc_get_review_request for full plans.`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"status": map[string]interface{}{
						"type":        "array",
						"description": "Filter by PR status (open, merged, closed, approved, changes_requested, review_required)",
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
					"search": map[string]interface{}{
						"type":        "string",
						"description": "Search PR number, title, commit SHA, branch names",
					},
					"draft": map[string]interface{}{
						"type":        "boolean",
						"description": "Filter by draft status",
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

			opts := &terramate.ReviewRequestsListOptions{}

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
			opts.Status = request.GetStringSlice("status", nil)
			opts.Repository = request.GetStringSlice("repository", nil)

			if draft, draftErr := request.RequireBool("draft"); draftErr == nil {
				opts.Draft = &draft
			}

			result, _, err := client.ReviewRequests.List(ctx, orgUUID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to list review requests: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}

// GetReviewRequest creates an MCP tool that retrieves detailed PR information including stack previews.
func GetReviewRequest(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_get_review_request",
			Description: `Get detailed information about a specific review request (PR/MR) including terraform plans for each affected stack.

This tool retrieves complete PR details including:
- PR metadata (title, description, status, branch, etc.)
- Preview summary (counts of changed, failed, unchanged stacks)
- Stack previews with terraform plan output for EACH affected stack
- Review status (approvals, changes requested, checks)
- Collaborators and labels

Use this to:
- View terraform plans for all stacks in a PR
- Analyze what infrastructure changes a PR will make
- Find failed terraform plans in a PR
- Get AI assistance reviewing infrastructure changes

The stack_previews array contains one entry per stack with:
- stack: Full stack object (includes stack_id, path, meta_id)
- changeset_details: Terraform plan (changeset_ascii up to 4MB)
- resource_changes: Summary of creates, updates, deletes
- status: changed, unchanged, failed, running, etc.

Workflow example:
1. tmc_list_review_requests to find open PRs
2. tmc_get_review_request to see all stack plans
3. Analyze terraform plans with AI assistance`,
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"organization_uuid": map[string]interface{}{
						"type":        "string",
						"description": "Organization UUID (get from tmc_authenticate)",
					},
					"review_request_id": map[string]interface{}{
						"type":        "number",
						"description": "Review Request ID (get from tmc_list_review_requests)",
					},
					"exclude_stack_previews": map[string]interface{}{
						"type":        "boolean",
						"description": "Exclude stack previews to get only PR metadata (default: false)",
					},
				},
				Required: []string{"organization_uuid", "review_request_id"},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			orgUUID, err := request.RequireString("organization_uuid")
			if err != nil {
				return mcp.NewToolResultError("Organization UUID is required and must be a string."), nil
			}

			reviewRequestID, err := request.RequireInt("review_request_id")
			if err != nil {
				return mcp.NewToolResultError("Review Request ID is required and must be a number."), nil
			}
			if reviewRequestID <= 0 {
				return mcp.NewToolResultError("Review Request ID must be positive."), nil
			}

			opts := &terramate.ReviewRequestGetOptions{}
			if exclude, excludeErr := request.RequireBool("exclude_stack_previews"); excludeErr == nil {
				opts.ExcludeStackPreviews = exclude
			}

			result, _, err := client.ReviewRequests.Get(ctx, orgUUID, reviewRequestID, opts)
			if err != nil {
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					if apiErr.IsNotFound() {
						return mcp.NewToolResultError(fmt.Sprintf("Review Request with ID %d not found.", reviewRequestID)), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get review request: %v", err)), nil
			}

			jsonData, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
