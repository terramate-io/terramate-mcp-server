package tools

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/terramate-io/terramate-mcp-server/sdk/terramate"
	"github.com/terramate-io/terramate-mcp-server/tools/tmc"
)

// ToolHandlers contains all MCP tool handlers
type ToolHandlers struct {
	tmcClient *terramate.Client
}

// New creates new tool handlers
func New(tmcClient *terramate.Client) *ToolHandlers {
	return &ToolHandlers{
		tmcClient: tmcClient,
	}
}

// Tools returns all MCP tools for Terramate Cloud
func (th *ToolHandlers) Tools() []server.ServerTool {
	tools := []server.ServerTool{}

	// Register authentication tool
	tools = append(tools, tmc.Authenticate(th.tmcClient))

	// Register stacks tools
	tools = append(tools, tmc.ListStacks(th.tmcClient))
	tools = append(tools, tmc.GetStack(th.tmcClient))

	// Register drift tools
	tools = append(tools, tmc.ListDrifts(th.tmcClient))
	tools = append(tools, tmc.GetDrift(th.tmcClient))

	// Register review request tools
	tools = append(tools, tmc.ListReviewRequests(th.tmcClient))
	tools = append(tools, tmc.GetReviewRequest(th.tmcClient))

	// Register deployment tools
	tools = append(tools, tmc.ListDeployments(th.tmcClient))
	tools = append(tools, tmc.GetStackDeployment(th.tmcClient))
	tools = append(tools, tmc.GetDeploymentLogs(th.tmcClient))

	// Register preview tools
	tools = append(tools, tmc.GetStackPreviewLogs(th.tmcClient))

	// Register resources tools
	tools = append(tools, tmc.ListResources(th.tmcClient))
	tools = append(tools, tmc.GetResource(th.tmcClient))

	// TODO: Add more tools here
	// tools = append(tools, tmc.ListAlerts(th.tmcClient))

	return tools
}
