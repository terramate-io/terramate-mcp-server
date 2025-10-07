from enum import Enum


class OutputFormat(str, Enum):
    """Output format options for API responses."""

    FORMATTED = "formatted"
    JSON = "json"


class ToolCategory(str, Enum):
    """Tool categories for discovery."""

    CORE = "core"
    WORKFLOW = "workflow"
    ADVANCED = "advanced"
