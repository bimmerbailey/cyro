"""
Code analysis tools for Cyro agents.

This module provides secure code execution and analysis capabilities using:
1. PydanticAI MCP Run Python server (WebAssembly sandboxed execution)
2. Custom code analysis utilities
3. Security-focused code execution with complete host isolation

MCP provides secure, sandboxed Python execution in WebAssembly environment.
"""

import json
import re
import time
from typing import List, Optional

from pydantic import BaseModel, Field
from pydantic_ai.mcp import MCPServerStdio
from pydantic_ai.toolsets import FunctionToolset

from cyro.config.settings import CyroConfig


class CodeExecutionRequest(BaseModel):
    """Request model for MCP-based code execution."""

    code: str = Field(description="Python code to execute in sandboxed environment")
    dependencies: Optional[List[str]] = Field(
        default=None, description="Optional list of required packages"
    )
    timeout: int = Field(default=30, description="Execution timeout in seconds")
    include_metadata: bool = Field(
        default=False, description="Include PEP 723 dependency metadata"
    )


class CodeExecutionResult(BaseModel):
    """Result model for MCP-based code execution."""

    output: str = Field(description="Standard output from execution")
    error: Optional[str] = Field(
        default=None, description="Error message if execution failed"
    )
    return_value: Optional[str] = Field(
        default=None, description="Return value from code execution"
    )
    success: bool = Field(description="Whether execution succeeded")
    execution_time: float = Field(description="Execution time in seconds")
    sandboxed: bool = Field(default=True, description="Whether execution was sandboxed")
    dependencies_installed: Optional[List[str]] = Field(
        default=None, description="Dependencies that were automatically installed"
    )


class CodeTools:
    """Code analysis and execution tools using PydanticAI MCP for secure sandboxed execution."""

    def __init__(
        self,
        config: Optional[CyroConfig] = None,
        mcp_server: Optional[MCPServerStdio] = None,
    ):
        """Initialize code tools.

        Args:
            config: Cyro configuration
            mcp_server: Optional MCP server instance (for testing or custom setups)
        """
        self.config = config or CyroConfig()

        # Initialize MCP server - either provided or create default
        self.mcp_server = mcp_server or MCPServerStdio(
            "deno",  # Use system deno executable
            args=[
                "run",
                "-N",
                "-R=node_modules",
                "-W=node_modules",
                "--node-modules-dir=auto",
                "jsr:@pydantic/mcp-run-python",
                "stdio",
            ],
            tool_prefix="mcp_",  # Simple default prefix
        )

    async def execute_code(self, request: CodeExecutionRequest) -> CodeExecutionResult:
        """Execute Python code using MCP Run Python server in sandboxed environment."""
        start_time = time.time()

        try:
            # Prepare code with optional PEP 723 metadata for dependencies
            code_to_execute = request.code
            if request.dependencies and request.include_metadata:
                # Add PEP 723 metadata comment
                deps_comment = (
                    f"# /// script\n# dependencies = {request.dependencies}\n# ///\n"
                )
                code_to_execute = deps_comment + code_to_execute

            # Execute code through MCP server
            async with self.mcp_server:
                result_str = await self.mcp_server.direct_call_tool(
                    "run_python_code", {"python_code": code_to_execute}
                )

            execution_time = time.time() - start_time

            # Parse MCP result XML-like format
            success = "<status>success</status>" in result_str
            output = CodeTools._extract_xml_content(result_str, "output")
            error = CodeTools._extract_xml_content(result_str, "error")
            return_value = CodeTools._extract_xml_content(result_str, "return_value")
            dependencies_str = CodeTools._extract_xml_content(
                result_str, "dependencies"
            )

            # Parse dependencies JSON if present
            dependencies_installed = None
            if dependencies_str:
                try:
                    dependencies_installed = json.loads(dependencies_str)
                except json.JSONDecodeError:
                    dependencies_installed = None

            return CodeExecutionResult(
                output=output or "",
                error=error,
                return_value=return_value,
                success=success,
                execution_time=execution_time,
                sandboxed=True,
                dependencies_installed=dependencies_installed,
            )

        except Exception as e:
            execution_time = time.time() - start_time

            return CodeExecutionResult(
                output="",
                error=str(e),
                return_value=None,
                success=False,
                execution_time=execution_time,
                sandboxed=True,
                dependencies_installed=None,
            )

    @staticmethod
    def _extract_xml_content(xml_str: str, tag: str) -> Optional[str]:
        """Extract content from XML-like tags in MCP response."""
        pattern = f"<{tag}>(.*?)</{tag}>"
        match = re.search(pattern, xml_str, re.DOTALL)
        if match:
            return match.group(1).strip()
        return None


def create_code_toolset(
    config: Optional[CyroConfig] = None, mcp_server: Optional[MCPServerStdio] = None
) -> FunctionToolset:
    """Create a secure code analysis toolset using PydanticAI MCP for sandboxed execution.

    Args:
        config: Cyro configuration
        mcp_server: Optional MCP server instance (for testing or custom setups)

    Returns:
        FunctionToolset containing secure MCP-based code analysis tools
    """
    # Use centralized operations to eliminate code duplication
    code_tools = CodeTools(config, mcp_server)

    # Create FunctionToolset for all code operations
    toolset = FunctionToolset()

    @toolset.tool
    async def execute_python_code_mcp(
        request: CodeExecutionRequest,
    ) -> CodeExecutionResult:
        """Execute Python code in secure MCP sandboxed environment with complete host isolation."""
        return await code_tools.execute_code(request)

    @toolset.tool
    async def execute_python_with_dependencies(
        code: str, dependencies: List[str]
    ) -> str:
        """Execute Python code with automatic dependency installation in sandboxed environment."""
        request = CodeExecutionRequest(
            code=code, dependencies=dependencies, include_metadata=True
        )
        result = await code_tools.execute_code(request)
        if result.success:
            return f"Output: {result.output}"
        else:
            return f"Error: {result.error}"

    return toolset
