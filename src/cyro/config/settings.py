"""
Configuration models for Cyro using Pydantic.

This module defines the configuration structure for Cyro with support
for TOML files and environment variables.
"""

from typing import Dict, List, Optional

from pydantic import BaseModel


class ProviderConfig(BaseModel):
    """Configuration for AI providers."""

    host: str = "localhost"
    port: int = 11434
    model: str = "llama3.2"
    timeout: int = 30
    api_key: Optional[str] = None
    organization: Optional[str] = None


class AgentConfig(BaseModel):
    """Configuration for agent system."""

    discovery_path: str = "~/.config/cyro/agents"
    auto_load: bool = True
    default_tools: List[str] = ["filesystem", "web"]


class SecurityConfig(BaseModel):
    """Security and safety configuration."""

    sandbox_mode: bool = True
    require_approval: List[str] = ["filesystem", "code-execution"]
    max_file_size: str = "10MB"
    allowed_extensions: List[str] = [".py", ".js", ".ts", ".md", ".txt"]


class CyroConfig(BaseModel):
    """Main configuration model for Cyro."""

    # General settings
    default_agent: str = "auto"
    verbose: bool = False
    color_output: bool = True
    auto_update: bool = True

    # Provider configurations
    default_provider: str = "ollama"
    providers: Dict[str, ProviderConfig] = {}

    # Agent configuration
    agents: AgentConfig = AgentConfig()

    # Security configuration
    security: SecurityConfig = SecurityConfig()

    # UI settings
    theme: str = "auto"
    show_progress: bool = True
    streaming: bool = True
    panel_style: str = "rounded"
