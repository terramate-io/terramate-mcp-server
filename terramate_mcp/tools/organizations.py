from typing import Optional, Union, Literal
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.types import OutputFormat
from terramate_mcp.utils.formatting import format_data
from terramate_mcp.http.client import make_terramate_request


@mcp.tool()
async def terramate_organizations(
    format: OutputFormat = OutputFormat.FORMATTED,
) -> Union[str, dict, list]:
    """
    Get organization membership information.

    Returns the organization associated with the current API key.

    Note: The GET endpoints for `/v1/organizations` and `/v1/organizations/{org_uuid}`
    are not implemented as they only support JWT token authentication, not API keys.
    This tool uses `/v1/memberships` which supports both API key and JWT authentication.

    Args:
        format: Output format (formatted, json)

    Returns:
        Organization membership data in requested format
    """
    # Returns the organization associated with the current API key
    data = await make_terramate_request("/v1/memberships")
    return format_data(data, format)
