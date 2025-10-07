from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_drifts(
    org_uuid: str,
    operation: Literal["list", "get", "summarize"] = "list",
    stack_id: Optional[int] = None,
    drift_id: Optional[int] = None,
    page: int = 1,
    per_page: int = 10,
    status: Optional[str] = None,
    force: bool = False,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Drift detection viewing and management.

    Args:
        operation: Operation to perform (list, get, summarize)
        org_uuid: Organization UUID
        stack_id: Stack ID (required for some operations)
        drift_id: Drift ID (required for some operations)
        page: Page number for list operations
        per_page: Items per page for list operations
        status: Filter by drift status
        force: Force regeneration for summarize operation
        format: Output format (formatted, json)

    Returns:
        Drift data in requested format
    """
    if operation == "list":
        if not stack_id:
            raise ValueError("stack_id is required for 'list' operation")

        params = {"page": page, "per_page": per_page}
        if status:
            params["drift_status"] = status

        endpoint = f"/v1/stacks/{org_uuid}/{stack_id}/drifts"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "get":
        if not stack_id or not drift_id:
            raise ValueError("stack_id and drift_id are required for 'get' operation")
        data = await make_terramate_request(
            f"/v1/drifts/{org_uuid}/{stack_id}/{drift_id}"
        )
        return format_data(data, format)

    elif operation == "summarize":
        if not stack_id or not drift_id:
            raise ValueError(
                "stack_id and drift_id are required for 'summarize' operation"
            )

        endpoint = f"/v1/drifts/{org_uuid}/{stack_id}/{drift_id}/summarize"
        if force:
            endpoint += "?force=true"

        data = await make_terramate_request(endpoint, "POST")
        return format_data(data, format)
    else:
        raise ValueError(f"Unknown operation: {operation}")
