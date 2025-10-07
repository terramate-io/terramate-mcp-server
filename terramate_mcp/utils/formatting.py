from typing import Any, Union
from .types import OutputFormat


def format_stack_info(stack: dict) -> str:
    """Format a stack object into a readable string."""
    status_emoji = {
        "ok": "✅",
        "drifted": "⚠️",
        "failed": "❌",
        "unknown": "❓",
        "canceled": "🚫",
    }

    emoji = status_emoji.get(stack.get("status", "unknown"), "❓")

    return f"""
{emoji} Stack: {stack.get('meta_name', 'Unnamed')} (ID: {stack.get('stack_id')})
Repository: {stack.get('repository', 'N/A')}
Path: {stack.get('path', 'N/A')}
Status: {stack.get('status', 'unknown')}
Target: {stack.get('target', 'default')}
Meta ID: {stack.get('meta_id', 'N/A')}
Description: {stack.get('meta_description', 'No description')}
Tags: {', '.join(stack.get('meta_tags', []))}
Updated: {stack.get('updated_at', 'N/A')}
"""


def format_data(data: Any, format_type: OutputFormat) -> Union[str, dict, list]:
    """Format data based on the requested output format."""
    if format_type == OutputFormat.JSON:
        return data
    elif format_type == OutputFormat.FORMATTED:
        if isinstance(data, list) and data and isinstance(data[0], dict):
            if "stack_id" in data[0]:  # Stack data
                return "\n---\n".join([format_stack_info(item) for item in data])
            elif "org_display_name" in data[0]:  # Organization data
                return "\n---\n".join(
                    [
                        f"Organization: {item.get('org_display_name', 'Unnamed')}\n"
                        f"Short Name: {item.get('org_name', 'N/A')}\n"
                        f"UUID: {item.get('org_uuid', 'N/A')}\n"
                        f"Role: {item.get('role', 'N/A')}\n"
                        f"Status: {item.get('status', 'N/A')}"
                        for item in data
                    ]
                )
        return str(data)
    return data
