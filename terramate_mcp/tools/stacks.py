from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_stacks(
    org_uuid: str,
    operation: Literal["list", "get", "update"] = "list",
    stack_id: Optional[int] = None,
    page: int = 1,
    per_page: int = 100,
    status: Optional[str] = None,
    repository: Optional[str] = None,
    search: Optional[str] = None,
    is_archived: Optional[bool] = None,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Get detailed stack information and manage stacks.

    Use this tool for questions about:
    - "Show me stacks with status 'drifted'"
    - "List all stacks in repository X"
    - "What are the details of stack Y?"
    - "Find stacks matching search term"

    Returns detailed stack information with pagination. Each stack includes:
    repository, path, status, metadata, tags, timestamps, and more.

    For simple stack counts, use terramate_dashboard instead.

    Args:
        operation: Operation to perform (list, get, update)
        org_uuid: Organization UUID
        stack_id: Stack ID (required for get/update operations)
        page: Page number for list operation
        per_page: Items per page for list operation
        status: Filter by stack status (ok, drifted, failed, unknown)
        repository: Filter by repository
        search: Search term for stack name/description/path
        is_archived: Archive/unarchive stack (update operation)
        format: Output format (formatted, json)

    Returns:
        Detailed stack data in requested format
    """
    if operation == "list":
        params = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        if repository:
            params["repository"] = repository
        if search:
            params["search"] = search

        endpoint = f"/v1/stacks/{org_uuid}"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        if format == OutputFormat.FORMATTED and data and "stacks" in data:
            return format_data(data["stacks"], format)
        return format_data(data, format)

    elif operation == "get":
        if not stack_id:
            raise ValueError("stack_id is required for 'get' operation")
        data = await make_terramate_request(f"/v1/stacks/{org_uuid}/{stack_id}")
        return format_data(data, format)

    elif operation == "update":
        if not stack_id:
            raise ValueError("stack_id is required for 'update' operation")
        if is_archived is None:
            raise ValueError("is_archived parameter is required for 'update' operation")

        payload = {"is_archived": is_archived}
        data = await make_terramate_request(
            f"/v1/stacks/{org_uuid}/{stack_id}", "PATCH", payload
        )
        return format_data(data, format)

    else:
        raise ValueError(f"Unknown operation: {operation}")
