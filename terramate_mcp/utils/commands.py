import asyncio
import shlex
import subprocess
from typing import Any, Optional
import inspect


async def run_command(
    command: str,
    cwd: Optional[str] = None,
    timeout: int = 60,
    capture_output: bool = True,
) -> dict[str, Any]:
    """Execute a CLI command safely and return the result."""
    try:
        if isinstance(command, str):
            cmd_args = shlex.split(command)
        else:
            cmd_args = command

        if not cmd_args or not cmd_args[0]:
            raise ValueError("Invalid command")

        process = await asyncio.create_subprocess_exec(
            *cmd_args,
            stdout=subprocess.PIPE if capture_output else None,
            stderr=subprocess.PIPE if capture_output else None,
            cwd=cwd,
        )

        try:

            async def _ensure_communicate(proc):
                result = proc.communicate()
                if inspect.isawaitable(result):
                    return await result
                return result

            stdout, stderr = await asyncio.wait_for(
                _ensure_communicate(process), timeout=timeout
            )
        except asyncio.TimeoutError:
            process.kill()
            try:
                wait_attr = getattr(process, "wait", None)
                if asyncio.iscoroutinefunction(wait_attr):
                    await process.wait()
            except Exception:
                pass
            raise Exception(f"Command timed out after {timeout} seconds")

        return {
            "success": process.returncode == 0,
            "return_code": process.returncode,
            "stdout": stdout.decode("utf-8") if stdout else "",
            "stderr": stderr.decode("utf-8") if stderr else "",
            "command": command,
            "cwd": cwd,
        }

    except Exception as e:
        return {
            "success": False,
            "return_code": -1,
            "stdout": "",
            "stderr": str(e),
            "command": command,
            "cwd": cwd,
            "error": str(e),
        }


def format_command_result(result: dict) -> str:
    """Format command execution result for human readability."""
    emoji = "✅" if result["success"] else "❌"

    output = f"""{emoji} Command: {result['command']}
Return Code: {result['return_code']}
Working Directory: {result.get('cwd', 'current')}

STDOUT:
{result['stdout'] if result['stdout'] else '(no output)'}
"""

    if result["stderr"]:
        output += f"\nSTDERR:\n{result['stderr']}"

    if "error" in result:
        output += f"\nError: {result['error']}"

    return output
