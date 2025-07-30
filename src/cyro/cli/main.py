"""
Main CLI application for Cyro using Typer framework.

This module provides the primary CLI interface with command routing,
global options, and support for different interaction modes.
"""

import os
from typing import Optional

import typer
from rich.console import Console
from rich.panel import Panel
from rich.text import Text

from cyro import __version__
from cyro.cli.chat import start_chat_mode, start_chat_mode_with_query

# Create the main Typer application
app = typer.Typer(
    name="cyro",
    help="Cyro - Terminal-based AI coding agent with dynamic subagent creation",
    rich_markup_mode="rich",
    no_args_is_help=False,  # Allow running without args for interactive mode
)

# Initialize Rich console
console = Console()


def version_callback(value: bool):
    """Show version information."""
    if value:
        console.print(f"Cyro version {__version__}", style="bold yellow")
        raise typer.Exit()


@app.callback()
def main(
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to use"
    ),
    model: Optional[str] = typer.Option(
        None, "--model", "-m", help="Specify which model to use"
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

    Examples:
    - cyro                           # Show help
    - cyro chat                      # Interactive mode
    - cyro chat "your query"         # Interactive mode with initial query
    - cyro print "your query"        # Print response and exit
    - cyro --agent code-reviewer chat "review this code"
    """
    pass


def execute_print_mode(
    query: str, agent: Optional[str], model: Optional[str], verbose: bool
):
    """Execute a query in print mode and exit."""
    if verbose:
        console.print(
            f"[dim]Executing query with agent: {agent or 'auto'}, model: {model or 'default'}[/dim]"
        )

    # Create styled panel for the query
    query_panel = Panel(
        Text(query, style="bright_white"),
        title="[bold yellow]Query[/bold yellow]",
        border_style="yellow",
    )
    console.print(query_panel)

    # TODO: Implement actual AI agent execution
    response_panel = Panel(
        Text(
            f"🚧 AI execution not yet implemented.\n\nReceived: '{query}'",
            style="bright_yellow",
        ),
        title="[bold bright_green]Response[/bold bright_green]",
        border_style="bright_green",
    )
    console.print(response_panel)


@app.command()
def chat(
    query: Optional[str] = typer.Argument(None, help="Initial query to start with"),
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to chat with"
    ),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Enable verbose output"
    ),
):
    """Start an interactive chat session."""
    if query:
        start_chat_mode_with_query(query, agent, verbose)
    else:
        start_chat_mode(agent, verbose)


@app.command("print")
def print_cmd(
    query: str = typer.Argument(..., help="Query to process and print response"),
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to use"
    ),
    model: Optional[str] = typer.Option(
        None, "--model", "-m", help="Specify which model to use"
    ),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Enable verbose output"
    ),
):
    """Print response to query and exit (non-interactive mode)."""
    execute_print_mode(query, agent, model, verbose)


@app.command("agent")
def agent_cmd(
    action: str = typer.Argument(..., help="Action to perform (list, use, etc.)"),
):
    """Manage AI agents."""
    # TODO: Implement agent management
    console.print(f"🚧 Agent management not yet implemented. Action: {action}")


@app.command("config")
def config_cmd(
    action: str = typer.Argument(..., help="Action to perform (show, set, etc.)"),
):
    """Manage configuration."""
    # TODO: Implement config management
    console.print(f"🚧 Configuration management not yet implemented. Action: {action}")


@app.command()
def status():
    """Show current Cyro setup and status."""
    status_text = Text.assemble(
        ("Cyro Status\n\n", "bold yellow"),
        ("Version: ", "bright_white"),
        (__version__, "bold bright_green"),
        ("\nWorking Directory: ", "bright_white"),
        (os.getcwd(), "bold orange"),
        ("\nConfig: ", "bright_white"),
        ("Default", "yellow"),
        ("\nAgent: ", "bright_white"),
        ("Auto-select", "yellow"),
        ("\nModel: ", "bright_white"),
        ("Default (Ollama)", "yellow"),
    )

    panel = Panel(
        status_text,
        title="[bold yellow]Status[/bold yellow]",
        border_style="yellow",
        padding=(1, 2),
    )
    console.print(panel)


if __name__ == "__main__":
    app()
