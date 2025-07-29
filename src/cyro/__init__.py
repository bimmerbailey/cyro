"""
Cyro - Terminal-based AI coding agent with dynamic subagent creation.

A privacy-first AI coding assistant that defaults to local Ollama models
with support for multiple providers and extensible agent architecture.
"""

__version__ = "0.1.0"

from cyro.cli.main import app as cli_app


def main():
    """Main entry point for the Cyro CLI application."""
    cli_app()


__all__ = ["main", "cli_app", "__version__"]
