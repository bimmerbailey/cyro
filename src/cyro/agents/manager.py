"""
Manager agent for routing tasks and coordinating subagents.

This module provides the ManagerAgent class that acts as a central coordinator
for all subagents, handling task routing, agent selection, and delegation.
"""

from pathlib import Path

import structlog.stdlib
from pydantic import BaseModel, Field
from pydantic.types import UUID4

from cyro.agents.base import (
    AgentConfig,
    AgentMetadata,
    AgentRegistry,
    CyroAgent,
    make_general_agent,
)
from cyro.config.settings import CyroConfig

logger = structlog.stdlib.get_logger(__name__)


class AgentSelection(BaseModel):
    """Response containing selected agent and reasoning."""

    recommended_agent: str = Field(
        description="Recommended agent to handle the user request"
    )
    agent_id: UUID4 = Field(description="The ID of the agent that has been selected")
    reasoning: str = Field(description="The reasoning that the agent is selected")


system_prompt = """
You are a Manager Agent responsible for analyzing user requests and routing them to the most appropriate specialized agent. 
Your sole responsibility is intelligent request routing and delegation.

## Core Function
- Analyze incoming user requests to understand intent, domain, and complexity
- Match requests to the most appropriate available agent based on their capabilities  
- Route requests decisively with clear reasoning
- Default to the Software Engineer when no specialized agent is clearly better

## What You Do NOT Do
- Write code, debug issues, or perform technical work
- Provide direct answers to user questions
- Execute commands or manipulate files
- Ask clarifying questions unless critical information is missing

## Response Requirements
You must respond with a structured JSON object matching the AgentSelection schema:

```json
{
    "recommended_agent": "exact_agent_name",
    "agent_id": "uuid4_of_selected_agent",
    "reasoning": "Brief explanation of selection rationale"
}
```

Follow the detailed instructions provided to make optimal routing decisions.
"""


