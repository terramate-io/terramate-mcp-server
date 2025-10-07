from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_review_requests(
    org_uuid: str,
    operation: Literal["list", "get"] = "list",
    review_request_id: Optional[int] = None,
    page: int = 1,
    per_page: int = 10,
    status: Optional[str] = None,
    repository: Optional[str] = None,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Review request management operations.

    Args:
        operation: Operation to perform (list, get)
        org_uuid: Organization UUID
        review_request_id: Review request ID (required for get operation)
        page: Page number for list operations
        per_page: Items per page for list operations
        status: Filter by review request status
        repository: Filter by repository
        format: Output format (formatted, json)

    Returns:
        Review request data in requested format
    """
    if operation == "list":
        params = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        if repository:
            params["repository"] = repository

        endpoint = f"/v1/review_requests/{org_uuid}"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "get":
        if not review_request_id:
            raise ValueError("review_request_id is required for 'get' operation")
        data = await make_terramate_request(
            f"/v1/review_requests/{org_uuid}/{review_request_id}"
        )
        return format_data(data, format)
    else:
        raise ValueError(f"Unknown operation: {operation}")
