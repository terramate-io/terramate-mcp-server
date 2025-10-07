from typing import Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_dashboard(
    org_uuid: str,
    operation: Literal["stacks"] = "stacks",
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict]:
    """
    Get dashboard metrics and stack counts.

    Use this tool for questions about:
    - "How many stacks does the organization have?"
    - Stack count statistics and health metrics
    - Quick overview of stack status distribution

    Returns aggregated counts (ok_count, drifted_count, failed_count) without pagination.
    For detailed stack information, use terramate_stacks instead.

    Args:
        operation: Dashboard operation to perform
        org_uuid: Organization UUID
        format: Output format (formatted, json)

    Returns:
        Dashboard data with stack counts in requested format
    """
    if operation == "stacks":
        data = await make_terramate_request(f"/v1/dashboards/{org_uuid}/stacks")
        return format_data(data, format)
    else:
        raise ValueError(f"Unknown operation: {operation}")
