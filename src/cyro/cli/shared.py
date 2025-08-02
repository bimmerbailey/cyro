"""Shared utilities for CLI modules."""

from dataclasses import dataclass
from pathlib import Path
from typing import Literal

from cyro.agents.manager import ManagerAgent


@dataclass
class ChatCommandResult:
    """Result of processing a chat command."""

    action: Literal[
        "exit",
        "clear",
        "help",
        "agent_switch",
        "history",
        "status",
        "config",
        "unknown",
        "error",
    ]
    value: str | None = None
    error_message: str | None = None


def process_agent_request(
    message: str, manager_agent: ManagerAgent, agent: str | None = None
) -> str:
    """Process a request through either a specific agent or the manager.

    Args:
        message: The user's message/query
        manager_agent: The manager agent instance
        agent: Optional specific agent name to use

    Returns:
        Response text from the selected agent

    Raises:
        Exception: If agent execution fails
    """
    if agent:
        # Use explicitly requested agent
        selected_agent = manager_agent.get_agent_by_name(agent)
        response = selected_agent.run_sync(message)
        return response.output
    else:
        # Route through manager
        return manager_agent.process_request(message)


# TODO: Should theme_manager be nullable?
def get_themed_color(semantic_name: str, theme_manager) -> str:
    """Get a themed color with fallback to global theme.

    Args:
        semantic_name: The semantic color name (e.g., 'primary', 'success')
        theme_manager: Optional theme manager instance

    Returns:
        Color string for Rich styling
    """
    from cyro.config.themes import get_theme_color

    if theme_manager is None:
        return get_theme_color(semantic_name)
    return theme_manager.get_color(semantic_name)


def get_config_directory() -> Path:
    """Get the configuration directory, preferring project-local over global.

    Returns:
        Path to the configuration directory (.cyro in current directory
        or ~/.cyro as fallback)
    """
    project_config_dir = Path(".cyro")
    if project_config_dir.exists():
        return project_config_dir
    else:
        return Path("~/.cyro").expanduser()


def get_themes_directory() -> str:
    """Get the themes directory path as string.

    Returns:
        String path to themes directory
    """
    return "~/.cyro/themes"
