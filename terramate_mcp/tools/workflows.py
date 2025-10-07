from typing import Optional, Literal
from pathlib import Path
import shlex
import uuid
from terramate_mcp.mcp_instance import mcp
from terramate_mcp.utils.commands import run_command, format_command_result


@mcp.tool()
async def terramate_trigger_workflow(
    stack_path: str,
    working_directory: Optional[str] = None,
    status: Optional[str] = None,
    recursive: bool = False,
    ignore_change: bool = False,
    commit_message: Optional[str] = None,
    create_pr: bool = True,
    pr_title: Optional[str] = None,
) -> str:
    """
    Complete Terramate stack triggering workflow with PR creation.

    This tool implements the complete Terramate trigger workflow:
    1. Run terramate trigger command
    2. Add triggered files to git
    3. Commit the changes
    4. Optionally create a draft pull request

    Args:
        stack_path: Path to the stack to trigger (or use status filter)
        working_directory: Directory containing Terramate project
        status: Trigger stacks by status ('ok', 'failed', 'drifted', 'unhealthy', 'healthy')
        recursive: Recursively trigger all nested stacks
        ignore_change: Mark stack as unchanged (ignore trigger)
        commit_message: Custom commit message (auto-generated if not provided)
        create_pr: Whether to create a draft pull request
        pr_title: Custom PR title (auto-generated if not provided)

    Returns:
        Complete workflow execution result
    """
    results = []

    try:
        # Step 1: Build and execute terramate trigger command
        cmd = "terramate trigger"

        if status:
            cmd += f" --status={status}"
        else:
            cmd += f" {shlex.quote(stack_path)}"

        if recursive:
            cmd += " --recursive"
        if ignore_change:
            cmd += " --ignore-change"

        results.append("🎯 Step 1: Triggering Terramate stack(s)")
        trigger_result = await run_command(cmd, cwd=working_directory)
        results.append(format_command_result(trigger_result))

        if not trigger_result["success"]:
            return "\n".join(results) + "\n❌ Trigger failed, stopping workflow."

        # Step 2: Check git status to see what files were created
        results.append("\n🔍 Step 2: Checking git status")
        git_status_result = await run_command(
            "git status --porcelain", cwd=working_directory
        )
        results.append(format_command_result(git_status_result))

        if not git_status_result["success"]:
            return "\n".join(results) + "\n❌ Git status check failed."

        # Step 3: Add trigger files to git
        results.append("\n📁 Step 3: Adding trigger files to git")
        git_add_result = await run_command("git add .", cwd=working_directory)
        results.append(format_command_result(git_add_result))

        if not git_add_result["success"]:
            return "\n".join(results) + "\n❌ Git add failed."

        # Step 4: Create commit message
        if not commit_message:
            if status:
                commit_message = f"chore: trigger stacks with status {status}"
            else:
                stack_name = Path(stack_path).name if stack_path else "stack"
                commit_message = f"chore: trigger stack {stack_name}"

        # Step 5: Commit the changes
        results.append(
            f"\n💾 Step 4: Committing changes with message: '{commit_message}'"
        )
        commit_cmd = f"git commit -m {shlex.quote(commit_message)}"
        commit_result = await run_command(commit_cmd, cwd=working_directory)
        results.append(format_command_result(commit_result))

        if not commit_result["success"]:
            return "\n".join(results) + "\n❌ Git commit failed."

        # Step 6: Create pull request (if requested)
        if create_pr:
            results.append("\n🚀 Step 5: Creating draft pull request")

            # Generate branch name
            branch_name = f"terramate-trigger-{uuid.uuid4().hex[:8]}"

            # Create and checkout new branch
            branch_result = await run_command(
                f"git checkout -b {branch_name}", cwd=working_directory
            )
            results.append(format_command_result(branch_result))

            if branch_result["success"]:
                # Push branch
                push_result = await run_command(
                    f"git push -u origin {branch_name}", cwd=working_directory
                )
                results.append(format_command_result(push_result))

                if push_result["success"]:
                    # Create PR using GitHub CLI (if available)
                    if not pr_title:
                        pr_title = commit_message.replace("chore: ", "").title()

                    pr_cmd = f"gh pr create --draft --title {shlex.quote(pr_title)} --body {shlex.quote(f'Automated Terramate stack trigger.\\n\\nTrigger command: {cmd}')}"
                    pr_result = await run_command(pr_cmd, cwd=working_directory)

                    if pr_result["success"]:
                        results.append(format_command_result(pr_result))
                        results.append("\n✅ Draft pull request created successfully!")
                    else:
                        results.append(format_command_result(pr_result))
                        results.append(
                            "\n⚠️ PR creation failed. You may need to install GitHub CLI or check permissions."
                        )
                else:
                    results.append("\n❌ Failed to push branch for PR creation.")
            else:
                results.append("\n❌ Failed to create branch for PR.")

        results.append("\n🎉 Terramate trigger workflow completed successfully!")
        return "\n".join(results)

    except Exception as e:
        results.append(f"\n❌ Workflow error: {str(e)}")
        return "\n".join(results)


@mcp.tool()
async def terramate_stack_operations(
    operation: Literal[
        "list", "list_changed", "run", "run_changed", "generate", "fmt"
    ] = "list",
    working_directory: Optional[str] = None,
    command: Optional[str] = None,
    changed: bool = False,
    parallel: int = 1,
    check: bool = False,
) -> str:
    """
    Unified Terramate CLI stack operations (list, run, generate, format).

    Args:
        operation: Operation to perform
        working_directory: Directory containing Terramate project
        command: Command to run (required for 'run' and 'run_changed' operations)
        changed: Only operate on changed stacks
        parallel: Number of parallel executions for run operations
        check: Only check formatting (for 'fmt' operation)

    Returns:
        Formatted command execution result
    """
    if operation == "list":
        cmd = "terramate list"
        if changed:
            cmd += " --changed"
    elif operation == "list_changed":
        cmd = "terramate list --changed"
    elif operation == "run":
        if not command:
            raise ValueError("command parameter is required for 'run' operation")
        cmd = f"terramate run"
        if changed:
            cmd += " --changed"
        if parallel > 1:
            cmd += f" --parallel {parallel}"
        cmd += f" -- {command}"
    elif operation == "run_changed":
        if not command:
            raise ValueError(
                "command parameter is required for 'run_changed' operation"
            )
        cmd = f"terramate run --changed"
        if parallel > 1:
            cmd += f" --parallel {parallel}"
        cmd += f" -- {command}"
    elif operation == "generate":
        cmd = "terramate generate"
        if changed:
            cmd += " --changed"
    elif operation == "fmt":
        cmd = "terramate fmt"
        if check:
            cmd += " --check"
    else:
        raise ValueError(f"Unknown operation: {operation}")

    timeout = 300 if operation.startswith("run") else 60
    result = await run_command(cmd, cwd=working_directory, timeout=timeout)
    return format_command_result(result)
