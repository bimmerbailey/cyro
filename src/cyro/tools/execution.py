"""
Shell execution tools for Cyro agents.

This module provides secure shell execution capabilities combining:
1. LangChain ShellTool with human approval for basic commands
2. Custom PydanticAI tools for enhanced security and timeout controls

Security: All shell commands include safety controls, timeouts, and approval workflows.
"""

import subprocess
import time
from pathlib import Path
from typing import Any, Optional

from pydantic import BaseModel, Field
from pydantic_ai import Agent
from pydantic_ai.ext.langchain import LangChainToolset

from cyro.config.settings import CyroConfig


class ShellCommandRequest(BaseModel):
    """Request model for shell command execution."""

    command: str = Field(description="Shell command to execute")
    timeout: int = Field(default=30, description="Timeout in seconds")
    working_dir: Optional[str] = Field(
        default=None, description="Working directory for command execution"
    )
    allow_dangerous: bool = Field(
        default=False, description="Allow potentially dangerous commands"
    )


class ShellCommandResult(BaseModel):
    """Result model for shell command execution."""

    command: str = Field(description="The command that was executed")
    return_code: int = Field(description="Return code from command execution")
    stdout: str = Field(description="Standard output")
    stderr: str = Field(description="Standard error")
    execution_time: float = Field(description="Time taken to execute in seconds")
    timeout_occurred: bool = Field(
        default=False, description="Whether command timed out"
    )