# TODO: Delegation method
class ManagerAgent(CyroAgent):
    """Central manager for routing tasks to appropriate subagents."""

    config_dir: Path
    registry: AgentRegistry

    def __init__(
        self, config: CyroConfig = CyroConfig(), config_dir: Path = Path("~/.cyro")
    ):
        """Initialize the manager agent.

        Args:
            config: Cyro configuration instance
            config_dir: Directory containing cyro configuration files
        """

        # Manager-specific attributes
        self.config_dir = config_dir
        self.registry = AgentRegistry()
        self.load_agents_from_directory(config=config)

        # Log loaded agents
        agent_names = [agent.metadata.name for agent in self.registry]
        logger.info(
            "Manager initialized",
            config_dir=str(self.config_dir),
            agent_count=len(agent_names),
            agents=agent_names,
        )

        metadata = AgentMetadata(
            name="manager",
            description="Responsible for delegating tasks to subagents",
            version="1.0",
        )

        # Build dynamic instructions based on current agent registry
        dynamic_instructions = self.build_agent_instructions()

        manager_config = AgentConfig(
            metadata=metadata,
            system_prompt=system_prompt,
            instructions=dynamic_instructions,
            result_type=AgentSelection,
        )

        # Initialize base CyroAgent
        super().__init__(manager_config, config.provider)

    def build_agent_instructions(self) -> str:
        """Build comprehensive instructions for agent selection based on registry.

        Returns:
            Formatted instructions string with detailed agent capabilities and selection guidance
        """
        instructions = """
You are the manager of this system. You are responsible for delegating tasks to subagents.
You should not do the task asked by the user        

### Available Agents

"""

        # Add registered specialized agents with enhanced information
        for agent in self.registry:
            # TODO: Do we need ID?
            instructions += f"**{agent.metadata.name}** (ID: {agent.id})\n"
            instructions += f"- **Description**: {agent.metadata.description}\n"
            instructions += f"- **Version**: {agent.metadata.version}\n"

            # Add capabilities from config if available
            if agent.config.tools:
                tools_list = (
                    agent.config.tools
                    if isinstance(agent.config.tools, list)
                    else [str(agent.config.tools)]
                )
                instructions += f"- **Available Tools**: {', '.join(tools_list)}\n"

            instructions += "\n"
        instructions += """
## Response Requirements
You must respond with a structured JSON object matching the AgentSelection schema:

```json
{
    "recommended_agent": "Code Reviewer", 
    "agent_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "reasoning": "Based on REVIEW request in Quality domain with MODERATE scope, selected Code Reviewer because specialized in code analysis and quality assurance. Score: 9/10."
}
```
"""
        return instructions

    def add_agent(self, agent: CyroAgent) -> None:
        """Add an agent to the registry and refresh instructions.

        Args:
            agent: CyroAgent instance to add to the registry
        """
        self.registry.add(agent)

    def get_agent_by_name(self, name: str) -> CyroAgent:
        """Get agent by name from the registry.

        Args:
            name: Agent name (case-insensitive)

        Returns:
            CyroAgent instance

        Raises:
            KeyError: If agent not found
        """
        return self.registry.get_by_name(name)

    def get_agent_by_id(self, agent_id: UUID4) -> CyroAgent:
        """Get agent by ID from the registry.

        Args:
            agent_id: Agent UUID4

        Returns:
            CyroAgent instance

        Raises:
            KeyError: If agent not found
        """
        return self.registry.get_by_id(agent_id)

    def add_general_agent(self, config: CyroConfig) -> None:
        general_agent = make_general_agent(settings=config)
        self.add_agent(general_agent)

    def load_agents_from_directory(self, config: CyroConfig = CyroConfig()) -> None:
        """Load all agents from markdown files in the agents directory.

        Args:
            config: Optional CyroConfig instance for creating agents

        Raises:
            FileNotFoundError: If agents directory doesn't exist
            ValueError: If any agent file has invalid format
        """

        agents_dir = self.config_dir / "agents"

        if not agents_dir.exists():
            # If the dir doesn't exist, add general agent and exit
            self.add_general_agent(config=config)
            return

        # Find all markdown files in the agents directory
        for agent_file in agents_dir.glob("*.md"):
            try:
                # Read and parse the markdown file
                content = agent_file.read_text(encoding="utf-8")
                agent_config = AgentConfig.from_markdown(content)

                # Create and add the agent to registry
                agent = CyroAgent(agent_config, config.provider)
                self.registry.add(agent)

            except Exception as e:
                # Log error but continue loading other agents
                print(f"Warning: Failed to load agent from {agent_file}: {e}")
                continue

        # NOTE: Only add fallback general-engineer if no general-purpose agent exists
        try:
            default_agent = "general-engineer"
            self.get_agent_by_name(default_agent)
        except KeyError:
            self.add_general_agent(config=config)

    def process_request(self, message: str) -> str:
        """Process user request by selecting the appropriate agent and returning response.

        Args:
            message: User's request/query

        Returns:
            Agent's response as string
        """

        # TODO: cache the agent to reuse on the following requests?
        logger.info("Manager agent request", message=message)
        if len(self.registry.agents) > 1:
            # Route through manager to select the best agent
            try:
                result = self.run_sync(message)
            except Exception as routing_error:
                # Fallback: Route to general-engineer when AI routing fails
                logger.warning("⚠️  AI routing failed", error=str(routing_error))
                agent = self.get_agent_by_name("general-engineer")
            else:
                selection: AgentSelection = result.output  # type: ignore

                # Print routing information
                logger.info(
                    "🤖 Manager routing message",
                    agent=selection.recommended_agent,
                    reasoning=selection.reasoning,
                )

                agent = self.get_agent_by_id(selection.agent_id)
        else:
            logger.info("Using the only subagent")
            # NOTE: No need to go through manager if we only have one subagent
            agent = self.get_agent_by_name("general-engineer")

        logger.info("Sending agent request", agent=agent.id)
        response = agent.run_sync(message)
        return str(response.output)
