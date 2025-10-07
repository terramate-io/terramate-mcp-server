from typing import Any
import base64
import httpx
from terramate_mcp import config


class TerramateApiError(Exception):
    pass


async def make_terramate_request(
    endpoint: str, method: str = "GET", data: dict | None = None
) -> dict[str, Any] | None:
    """Make a request to the Terramate Cloud API with auth and error handling."""
    headers = {
        "User-Agent": config.USER_AGENT,
        "Accept": "application/json",
        "Content-Type": "application/json",
    }

    # Read API key at call-time to respect test patches
    api_key = config.TERRAMATE_API_KEY
    if api_key:
        auth_string = base64.b64encode(f"{api_key}:".encode()).decode()
        headers["Authorization"] = f"Basic {auth_string}"
    else:
        raise ValueError(
            "No authentication configured. Set TERRAMATE_API_KEY environment variable."
        )

    url = f"{config.TERRAMATE_API_BASE}{endpoint}"
    if method not in {"GET", "POST", "PATCH", "DELETE"}:
        raise ValueError(f"Unsupported HTTP method: {method}")

    async with httpx.AsyncClient() as client:
        try:
            if method == "GET":
                response = await client.get(url, headers=headers)
            elif method == "POST":
                response = await client.post(url, headers=headers, json=data)
            elif method == "PATCH":
                response = await client.patch(url, headers=headers, json=data)
            elif method == "DELETE":
                response = await client.delete(url, headers=headers)

            response.raise_for_status()

            if response.status_code == 204 or not response.content:
                return {"success": True, "status_code": response.status_code}

            return response.json()

        except httpx.HTTPStatusError as e:
            error_detail = f"HTTP {e.response.status_code}"
            try:
                error_body = e.response.json()
                if "error_message" in error_body:
                    error_detail += f": {error_body['error_message']}"
            except Exception:
                error_detail += f": {e.response.text}"
            raise TerramateApiError(f"Terramate API error - {error_detail}")
        except Exception as e:
            raise TerramateApiError(f"Request failed: {str(e)}")
