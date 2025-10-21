package tmc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
)

// Authenticate creates an MCP tool that authenticates with Terramate Cloud
// and returns the user's organization information
func Authenticate(client *terramate.Client) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.Tool{
			Name: "tmc_authenticate",
			Description: `Authenticate with Terramate Cloud and retrieve organization membership information.

This tool verifies the API key is valid and returns essential organization details including:
- Organization UUID (required for most other Terramate Cloud API endpoints)
- Organization name and display name
- User's role (admin or member)
- Membership status

Use this tool first before calling other Terramate Cloud operations to get the organization UUID.`,
			InputSchema: mcp.ToolInputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
				Required:   []string{},
			},
		},
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Call the memberships endpoint to authenticate and get org info
			memberships, _, err := client.Memberships.List(ctx)
			if err != nil {
				// Check if it's an API error
				if apiErr, ok := err.(*terramate.APIError); ok {
					if apiErr.IsUnauthorized() {
						return mcp.NewToolResultError(terramate.ErrAuthenticationFailed), nil
					}
					return mcp.NewToolResultError(fmt.Sprintf("API error: %s", apiErr.Error())), nil
				}
				return mcp.NewToolResultError(fmt.Sprintf("Failed to authenticate: %v", err)), nil
			}

			if len(memberships) == 0 {
				return mcp.NewToolResultError("No organization memberships found for this API key"), nil
			}

			// Format response with all memberships
			response := map[string]interface{}{
				"authenticated": true,
				"memberships":   memberships,
			}

			// If there's only one membership (typical for API keys), also provide it at the top level
			if len(memberships) == 1 {
				response["organization_uuid"] = memberships[0].OrgUUID
				response["organization_name"] = memberships[0].OrgName
				response["organization_display_name"] = memberships[0].OrgDisplayName
				response["organization_domain"] = memberships[0].OrgDomain
				response["member_id"] = memberships[0].MemberID
				response["role"] = memberships[0].Role
				response["status"] = memberships[0].Status
			}

			jsonData, err := json.MarshalIndent(response, "", "  ")
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal response: %v", err)), nil
			}

			return mcp.NewToolResultText(string(jsonData)), nil
		},
	}
}