class ExecutionTools:
    """
    Shell execution tools combining LangChain and custom PydanticAI tools.

    Provides secure command execution with configurable safety controls.
    """

    DANGEROUS_COMMANDS = {
        # System modification
        "rm",
        "rmdir",
        "mv",
        "cp",
        "chmod",
        "chown",
        "sudo",
        "su",
        # Network/Security
        "curl",
        "wget",
        "ssh",
        "scp",
        "nc",
        "netcat",
        # Process management
        "kill",
        "killall",
        "pkill",
        # System info that could be sensitive
        "ps",
        "netstat",
        "lsof",
        # Package management
        "apt",
        "yum",
        "pip",
        "npm",
        "brew",
    }

    def __init__(
        self,
        config: Optional[CyroConfig] = None,
        enable_human_approval: bool = True,
    ):
        """
        Initialize execution tools using current working directory.

        Args:
            config: Cyro configuration for security settings.
            enable_human_approval: Enable human approval for dangerous commands.
        """
        self.config = config or CyroConfig()

        # Set up secure working directory
        self.working_dir = Path.cwd()  # Always use current working directory

        # Ensure working directory exists (only create if it doesn't exist and it's not cwd)
        if not self.working_dir.exists():
            self.working_dir.mkdir(parents=True, exist_ok=True)

        # We don't need LangChain's ShellTool - our custom tools are better
        # Create empty LangChain toolset (custom tools will handle everything)
        self.langchain_toolset = LangChainToolset([])  # type: ignore

        self.enable_human_approval = enable_human_approval

    def _is_dangerous_command(self, command: str) -> bool:
        """
        Check if a command contains dangerous elements.

        Args:
            command: Command string to check

        Returns:
            True if command is potentially dangerous
        """
        # Extract the base command (first word)
        base_command = command.strip().split()[0] if command.strip() else ""

        # Check against dangerous commands list
        if base_command in self.DANGEROUS_COMMANDS:
            return True

        # Check for dangerous patterns
        dangerous_patterns = [
            "|",  # Pipes can be dangerous
            ">",  # Redirects can overwrite files
            ">>",  # Appends can modify files
            "&",  # Background execution
            ";",  # Command chaining
            "$(",  # Command substitution
            "`",  # Command substitution (backticks)
            "&&",  # Conditional execution
            "||",  # Conditional execution
        ]

        return any(pattern in command for pattern in dangerous_patterns)

    def _validate_working_dir(self, working_dir: Optional[str]) -> Path:
        """
        Validate and resolve working directory.

        Args:
            working_dir: Working directory path

        Returns:
            Resolved Path object

        Raises:
            ValueError: If directory is invalid or outside allowed paths
        """
        if working_dir is None:
            return self.working_dir

        path = Path(working_dir)
        if not path.is_absolute():
            path = self.working_dir / path

        resolved_path = path.resolve()

        # Security check: ensure path exists and is a directory
        if not resolved_path.exists():
            raise ValueError(f"Working directory does not exist: {working_dir}")

        if not resolved_path.is_dir():
            raise ValueError(f"Path is not a directory: {working_dir}")

        return resolved_path

    def _get_user_approval(self, command: str) -> bool:
        """
        Get user approval for command execution.

        Args:
            command: Command to ask approval for

        Returns:
            True if user approves, False otherwise
        """
        if not self.enable_human_approval:
            return True

        print("\n🔐 Security Check: About to execute command:")
        print(f"Command: {command}")
        print(f"Working Directory: {self.working_dir}")

        while True:
            response = input("Do you want to proceed? (y/n): ").lower().strip()
            if response in ["y", "yes"]:
                return True
            elif response in ["n", "no"]:
                return False
            else:
                print("Please respond with 'y' or 'n'")

    def get_langchain_toolset(self) -> LangChainToolset:
        """Get the LangChain toolset for PydanticAI integration."""
        return self.langchain_toolset

    def create_custom_tools(self, agent: Agent) -> None:
        """
        Add custom execution tools to a PydanticAI agent.

        Args:
            agent: PydanticAI agent to add tools to
        """

        @agent.tool_plain
        def execute_shell_command(request: ShellCommandRequest) -> str:
            """
            Execute a shell command with security controls and timeout.

            Args:
                request: Shell command execution request

            Returns:
                Formatted execution result or error message
            """
            try:
                # Validate working directory
                work_dir = self._validate_working_dir(request.working_dir)

                # Security check for dangerous commands
                is_dangerous = self._is_dangerous_command(request.command)

                if is_dangerous and not request.allow_dangerous:
                    return f"🚫 Command blocked: '{request.command}' is potentially dangerous. Use allow_dangerous=True to override."

                # Human approval check
                if is_dangerous or self.enable_human_approval:
                    if not self._get_user_approval(request.command):
                        return (
                            f"❌ Command execution cancelled by user: {request.command}"
                        )

                # Execute command with timeout
                start_time = time.time()
                try:
                    result = subprocess.run(
                        request.command,
                        shell=True,
                        cwd=str(work_dir),
                        capture_output=True,
                        text=True,
                        timeout=request.timeout,
                    )
                    execution_time = time.time() - start_time

                    # Create result object
                    cmd_result = ShellCommandResult(
                        command=request.command,
                        return_code=result.returncode,
                        stdout=result.stdout,
                        stderr=result.stderr,
                        execution_time=execution_time,
                        timeout_occurred=False,
                    )

                except subprocess.TimeoutExpired:
                    execution_time = time.time() - start_time
                    cmd_result = ShellCommandResult(
                        command=request.command,
                        return_code=-1,
                        stdout="",
                        stderr=f"Command timed out after {request.timeout} seconds",
                        execution_time=execution_time,
                        timeout_occurred=True,
                    )

                # Format result
                if cmd_result.return_code == 0:
                    result_msg = f"✅ Command executed successfully (took {cmd_result.execution_time:.2f}s)"
                else:
                    result_msg = (
                        f"❌ Command failed with return code {cmd_result.return_code}"
                    )

                if cmd_result.timeout_occurred:
                    result_msg += f" (TIMEOUT after {request.timeout}s)"

                result_msg += f"\nCommand: {cmd_result.command}"

                if cmd_result.stdout:
                    result_msg += f"\n\nSTDOUT:\n{cmd_result.stdout}"

                if cmd_result.stderr:
                    result_msg += f"\n\nSTDERR:\n{cmd_result.stderr}"

                return result_msg

            except Exception as e:
                return f"💥 Error executing command: {str(e)}"

        @agent.tool_plain
        def execute_safe_command(command: str, timeout: int = 15) -> str:
            """
            Execute a safe shell command (read-only operations only).

            Args:
                command: Shell command to execute (must be read-only)
                timeout: Timeout in seconds

            Returns:
                Command output or error message
            """
            # List of allowed safe commands
            safe_commands = {
                "ls",
                "cat",
                "head",
                "tail",
                "grep",
                "find",
                "which",
                "whoami",
                "pwd",
                "date",
                "echo",
                "wc",
                "sort",
                "uniq",
                "cut",
                "awk",
                "sed",
            }

            base_command = command.strip().split()[0] if command.strip() else ""

            if base_command not in safe_commands:
                return f"🚫 Command '{base_command}' not allowed in safe mode. Allowed commands: {', '.join(sorted(safe_commands))}"

            # Execute with minimal request
            request = ShellCommandRequest(
                command=command, timeout=timeout, allow_dangerous=False
            )

            # Disable human approval for safe commands
            original_approval = self.enable_human_approval
            self.enable_human_approval = False

            try:
                result = execute_shell_command(request)
                return result
            finally:
                self.enable_human_approval = original_approval


def create_execution_toolset(
    config: Optional[CyroConfig] = None,
    enable_human_approval: bool = True,
) -> tuple[LangChainToolset, Any]:
    """
    Create a complete execution toolset for PydanticAI agents using current working directory.

    Args:
        config: Cyro configuration
        enable_human_approval: Enable human approval workflow

    Returns:
        Tuple of (LangChain toolset, custom tools creator function)
    """
    execution_tools = ExecutionTools(
        config=config,
        enable_human_approval=enable_human_approval,
    )

    return (
        execution_tools.get_langchain_toolset(),
        execution_tools.create_custom_tools,
    )


# Convenience function for safe-only execution
def get_safe_execution_tools() -> LangChainToolset:
    """
    Get safe execution tools (read-only commands only) using current working directory.

    Returns:
        LangChain toolset with safe execution capabilities
    """
    # For safe tools, we provide only custom safe tools via empty toolset
    return LangChainToolset([])  # type: ignore  # Empty toolset, custom tools handle everything
