from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_alerts(
    org_uuid: str,
    operation: Literal["list", "get", "affected_stacks", "timeline"] = "list",
    alert_uuid: Optional[str] = None,
    page: int = 1,
    per_page: int = 10,
    severity: Optional[str] = None,
    status: Optional[str] = None,
    unassigned: Optional[bool] = None,
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Comprehensive alert monitoring and management.

    Args:
        operation: Operation to perform (list, get, affected_stacks, timeline)
        org_uuid: Organization UUID
        alert_uuid: Alert UUID (required for some operations)
        page: Page number for list operations
        per_page: Items per page for list operations
        severity: Filter by alert severity
        status: Filter by alert status
        unassigned: Filter for unassigned alerts
        format: Output format (formatted, json)

    Returns:
        Alert data in requested format
    """
    if operation == "list":
        params = {"page": page, "per_page": per_page}
        if severity:
            params["severity"] = severity
        if status:
            params["status"] = status
        if unassigned is not None:
            params["unassigned"] = str(unassigned).lower()

        endpoint = f"/v1/alerts/{org_uuid}"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "get":
        if not alert_uuid:
            raise ValueError("alert_uuid is required for 'get' operation")
        data = await make_terramate_request(f"/v1/alerts/{org_uuid}/{alert_uuid}")
        return format_data(data, format)

    elif operation == "affected_stacks":
        if not alert_uuid:
            raise ValueError("alert_uuid is required for 'affected_stacks' operation")

        params = {"page": page, "per_page": per_page}
        endpoint = f"/v1/alerts/{org_uuid}/{alert_uuid}/stacks"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)

    elif operation == "timeline":
        if not alert_uuid:
            raise ValueError("alert_uuid is required for 'timeline' operation")

        params = {"page": page, "per_page": per_page}
        endpoint = f"/v1/alerts/{org_uuid}/{alert_uuid}/timeline"
        if params:
            query_string = "&".join([f"{k}={v}" for k, v in params.items()])
            endpoint += f"?{query_string}"

        data = await make_terramate_request(endpoint)
        return format_data(data, format)
    else:
        raise ValueError(f"Unknown operation: {operation}")
