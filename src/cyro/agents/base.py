"""
Base agent classes and models for Cyro.

This module provides the foundation for the agent system, including
the base CyroAgent class that extends PydanticAI Agent and core
metadata structures.
"""

from dataclasses import dataclass
from typing import Dict, Any, Type, Iterator
from uuid import uuid4
from pathlib import Path
import re

from pydantic import BaseModel, Field
from pydantic.types import UUID4
from pydantic_ai import Agent

from cyro.config.settings import CyroConfig


class AgentMetadata(BaseModel):
    """Structured metadata for agents."""

    name: str
    description: str
    version: str = Field(default="1.0", pattern=r"^[1-9]\.[0-9]$")


@dataclass
class AgentConfig:
    """Parsed agent configuration from markdown files."""

    metadata: AgentMetadata
    system_prompt: str
    instructions: str | None = None
    # TODO: come back to this
    tools: Any | None = None
    result_type: Type[BaseModel] | None = None

    @classmethod
    def from_markdown(
        cls, content: str | bytes, result_type: Type[BaseModel] | type = str
    ) -> "AgentConfig":
        """Create AgentConfig from markdown content.

        Args:
            content: Markdown content as string or bytes
            result_type: Optional result type class that must be a subclass of BaseModel

        Returns:
            AgentConfig instance

        Raises:
            ValueError: If the content format is invalid
        """
        if isinstance(content, bytes):
            content = content.decode("utf-8")

        # Parse YAML frontmatter
        frontmatter_match = re.match(r"^---\n(.*?)\n---\n(.*)", content, re.DOTALL)
        if not frontmatter_match:
            raise ValueError("Invalid agent format: missing YAML frontmatter")

        frontmatter_text, system_prompt = frontmatter_match.groups()
        system_prompt = system_prompt.strip()

        # Parse frontmatter fields
        config_data = {}
        for line in frontmatter_text.strip().split("\n"):
            if ":" in line:
                key, value = line.split(":", 1)
                key = key.strip()
                value = value.strip()

                if key == "tools" and value:
                    # Parse comma-separated tools list
                    config_data[key] = [tool.strip() for tool in value.split(",")]
                else:
                    config_data[key] = value

        # Validate required fields
        if "name" not in config_data:
            raise ValueError("Agent missing required 'name' field")
        if "description" not in config_data:
            raise ValueError("Agent missing required 'description' field")

        # Create metadata
        metadata = AgentMetadata(
            name=config_data["name"],
            description=config_data["description"],
            version=config_data.get("version", "1.0"),
        )

        return cls(
            metadata=metadata,
            system_prompt=system_prompt,
            tools=config_data.get("tools"),
            result_type=result_type,
        )

    @classmethod
    def from_file(
        cls, file_path: Path, result_type: Type[BaseModel] | type = str
    ) -> "AgentConfig":
        """Create AgentConfig from a markdown file.

        Args:
            file_path: Path to the markdown file
            result_type: Optional result type class that must be a subclass of BaseModel

        Returns:
            AgentConfig instance

        Raises:
            FileNotFoundError: If the file doesn't exist
            ValueError: If the file format is invalid
        """
        if not file_path.exists():
            raise FileNotFoundError(f"Agent file not found: {file_path}")

        content = file_path.read_text(encoding="utf-8")
        return cls.from_markdown(content, result_type)


class CyroAgent:
    """Cyro agent wrapper that contains a PydanticAI Agent instance."""

    id: UUID4
    metadata: AgentMetadata
    config: AgentConfig
    agent: Agent

    def __init__(self, config: AgentConfig, model: Any):
        """Initialize CyroAgent with configuration and model.

        Args:
            config: Agent configuration from markdown parsing
            model: PydanticAI model instance (from CyroConfig.provider)
        """
        self.id = uuid4()
        self.metadata = config.metadata
        self.config = config
        self.agent = Agent(
            model,
            system_prompt=config.system_prompt,
            instructions=config.instructions,
            output_type=config.result_type,
        )

    def run_sync(self, prompt: str, output_type: Type[BaseModel] | None = None):
        """Delegate to the underlying PydanticAI Agent."""
        return self.agent.run_sync(user_prompt=prompt, output_type=output_type)

    async def run(self, prompt: str, output_type: Type[BaseModel] | None = None):
        """Delegate to the underlying PydanticAI Agent."""
        return await self.agent.run(user_prompt=prompt, output_type=output_type)


class AgentRegistry:
    """Registry for storing and retrieving agents."""

    agents: Dict[UUID4, CyroAgent]

    def __init__(self):
        """Initialize empty agent registry."""
        self.agents: Dict[UUID4, CyroAgent] = {}

    def add(self, agent: CyroAgent) -> None:
        """Add an agent to the registry.

        Args:
            agent: CyroAgent instance to add
        """
        if agent.id not in self.agents:
            self.agents[agent.id] = agent

    def get_by_id(self, agent_id: UUID4) -> CyroAgent:
        """Get agent by UUID4.

        Args:
            agent_id: Agent UUID4

        Returns:
            CyroAgent instance

        Raises:
            KeyError: If agent with given ID is not found
        """
        if agent_id not in self.agents:
            raise KeyError(f"Agent with ID {agent_id} not found in registry")
        return self.agents[agent_id]

    def get_by_name(self, name: str) -> CyroAgent:
        """Get agent by name (case-insensitive).

        Args:
            name: Agent name string

        Returns:
            CyroAgent instance

        Raises:
            KeyError: If agent with given name is not found
        """
        for agent in self.agents.values():
            if agent.metadata.name.lower() == name.lower():
                return agent
        raise KeyError(f"Agent with name '{name}' not found in registry")

    def __iter__(self) -> Iterator[CyroAgent]:
        """Iterate over agents."""
        return iter(self.agents.values())


def make_general_agent(settings: CyroConfig = CyroConfig()) -> CyroAgent:
    """Make general agent instance."""

    # TODO: All tools available here
    metadata = AgentMetadata(
        name="general-engineer",
        description="General-purpose software engineering agent for coding tasks, "
        "debugging, refactoring, and technical problem-solving."
        "Not an expert in any one task but has knowledge of most things.",
        version="1.0",
    )

    config = AgentConfig(
        metadata=metadata,
        system_prompt="You are a helpful assistant",
    )
    return CyroAgent(config, settings.provider)
