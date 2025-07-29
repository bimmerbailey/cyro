"""
Main CLI application for Cyro using Typer framework.

This module provides the primary CLI interface with command routing,
global options, and support for different interaction modes.
"""

from typing import Optional

import typer
from rich.console import Console
from rich.panel import Panel
from rich.text import Text

from cyro.cli.agent import agent_app
from cyro.cli.chat import chat_app
from cyro.cli.config import config_app

# Create the main Typer application
app = typer.Typer(
    name="cyro",
    help="Cyro - Terminal-based AI coding agent with dynamic subagent creation",
    rich_markup_mode="rich",
    no_args_is_help=False,  # Allow running without args for interactive mode
)

# Initialize Rich console
console = Console()

# Register subcommands
app.add_typer(chat_app, name="chat", help="Interactive chat commands")
app.add_typer(agent_app, name="agent", help="Agent management commands")
app.add_typer(config_app, name="config", help="Configuration management commands")


def version_callback(value: bool):
    """Show version information."""
    if value:
        from cyro import __version__

        console.print(f"Cyro version {__version__}", style="bold blue")
        raise typer.Exit()


@app.callback(invoke_without_command=True)
def main(
    ctx: typer.Context,
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to use for the prompt"
    ),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Enable verbose output"
    ),
    config: Optional[str] = typer.Option(
        None, "--config", "-c", help="Path to configuration file"
    ),
    version: Optional[bool] = typer.Option(
        None,
        "--version",
        callback=version_callback,
        is_eager=True,
        help="Show version and exit",
    ),
):
    """
    Cyro - Terminal-based AI coding agent.

    Run without arguments for interactive mode.

    Examples:
    - cyro                           # Interactive mode
    - cyro chat                      # Start chat mode
    - cyro agent list                # List available agents
    """
    # Store global options in context for other commands
    if not hasattr(ctx, "obj") or ctx.obj is None:
        ctx.obj = {}
    ctx.obj.update(
        {
            "verbose": verbose,
            "config": config,
            "agent": agent,
        }
    )

    # If no subcommand, start chat mode directly
    if ctx.invoked_subcommand is None:
        from cyro.cli.chat import start_chat_mode
        start_chat_mode(agent, verbose)


@app.command("run")
def run_prompt(
    prompt: str = typer.Argument(..., help="Prompt to execute"),
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to use"
    ),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Enable verbose output"
    ),
):
    """Execute a direct prompt with optional agent specification."""
    execute_prompt(prompt, agent, verbose)


def execute_prompt(prompt: str, agent: Optional[str], verbose: bool):
    """Execute a direct prompt with optional agent specification."""
    if verbose:
        console.print(f"[dim]Executing prompt with agent: {agent or 'auto'}[/dim]")

    # Create styled panel for the prompt
    prompt_panel = Panel(
        Text(prompt, style="white"),
        title="[bold blue]Prompt[/bold blue]",
        border_style="blue",
    )
    console.print(prompt_panel)

    # TODO: Implement actual AI agent execution
    response_panel = Panel(
        Text(
            f"🚧 AI execution not yet implemented.\n\nReceived: '{prompt}'",
            style="yellow",
        ),
        title="[bold green]Response[/bold green]",
        border_style="green",
    )
    console.print(response_panel)




if __name__ == "__main__":
    app()
