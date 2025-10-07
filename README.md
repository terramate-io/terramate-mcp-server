# Terramate MCP Server

A [Model Context Protocol](https://modelcontextprotocol.io/docs/getting-started/intro) (MCP) server that exposes [Terramate Cloud](https://cloud.terramate.io) and [Terramate CLI](https://github.com/terramate-io/terramate) capabilities through the MCP protocol, enabling AI assistants like Claude and IDEs like Cursor to help manage your infrastructure, monitor deployments, detect drift, and orchestrate workflows using natural language.

## Quick Start

```bash
# 1. Install dependencies
uv sync

# 2. Set environment variables
export TERRAMATE_API_KEY="your_api_key"
export TERRAMATE_LOCATION="eu"  # or "us"

# 3. Test the server
uv run server.py
```

Then integrate with your AI tool:
- **Claude Desktop**: See [Claude Desktop Integration](#-claude-desktop-integration)
- **Cursor**: See [Cursor Integration](#-cursor-integration)

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
  - [Local Setup](#local-installation--setup)
- [Integration with AI Tools](#integration-with-ai-tools)
  - [Claude Desktop Integration](#-claude-desktop-integration)
  - [Cursor Integration](#-cursor-integration)
- [Configuration Reference](#configuration-reference)
- [Available Tools](#available-tools)
  - [Core Tools](#-core-tools)
  - [Workflow & Execution Tools](#-workflow--execution-tools)
  - [Advanced Tools](#-advanced-tools)
  - [Utility Tools](#-utility-tools)
  - [Tool Availability](#tool-availability)

## Installation

### Local Installation & Setup

#### 1. Prerequisites
- **Python 3.13+** installed
- **uv** package manager ([installation guide](https://docs.astral.sh/uv/getting-started/installation/))
- **Terramate Cloud API Key** ([create one here](https://terramate.io/docs/cloud/organization/api-keys#managing-api-keys))
- **Terramate CLI** (optional, [installation guide](https://terramate.io/docs/cli/installation))

#### 2. Clone and Install Dependencies

```bash
# Clone the repository
git clone https://github.com/your-org/terramate-mcp-server.git
cd terramate-mcp-server

# Install dependencies using uv
uv sync
```

#### 3. Configure Environment Variables

Create a `.env` file or export environment variables:

```bash
# Copy the example environment file
cp env.example .env

# Edit with your actual values
export TERRAMATE_API_KEY="your_api_key_here"
export TERRAMATE_LOCATION="eu"  # or "us"
```

#### 4. Test the Server Locally

```bash
# Run the server directly
uv run server.py

# The server will output:
# ℹ️ Terramate CLI detected - all tools available
# (or warning if CLI not found)
```

---

## Integration with AI Tools

### 🤖 Claude Desktop Integration

#### Step 1: Locate Claude Desktop Config

The configuration file location depends on your OS:

- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

#### Step 2: Add Terramate MCP Server

Edit the config file and add the `terramate` server under `mcpServers`:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "uv",
      "args": [
        "--directory",
        "/absolute/path/to/terramate-mcp-server",
        "run",
        "server.py"
      ],
      "env": {
        "TERRAMATE_API_KEY": "your_api_key_here",
        "TERRAMATE_LOCATION": "eu"
      }
    }
  }
}
```

#### Step 3: Restart Claude Desktop

1. Quit Claude Desktop completely
2. Reopen Claude Desktop
3. Look for the 🔨 (hammer) icon in the input area - this indicates MCP tools are available
4. Test with: _"What Terramate tools are available?"_

#### Troubleshooting Claude Desktop

If tools don't appear:

1. **Check logs**:
   - macOS: `~/Library/Logs/Claude/mcp*.log`
   - Windows: `%APPDATA%\Claude\logs\mcp*.log`
   
2. **Common issues**:
   - Invalid API key → Check your `TERRAMATE_API_KEY`
   - Wrong path → Ensure absolute paths are used
   - `uv` not found → Make sure `uv` is in your PATH

---

### 💻 Cursor Integration

#### Step 1: Open Cursor Settings

1. Open Cursor
2. Go to **Settings** → **Cursor Settings** (or press `Cmd/Ctrl + Shift + J`)
3. Navigate to the **MCP** section

Or directly edit the config file:
- **macOS/Linux**: `~/.cursor/mcp.json` or `~/.config/cursor/mcp.json`
- **Windows**: `%APPDATA%\Cursor\mcp.json`

#### Step 2: Add Terramate MCP Server

Add the following configuration to your Cursor MCP settings:

```json
{
  "mcpServers": {
    "terramate": {
      "command": "uv",
      "args": [
        "--directory",
        "/absolute/path/to/terramate-mcp-server",
        "run",
        "server.py"
      ],
      "env": {
        "TERRAMATE_API_KEY": "your_api_key_here",
        "TERRAMATE_LOCATION": "eu"
      }
    }
  }
}
```

#### Step 3: Reload Cursor

1. Reload Cursor window (`Cmd/Ctrl + R`) or restart Cursor
2. Open Cursor Chat (`Cmd/Ctrl + L`)
3. The Terramate tools should now be available
4. Test with: _"List my Terramate organizations"_

#### Troubleshooting Cursor

If tools don't appear:

1. **Check MCP status** in Cursor:
   - Look for MCP indicators in the chat interface
   - Check for error messages in the Cursor output panel

2. **Verify configuration**:
   - Ensure the JSON is valid (no trailing commas)
   - Use absolute paths, not relative paths or `~`
   - Make sure `uv` command is accessible

3. **Test locally first**:
   ```bash
   cd /absolute/path/to/terramate-mcp-server
   uv run server.py
   ```
   This should start without errors.

---

## Configuration Reference

### Required Environment Variables

- `TERRAMATE_API_KEY` - Your Terramate Cloud API key
- `TERRAMATE_LOCATION` - Your Terramate Cloud region (`us` or `eu`)

## Available Tools

### 🎯 Core Tools

Essential tools for monitoring and managing your Terramate Cloud infrastructure:

#### `terramate_organizations`
**Get your organization details and membership**
- Retrieves organization info associated with your API key
- Returns: org UUID (needed for other operations), name, display name, role, and status
- 💡 *Start here to get your org UUID for use in other tools*

#### `terramate_stacks`
**Deep dive into your infrastructure stacks** (Returns up to 100 stacks per page)
- **Operations**: `list`, `get`, `update`
- **What you can do**:
  - 🔍 Search and filter stacks by status (ok, drifted, failed, unknown)
  - 📦 Filter by repository or search for specific stack names
  - 📊 Get comprehensive stack details (metadata, tags, timestamps, paths)
  - 📝 Archive or unarchive stacks
- **Perfect for**: 
  - _"Show me all drifted stacks in the backend repo"_
  - _"What's the current status of stack X?"_
  - _"Find all stacks with 'database' in the name"_
- 💡 *For quick counts only, use `terramate_dashboard` instead*

#### `terramate_deployments`
**Track deployment history and investigate issues**
- **Operations**: `list`, `get`, `logs`
- **What you can do**:
  - 📜 View complete deployment history for any stack
  - 🔎 Get detailed deployment information (status, trigger, duration)
  - 📋 Access deployment logs with stdout/stderr separation
- **Perfect for**: 
  - _"Show me recent failed deployments"_
  - _"What happened in the last deployment of stack X?"_
  - _"Get the error logs from deployment Y"_

#### `terramate_drifts`
**Detect and understand infrastructure drift**
- **Operations**: `list`, `get`, `summarize`
- **What you can do**:
  - 🔍 List all drifts for a stack with status filtering
  - 📊 Get detailed drift information
  - 🤖 Generate AI-powered summaries explaining what drifted
- **Perfect for**: 
  - _"What resources have drifted in stack X?"_
  - _"Explain the drift detected yesterday"_
  - _"Show me all unresolved drifts"_

#### `terramate_alerts`
**Monitor and respond to infrastructure alerts**
- **Operations**: `list`, `get`, `affected_stacks`, `timeline`
- **What you can do**:
  - 🚨 List alerts filtered by severity (critical, error, warning) or status
  - 📍 See which stacks are affected by an alert
  - 📅 View alert timeline and history
- **Perfect for**: 
  - _"Show me all critical alerts"_
  - _"Which stacks are affected by alert X?"_
  - _"What's the timeline of this alert?"_

---

### 🔄 Workflow & Execution Tools

Automate Terramate operations and orchestrate infrastructure workflows:

> ⚠️ **Requires Terramate CLI**: These tools require the Terramate CLI to be installed and available in PATH.

#### `terramate_trigger_workflow`
**Complete automated stack triggering workflow with PR creation**
- **What it does** (end-to-end automation):
  1. 🎯 Triggers stack(s) using Terramate CLI
  2. 🔍 Checks git status for changes
  3. 📁 Stages trigger files (`.tmtriggers`)
  4. 💾 Creates a commit with auto-generated or custom message
  5. 🚀 Creates a draft PR (optional, enabled by default)
- **Triggering options**:
  - Trigger by stack path: `/stacks/prod/database`
  - Trigger by status: `drifted`, `failed`, `ok`, `healthy`, `unhealthy`
  - Recursive mode for nested stacks
- **Git & PR features**:
  - Auto-generates unique branch names: `terramate-trigger-{random}`
  - Customizable commit messages and PR titles
  - Can disable PR creation for local-only commits
  - Requires GitHub CLI (`gh`) for PR creation
- **Perfect for**: 
  - _"Trigger all drifted stacks and create a PR"_
  - _"Trigger deployment for stacks in e.g. /infrastructure/prod"_
  - _"Mark all failed stacks for re-deployment"_
- 💡 *This is a complete workflow tool - it handles the entire process from trigger to PR*

#### `terramate_stack_operations`
**Run Terramate CLI commands across your stacks**
- **Operations**: `list`, `list_changed`, `run`, `run_changed`, `generate`, `fmt`
- **What you can do**:
  - 📋 List all stacks or only changed ones
  - ⚡ Run commands across multiple stacks in parallel
  - 🔄 Generate Terramate code from templates
  - ✨ Format Terramate configuration files
- **Perfect for**: 
  - _"Run terraform plan on all changed stacks"_
  - _"Generate code for all stacks"_
  - _"Format all Terramate files"_

---

### 🔧 Advanced Tools

Specialized tools for power users and complex scenarios:

#### `terramate_resources`
**Manage and monitor individual infrastructure resources**
- **Operations**: `list`, `get`, `deployments`, `drift`
- **What you can do**:
  - 🔍 Filter resources by status, provider, technology, or repository
  - 📊 View deployment history for specific resources
  - 🔄 Check drift status of individual resources
- **Perfect for**: 
  - _"Show me all AWS S3 buckets across stacks"_
  - _"What's the deployment history of resource X?"_
  - _"Find all drifted database resources"_

#### `terramate_review_requests`
**Track and manage infrastructure review requests**
- **Operations**: `list`, `get`
- **What you can do**:
  - 📋 List review requests with status and repository filters
  - 👀 Get detailed review request information
  - Pagination support for large teams
- **Perfect for**: 
  - _"Show me pending review requests"_
  - _"What's the status of review request X?"_

#### `terramate_logs`
**Advanced log management and AI-powered insights**
- **Operations**: `get`, `send`, `summarize`
- **What you can do**:
  - 📋 Retrieve logs with channel filtering (stdout/stderr)
  - 📤 Send log lines to Terramate Cloud
  - 🤖 Generate AI-powered log summaries
- **Perfect for**: 
  - _"Summarize the errors in this deployment"_
  - _"Send these log lines to Terramate Cloud"_

---

### 🔍 Utility Tools

#### `terramate_dashboard`
**Get instant infrastructure health metrics**
- Quick stack counts aggregated by status (ok, drifted, failed, unknown)
- No pagination - perfect for overview questions
- **Perfect for**: 
  - _"How many stacks do I have?"_
  - _"How many stacks are currently drifted?"_
  - _"Give me a quick health overview"_
- 💡 *For detailed stack info, use `terramate_stacks` instead*

#### `discover_tools`
**Explore available capabilities**
- Browse all available Terramate tools
- Filter by category (core, workflow, advanced)
- Get detailed tool descriptions
- **Perfect for**: 
  - _"What tools are available?"_
  - _"Show me all workflow tools"_

### Tool Availability

**API-Based Tools** (always available with just an API key):
- Core tools: `terramate_organizations`, `terramate_stacks`, `terramate_deployments`, `terramate_drifts`, `terramate_alerts`, `terramate_dashboard`
- Advanced tools: `terramate_resources`, `terramate_review_requests`, `terramate_logs`
- Utility: `discover_tools`

**CLI-Dependent Tools** (require Terramate CLI installed):
- Workflow tools: `terramate_trigger_workflow`, `terramate_stack_operations`

> 💡 **Note**: Generic file system and shell operations are not included. Use Cursor's built-in capabilities or dedicated MCP servers (like [filesystem server](https://github.com/modelcontextprotocol/servers)) for those needs.
  
## TODO

- Support remote deployment and allow users to copy the correct endpoint depending on organization and location 
- Make sure to implement [Streamable HTTP](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#streamable-http) for remote deployment support
