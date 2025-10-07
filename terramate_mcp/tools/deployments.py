from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_deployments(
    org_uuid: str,
    operation: Literal["list", "get", "logs"] = "list",
    stack_id: Optional[int] = None,
    deployment_uuid: Optional[str] = None,
    workflow_deployment_group_id: Optional[int] = None,
    page: int = 1,
    per_page: int = 10,
    status: Optional[str] = None,
    channel: Optional[str] = None,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Deployment operations and history viewing.

    Args:
        operation: Operation to perform (list, get, logs)
        org_uuid: Organization UUID
        stack_id: Stack ID (required for some operations)
        deployment_uuid: Deployment UUID (required for some operations)
        workflow_deployment_group_id: Workflow deployment group ID
        page: Page number for list operations
        per_page: Items per page for list operations
        status: Filter by deployment status
        channel: Log channel filter (stdout, stderr)
        format: Output format (formatted, json)

    Returns:
        Deployment data in requested format
    """
    if operation == "list":
        if not stack_id:
            raise ValueError("stack_id is required for 'list' operation")

        params = {"page": page, "per_page": per_page}
        if status:
            params["deployment_status"] = status

        endpoint = f"/v1/stacks/{org_uuid}/{stack_id}/deployments"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "get":
        if workflow_deployment_group_id:
            data = await make_terramate_request(
                f"/v1/workflow_deployment_groups/{org_uuid}/{workflow_deployment_group_id}"
            )
        else:
            raise ValueError(
                "workflow_deployment_group_id is required for 'get' operation"
            )
        return format_data(data, format)

    elif operation == "logs":
        if not stack_id or not deployment_uuid:
            raise ValueError(
                "stack_id and deployment_uuid are required for 'logs' operation"
            )

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
    else:
        raise ValueError(f"Unknown operation: {operation}")
