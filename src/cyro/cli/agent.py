"""
Agent management commands for Cyro CLI.

This module provides commands to list, select, and manage AI agents.
"""

from typing import Optional

import typer

from cyro.utils.console import print_info

# Create agent management subcommand app
agent_app = typer.Typer(
    name="agent",
    help="Agent management commands",
    rich_markup_mode="rich",
)


@agent_app.command("list")
def list_agents(
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Show detailed information"
    ),
    category: Optional[str] = typer.Option(
        None, "--category", "-c", help="Filter by category"
    ),
):
    """List all available agents."""
    print_info("🚧 Agent listing not yet implemented.")


@agent_app.command("show")
def show_agent(
    name: str = typer.Argument(..., help="Name of the agent to show"),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Show technical details"
    ),
):
    """Show detailed information about a specific agent."""
    print_info(f"🚧 Agent details for '{name}' not yet implemented.")


@agent_app.command("use")
def use_agent(
    name: str = typer.Argument(..., help="Name of the agent to use as default"),
    global_default: bool = typer.Option(
        False, "--global", "-g", help="Set as global default"
    ),
):
    """Set the default agent for future interactions."""
    scope = "globally" if global_default else "for this session"
    print_info(f"🚧 Setting '{name}' as default agent {scope} not yet implemented.")


@agent_app.command("status")
def agent_status():
    """Show the current agent status and configuration."""
    print_info("🚧 Agent status not yet implemented.")


# Default command when just running 'cyro agent'
@agent_app.callback(invoke_without_command=True)
def agent_main(ctx: typer.Context):
    """Agent management commands."""
    if ctx.invoked_subcommand is None:
        agent_status()
