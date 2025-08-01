"""
Manager agent for routing tasks and coordinating subagents.

This module provides the ManagerAgent class that acts as a central coordinator
for all subagents, handling task routing, agent selection, and delegation.
"""

from pathlib import Path

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


class AgentSelection(BaseModel):
    """Response containing selected agent and reasoning."""

    recommended_agent: str = Field(
        description="Recommended agent to handle the user request"
    )
    agent_id: UUID4 = Field(description="The ID of the agent that has been selected")
    reasoning: str = Field(description="The reasoning that the agent is selected")


system_prompt = """
You are a Manager Agent responsible for analyzing user requests and routing them to the most appropriate specialized agent. Your sole responsibility is intelligent request routing and delegation.

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

        # Debug: Show loaded agents with names
        agent_names = [agent.metadata.name for agent in self.registry]
        print(
            f"📁 Loaded {len(agent_names)} agents from {self.config_dir}: {agent_names}"
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
        instructions = """## Agent Selection Instructions

You are a routing specialist. Your job is to analyze user requests and select the single best agent to handle them. Follow this systematic approach:

### Step 1: Request Analysis

**Parse the user's request for these key elements:**

1. **Primary Action Verb**: What is the user trying to do?
   - CREATE: build, make, implement, develop, generate
   - DEBUG: fix, resolve, troubleshoot, diagnose, investigate  
   - REVIEW: analyze, audit, check, validate, evaluate
   - MODIFY: update, refactor, optimize, enhance, change
   - DEPLOY: launch, publish, release, configure, setup
   - LEARN: explain, understand, document, research

2. **Technical Domain**: What area of expertise is involved?
   - Frontend: React, Vue, Angular, HTML/CSS, UI/UX
   - Backend: APIs, databases, servers, microservices
   - Data: analysis, processing, ML, visualization, ETL
   - DevOps: CI/CD, infrastructure, containers, cloud
   - Mobile: iOS, Android, React Native, Flutter
   - Security: authentication, encryption, penetration testing
   - Testing: unit tests, integration, e2e, performance

3. **Scope & Complexity**: How extensive is the work?
   - SIMPLE: Single file, quick fix, straightforward task
   - MODERATE: Multiple files, requires planning, some complexity
   - COMPLEX: Architecture changes, multi-system integration

4. **Context Clues**: Additional indicators from the request
   - File types mentioned (.py, .js, .dockerfile, etc.)
   - Technologies named (React, PostgreSQL, AWS, etc.)
   - Specific tools referenced (Jest, Docker, Kubernetes, etc.)

### Step 2: Agent Capability Matching

**Score each available agent (0-10) based on:**

- **Domain Expertise** (0-4): Does the agent specialize in this technical area?
- **Tool Availability** (0-3): Does the agent have the right tools for the job?
- **Scope Match** (0-3): Can the agent handle the complexity and scope?

**Selection Logic:**
- Score ≥ 8: Strong match - select this specialized agent
- Score 6-7: Good match - prefer over generalist if clearly better
- Score ≤ 5: Weak match - consider Software Engineer instead

### Available Agents

"""

        # Add registered specialized agents with enhanced information
        for agent in self.registry:
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

        instructions += """### Decision Matrix Examples

**Request**: "Fix the login bug in our React app"
- Action: DEBUG, Domain: Frontend, Complexity: MODERATE
- Frontend Specialist (if available): Score 9/10 → SELECT
- Software Engineer: Score 7/10 → Use if no specialist

**Request**: "Analyze customer data trends from this CSV"
- Action: LEARN, Domain: Data, Complexity: MODERATE  
- Data Analyst (if available): Score 10/10 → SELECT
- Software Engineer: Score 5/10 → Only if no data specialist

**Request**: "Implement user authentication system"
- Action: CREATE, Domain: Backend/Security, Complexity: COMPLEX
- Security Engineer (if available): Score 9/10 → SELECT
- Backend Engineer (if available): Score 8/10 → SELECT  
- Software Engineer: Score 7/10 → Use if no specialists

**Request**: "Refactor this Python function for better performance"
- Action: MODIFY, Domain: Backend, Complexity: SIMPLE
- Software Engineer: Score 8/10 → SELECT (general optimization task)

### Critical Decision Rules

1. **Specialized > Generalist**: When a specialist scores ≥ 8, choose them over Software Engineer
2. **Complexity Matching**: Ensure selected agent can handle the scope (Simple/Moderate/Complex)
3. **Tool Requirements**: Verify the agent has necessary tools (filesystem, git, web, etc.)
4. **Domain Alignment**: Strong domain match trumps secondary considerations
5. **Default Fallback**: When in doubt or scores are close (≤ 1 point difference), choose Software Engineer

### Response Format

Respond with exactly this JSON structure:

```json
{
    "recommended_agent": "exact_agent_name_from_above_list",
    "agent_id": "uuid4_from_agent_listing_or_default_for_software_engineer", 
    "reasoning": "Based on [ACTION] request in [DOMAIN] domain with [COMPLEXITY] scope, selected [AGENT] because [SPECIFIC_CAPABILITY_MATCH]. Score: X/10."
}
```

**Example responses:**

```json
{
    "recommended_agent": "Software Engineer",
    "agent_id": "default",
    "reasoning": "Based on CREATE request in Backend domain with COMPLEX scope, selected Software Engineer because no specialized backend agent available and requires full-stack capabilities. Score: 7/10."
}
```

```json
{
    "recommended_agent": "Code Reviewer", 
    "agent_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "reasoning": "Based on REVIEW request in Quality domain with MODERATE scope, selected Code Reviewer because specialized in code analysis and quality assurance. Score: 9/10."
}
```

### Final Reminders

- **Be Decisive**: Never ask clarifying questions - make the best decision with available information
- **Score Systematically**: Use the 0-10 scoring system to justify selections
- **Prefer Specialists**: When domain expertise clearly applies (score ≥ 8)  
- **Trust the Default**: Software Engineer handles anything specialists can't
- **Match Complexity**: Ensure selected agent can handle the request scope
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

        # TODO: Clean this up
        # NOTE: Only add fallback general-engineer if no general-purpose agent exists
        has_general_agent = False
        # Check for common general-purpose agent names
        for agent_name in ["general-engineer", "engineer", "software-engineer"]:
            try:
                self.get_agent_by_name(agent_name)
                has_general_agent = True
                break
            except KeyError:
                continue

        # Only add fallback if no general-purpose agent was found
        if not has_general_agent:
            self.add_general_agent(config=config)

    def process_request(self, message: str) -> str:
        """Process user request by selecting appropriate agent and returning response.

        Args:
            message: User's request/query

        Returns:
            Agent's response as string
        """

        # Route through manager to select best agent
        try:
            result = self.run_sync(message)

        except Exception as routing_error:
            # Fallback: Route to general-engineer when AI routing fails
            print(
                f"⚠️  AI routing failed ({routing_error}), falling back to general-engineer"
            )
            agent = self.get_agent_by_name("general-engineer")

        else:
            selection: AgentSelection = result.output  # type: ignore

            # Print routing information
            print(f"🤖 Manager routing message to: {selection.recommended_agent}")
            print(f"📝 Reasoning: {selection.reasoning}")

            # Get the selected agent and execute
            agent = self.get_agent_by_id(selection.agent_id)

        print(f"Agent: {agent.id}, name: {agent.metadata.name}")
        response = agent.run_sync(message)
        return str(response.data)
