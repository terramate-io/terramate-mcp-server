import os

USER_AGENT = "terramate-mcp/1.0"
TERRAMATE_API_KEY = os.getenv("TERRAMATE_API_KEY")
TERRAMATE_LOCATION = os.getenv("TERRAMATE_LOCATION")


# Determine API base URL based on location
def get_api_base() -> str:
    """Get the Terramate API base URL based on the configured location."""
    location = TERRAMATE_LOCATION

    if not location:
        return None  # Will be validated at startup

    location = location.lower()

    if location == "eu":
        return "https://api.terramate.io"
    elif location == "us":
        return "https://us.api.terramate.io"
    else:
        return None  # Invalid location, will be validated at startup


TERRAMATE_API_BASE = get_api_base()
