from typing import Optional
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import ToolCategory


@mcp.tool()
async def discover_tools(category: Optional[ToolCategory] = None) -> str:
    """
    Discover available tools by category for better navigation.

    Args:
        category: Filter tools by category (core, workflow, advanced)

    Returns:
        List of available tools with descriptions
    """
    tools_by_category = {
        ToolCategory.CORE: [
            "terramate_organizations - Get organization membership info for current API key",
            "terramate_stacks - Get detailed stack information with filtering and search",
            "terramate_deployments - View deployment history and logs",
            "terramate_drifts - View drift detection results and generate summaries",
            "terramate_alerts - Monitor and manage infrastructure alerts",
            "terramate_dashboard - Get stack counts and health metrics (use for 'how many stacks')",
        ],
        ToolCategory.WORKFLOW: [
            "terramate_trigger_workflow - Complete stack triggering workflow with PR creation",
            "terramate_stack_operations - Run Terramate CLI commands (list, run, generate, fmt)",
        ],
        ToolCategory.ADVANCED: [
            "terramate_resources - Resource management and monitoring",
            "terramate_review_requests - Review request management",
            "terramate_logs - Deployment and drift log management",
        ],
    }

    if category:
        if category not in tools_by_category:
            return f"Unknown category: {category}. Available categories: {', '.join(tools_by_category.keys())}"
        tools = tools_by_category[category]
        return f"**{category.upper()} TOOLS:**\n" + "\n".join(
            f"• {tool}" for tool in tools
        )

    result = "**TERRAMATE MCP SERVER TOOLS**\n\n"
    for cat, tools in tools_by_category.items():
        result += f"**{cat.upper()}:**\n"
        result += "\n".join(f"• {tool}" for tool in tools) + "\n\n"

    return result
