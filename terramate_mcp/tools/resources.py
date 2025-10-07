from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_resources(
    org_uuid: str,
    operation: Literal["list", "get", "deployments", "drift"] = "list",
    resource_uuid: Optional[str] = None,
    page: int = 1,
    per_page: int = 10,
    status: Optional[str] = None,
    provider: Optional[str] = None,
    technology: Optional[str] = None,
    repository: Optional[str] = None,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Resource management and monitoring operations.

    Args:
        operation: Operation to perform (list, get, deployments, drift)
        org_uuid: Organization UUID
        resource_uuid: Resource UUID (required for some operations)
        page: Page number for list operations
        per_page: Items per page for list operations
        status: Filter by resource status
        provider: Filter by provider
        technology: Filter by technology
        repository: Filter by repository
        format: Output format (formatted, json)

    Returns:
        Resource data in requested format
    """
    if operation == "list":
        params = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        if provider:
            params["provider"] = provider
        if technology:
            params["technology"] = technology
        if repository:
            params["repository"] = repository

        endpoint = f"/v1/resources/{org_uuid}"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "get":
        if not resource_uuid:
            raise ValueError("resource_uuid is required for 'get' operation")
        data = await make_terramate_request(f"/v1/resources/{org_uuid}/{resource_uuid}")
        return format_data(data, format)

    elif operation == "deployments":
        if not resource_uuid:
            raise ValueError("resource_uuid is required for 'deployments' operation")

        params = {"page": page, "per_page": per_page}
        endpoint = f"/v1/resources/{org_uuid}/{resource_uuid}/deployments"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "drift":
        if not resource_uuid:
            raise ValueError("resource_uuid is required for 'drift' operation")
        data = await make_terramate_request(
            f"/v1/resources/{org_uuid}/{resource_uuid}/drift"
        )
        return format_data(data, format)
    else:
        raise ValueError(f"Unknown operation: {operation}")
