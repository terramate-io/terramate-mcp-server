"""
Terramate MCP Server - Main entrypoint

This module serves as the main entrypoint for the Terramate MCP server.
It imports the shared MCP instance and all tool modules to register them.
"""

import sys
import subprocess

# Import shared MCP instance
from terramate_mcp.mcp_instance import mcp

# Import configuration
from terramate_mcp.config import TERRAMATE_API_KEY, TERRAMATE_LOCATION


def check_terramate_cli() -> bool:
    """Check if Terramate CLI is available in PATH."""
    try:
        result = subprocess.run(["which", "terramate"], capture_output=True, timeout=5)
        return result.returncode == 0
    except Exception:
        return False


# Import core API-based tools (always available)
from terramate_mcp.tools import discover
from terramate_mcp.tools import organizations
from terramate_mcp.tools import stacks
from terramate_mcp.tools import deployments
from terramate_mcp.tools import drifts
from terramate_mcp.tools import alerts
from terramate_mcp.tools import resources
from terramate_mcp.tools import review_requests
from terramate_mcp.tools import logs
from terramate_mcp.tools import dashboard

# Conditionally import Terramate CLI-dependent tools
cli_available = check_terramate_cli()
if cli_available:
    from terramate_mcp.tools import workflows

    print("ℹ️ Terramate CLI detected - workflow tools enabled", file=sys.stderr)
else:
    print(
        "⚠️ Terramate CLI not found in PATH - workflow tools disabled",
        file=sys.stderr,
    )
    print(
        "   Install Terramate CLI from: https://terramate.io/docs/cli/installation",
        file=sys.stderr,
    )
    print("   All API-based Terramate Cloud tools are available.", file=sys.stderr)


if __name__ == "__main__":
    # Check for required environment variables
    errors = []

    if not TERRAMATE_API_KEY:
        errors.append("Error: No TERRAMATE_API_KEY environment variable found.")
        errors.append(
            "Please set the TERRAMATE_API_KEY environment variable to authenticate with Terramate Cloud."
        )
        errors.append(
            "For more information on creating API keys, visit: https://terramate.io/docs/cloud/organization/api-keys#managing-api-keys"
        )

    if not TERRAMATE_LOCATION:
        errors.append("Error: No TERRAMATE_LOCATION environment variable found.")
        errors.append(
            "Please set TERRAMATE_LOCATION to either 'us' or 'eu' to specify your Terramate Cloud region."
        )
    elif TERRAMATE_LOCATION.lower() not in ["us", "eu"]:
        errors.append(f"Error: Invalid TERRAMATE_LOCATION '{TERRAMATE_LOCATION}'.")
        errors.append("TERRAMATE_LOCATION must be either 'us' or 'eu'.")

    if errors:
        for error in errors:
            print(error, file=sys.stderr)
        sys.exit(1)

    # Initialize and run the server
    mcp.run(transport="stdio")
