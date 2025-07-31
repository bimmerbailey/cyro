"""
Manager agent for routing tasks and coordinating subagents.

This module provides the ManagerAgent class that acts as a central coordinator
for all subagents, handling task routing, agent selection, and delegation.
"""

from typing import Dict, List, Optional
from pathlib import Path

from pydantic import BaseModel, UUID4

from cyro.agents.base import CyroAgent, AgentRegistry, AgentMetadata, AgentConfig
from cyro.config.settings import CyroConfig


class TaskRoutingRequest(BaseModel):
    """Request for task routing to appropriate agent."""

    task_description: str
    user_prompt: str
    explicit_agent: Optional[str] = None


class AgentSelection(BaseModel):
    """Response containing selected agent and reasoning."""

    agent_name: str
    agent_id: UUID4
    reasoning: str


class ManagerAgent(CyroAgent):
    """Central manager for routing tasks to appropriate subagents."""

    config_dir: Path
    registry: AgentRegistry

    def __init__(self, config: CyroConfig, config_dir: Path | None = None):
        """Initialize the manager agent.

        Args:
            config: Cyro configuration instance
            config_dir: Directory containing agent configuration files
        """
        # Create manager agent configuration
        manager_config = AgentConfig.from_file()

        # Initialize parent CyroAgent
        super().__init__(manager_config, config.provider)
        # Manager-specific attributes
        default_config = Path("~/.cyro")

        self.config = config_dir or default_config
        self.registry = AgentRegistry()
