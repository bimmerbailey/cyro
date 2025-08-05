"""
Tool factory system for creating agent toolsets.

This module provides a centralized way to create and manage tool combinations
for different agent types and use cases.
"""

from typing import Callable, List, Optional

from pydantic_ai.toolsets import FunctionToolset

from cyro.config.settings import CyroConfig
from cyro.tools.execution import create_execution_toolset
from cyro.tools.filesystem import create_filesystem_toolset
from cyro.tools.task_management import create_task_management_toolset
from cyro.tools.web import create_web_toolset

# Import code and git tools conditionally
try:
    from cyro.tools.code import create_code_toolset

    CODE_TOOLS_AVAILABLE = True
except ImportError:
    print("Warning: Code tools not available due to missing dependencies")
    CODE_TOOLS_AVAILABLE = False

try:
    from cyro.tools.git import create_git_toolset

    GIT_TOOLS_AVAILABLE = True
except ImportError:
    print("Warning: Git tools not available due to missing dependencies")
    GIT_TOOLS_AVAILABLE = False


ToolSetCreator = Callable[[CyroConfig], FunctionToolset]


# TODO: This seems complicated
def create_agent_toolset(
    tools: List[str], config: Optional[CyroConfig] = None
) -> FunctionToolset:
    """Create a combined toolset for an agent based on tool names.

    Args:
        tools: List of tool category names to include
        config: Cyro configuration (optional)

    Returns:
        FunctionToolset containing all requested tools

    Raises:
        ValueError: If an unknown tool name is requested
        ImportError: If a requested tool module is not available
    """
    if config is None:
        config = CyroConfig()

    # Available tool factories
    tool_factories: dict[str, ToolSetCreator] = {
        "filesystem": create_filesystem_toolset,
        "execution": create_execution_toolset,
        "web": create_web_toolset,
        "task_management": create_task_management_toolset,
    }

    # TODO: Make these match the types from above
    # Conditionally available tools
    if CODE_TOOLS_AVAILABLE:
        tool_factories["code"] = create_code_toolset

    if GIT_TOOLS_AVAILABLE:
        tool_factories["git"] = create_git_toolset

    # Handle empty tool list (e.g., manager agent)
    if len(tools) == 0:
        return FunctionToolset()  # Return empty toolset

    # For single tool category, return the toolset directly
    if len(tools) == 1:
        tool_name = tools[0]
        if tool_name not in tool_factories:
            available_tools = list(tool_factories.keys())
            raise ValueError(
                f"Unknown tool '{tool_name}'. Available tools: {available_tools}"
            )

        tool_factory = tool_factories[tool_name]
        return tool_factory(config)

    # For multiple tool categories, combine them
    master_toolset = FunctionToolset()

    for tool_name in tools:
        if tool_name not in tool_factories:
            available_tools = list(tool_factories.keys())
            raise ValueError(
                f"Unknown tool '{tool_name}'. Available tools: {available_tools}"
            )

        # Get the toolset for this tool category
        tool_factory = tool_factories[tool_name]
        category_toolset = tool_factory(config)

        # Add all tools from this category to the master toolset
        for tool in category_toolset.tools.values():
            master_toolset.add_tool(tool)

    return master_toolset


def get_toolset_for_agent_type(
    agent_type: str, config: Optional[CyroConfig] = None
) -> FunctionToolset:
    """Get a predefined toolset for common agent types.

    Args:
        agent_type: Type of agent (e.g., 'general', 'web', 'coding', 'debug')
        config: Cyro configuration (optional)

    Returns:
        FunctionToolset appropriate for the agent type

    Raises:
        ValueError: If unknown agent type is requested
    """
    # Predefined toolset combinations for common agent types
    toolset_mappings = {
        "general": ["filesystem", "execution", "web", "task_management"],
        "web": ["web", "task_management"],
        "coding": ["filesystem", "execution", "code", "git", "task_management"],
        "debug": ["filesystem", "execution", "git", "task_management"],
        "file": ["filesystem", "task_management"],
        "search": ["web"],
        "manager": [],  # Manager only routes - no tools needed
    }

    # Add code and git to appropriate categories if available
    if CODE_TOOLS_AVAILABLE:
        if "code" not in toolset_mappings["general"]:
            toolset_mappings["general"].append("code")

    if GIT_TOOLS_AVAILABLE:
        if "git" not in toolset_mappings["general"]:
            toolset_mappings["general"].append("git")
        if "git" not in toolset_mappings["debug"]:
            toolset_mappings["debug"].append("git")

    if agent_type not in toolset_mappings:
        available_types = list(toolset_mappings.keys())
        raise ValueError(
            f"Unknown agent type '{agent_type}'. Available types: {available_types}"
        )

    tool_list = toolset_mappings[agent_type]
    return create_agent_toolset(tool_list, config)


def list_available_tools() -> List[str]:
    """List all available tool categories.

    Returns:
        List of available tool category names
    """
    base_tools = ["filesystem", "execution", "web", "task_management"]

    optional_tools = []
    if CODE_TOOLS_AVAILABLE:
        optional_tools.append("code")
    if GIT_TOOLS_AVAILABLE:
        optional_tools.append("git")

    return base_tools + optional_tools


def list_available_agent_types() -> List[str]:
    """List all predefined agent types.

    Returns:
        List of predefined agent type names
    """
    return ["general", "web", "coding", "debug", "file", "search", "manager"]


def validate_tool_list(tools: List[str]) -> List[str]:
    """Validate a list of tool names and return any invalid ones.

    Args:
        tools: List of tool names to validate

    Returns:
        List of invalid tool names (empty if all valid)
    """
    available_tools = list_available_tools()
    return [tool for tool in tools if tool not in available_tools]
