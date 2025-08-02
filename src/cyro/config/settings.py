"""
Configuration models for Cyro using Pydantic Settings.

This module defines the configuration structure for Cyro with support
for TOML files and environment variables.
"""

from typing import List, cast

from pydantic import AnyHttpUrl, BaseModel, Field
from pydantic_ai.models.openai import OpenAIModel
from pydantic_ai.providers.openai import OpenAIProvider
from pydantic_settings import (
    BaseSettings,
    PydanticBaseSettingsSource,
    SettingsConfigDict,
    TomlConfigSettingsSource,
)


class SecurityConfig(BaseModel):
    """Security and safety configuration."""

    sandbox_mode: bool = True
    require_approval: List[str] = ["filesystem", "code-execution"]
    max_file_size: str = "10MB"


class CyroConfig(BaseSettings):
    """Main configuration model for Cyro with support for TOML files and environment variables."""

    model_config = SettingsConfigDict(
        toml_file=["~/.cyro/config.toml"],
        env_prefix="CYRO_",
        env_nested_delimiter="__",
        case_sensitive=False,
    )

    # General settings
    default_agent: str = "auto"
    verbose: bool = False
    color_output: bool = True
    auto_update: bool = True

    # Provider configurations (Ollama-focused)
    host: str = "localhost"
    port: int = 11434
    model: str = "qwen2.5-coder"
    timeout: int = 30
    base_url: AnyHttpUrl = Field(
        default=cast("AnyHttpUrl", "http://localhost:11434/v1")
    )

    # Security configuration
    security: SecurityConfig = SecurityConfig()

    # UI settings
    theme: str = "cyro"
    show_progress: bool = True
    streaming: bool = True
    panel_style: str = "rounded"

    @classmethod
    def settings_customise_sources(
        cls,
        settings_cls: type[BaseSettings],
        init_settings: PydanticBaseSettingsSource,
        env_settings: PydanticBaseSettingsSource,
        dotenv_settings: PydanticBaseSettingsSource,
        file_secret_settings: PydanticBaseSettingsSource,
    ) -> tuple[PydanticBaseSettingsSource, ...]:
        """
        Customize settings sources priority:
        1. Initialization arguments (highest priority)
        2. Environment variables (CYRO_*)
        3. TOML configuration files
        """
        return (
            init_settings,  # Highest priority
            env_settings,  # Environment variables
            TomlConfigSettingsSource(settings_cls),  # TOML files
        )

    @property
    def provider(self) -> OpenAIModel:
        # TODO: this will expand over time
        #  we might need a token as well
        """Get the configured AI model provider."""
        provider_instance = OpenAIProvider(base_url=str(self.base_url))
        return OpenAIModel(
            model_name=self.model,
            provider=provider_instance,
        )
