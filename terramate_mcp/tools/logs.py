from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_logs(
    org_uuid: str,
    stack_id: int,
    deployment_uuid: str,
    operation: Literal["get", "send", "summarize"] = "get",
    page: int = 1,
    per_page: int = 50,
    channel: Optional[str] = None,
    log_lines: Optional[list] = None,
    force: bool = False,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Deployment and drift log management.

    Args:
        operation: Operation to perform (get, send, summarize)
        org_uuid: Organization UUID
        stack_id: Stack ID
        deployment_uuid: Deployment UUID
        page: Page number for get operation
        per_page: Items per page for get operation
        channel: Log channel filter (stdout, stderr)
        log_lines: Log lines to send (for send operation)
        force: Force regeneration (for summarize operation)
        format: Output format (formatted, json)

    Returns:
        Log data in requested format
    """
    if operation == "get":
        params = {"page": page, "per_page": per_page}
        if channel:
            params["channel"] = channel

        endpoint = (
            f"/v1/stacks/{org_uuid}/{stack_id}/deployments/{deployment_uuid}/logs"
        )
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "send":
        if not log_lines:
            raise ValueError("log_lines parameter is required for 'send' operation")
        data = await make_terramate_request(
            f"/v1/stacks/{org_uuid}/{stack_id}/deployments/{deployment_uuid}/logs",
            "POST",
            log_lines,
        )
        return format_data(data, format)

    elif operation == "summarize":
        endpoint = f"/v1/stacks/{org_uuid}/{stack_id}/deployments/{deployment_uuid}/logs/summarize"
        if force:
            endpoint += "?force=true"

        data = await make_terramate_request(endpoint, "POST")
        return format_data(data, format)
    else:
        raise ValueError(f"Unknown operation: {operation}")
