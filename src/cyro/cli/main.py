"""
Main CLI application for Cyro using Typer framework.

This module provides the primary CLI interface with command routing,
global options, and support for different interaction modes.
"""

from dataclasses import dataclass
import os
from pathlib import Path
from typing import Optional

import typer
from rich.panel import Panel
from rich.table import Table
from rich.text import Text

from cyro import __version__
from cyro.agents.manager import ManagerAgent
from cyro.cli.chat import start_chat_mode, start_chat_mode_with_query
from cyro.config.themes import (
    get_current_theme_name,
    get_theme_color,
    get_theme_info,
    list_themes,
    load_custom_themes,
    set_theme,
    ThemeManager,
)
from cyro.utils.console import console, print_error, print_info, print_success


@dataclass
class AppContext:
    theme: ThemeManager
    manager: ManagerAgent


# Create the main Typer application
app = typer.Typer(
    name="cyro",
    help="Cyro - Terminal-based AI coding agent with dynamic subagent creation",
    rich_markup_mode="rich",
    no_args_is_help=False,  # Allow running without args for interactive mode
)


def _get_color(semantic_name: str, ctx: typer.Context) -> str:
    """Helper to get theme color from context."""
    if ctx.obj is None or not isinstance(ctx.obj, AppContext):
        # Fallback to global theme manager
        return get_theme_color(semantic_name)
    return ctx.obj.theme.get_color(semantic_name)


def version_callback(value: bool):
    """Show version information."""
    if value:
        console.print(
            f"Cyro version {__version__}", style=f"bold {get_theme_color('primary')}"
        )
        raise typer.Exit()


@app.callback()
def main(
    ctx: typer.Context,
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
    # Initialize theme manager and manager agent, store in context
    ctx.ensure_object(dict)

    # Look for agents in project's .cyro directory, fallback to ~/.cyro
    project_config_dir = Path(".cyro")
    if project_config_dir.exists():
        config_dir = project_config_dir
    else:
        config_dir = Path("~/.cyro").expanduser()

    ctx.obj = AppContext(
        theme=ThemeManager(), manager=ManagerAgent(config_dir=config_dir)
    )


def execute_print_mode(
    query: str,
    agent: Optional[str],
    model: Optional[str],
    verbose: bool,
    ctx: typer.Context,
):
    """Execute a query in print mode and exit."""
    manager_agent = ctx.obj.manager

    if verbose:
        console.print(
            f"[{get_theme_color('text_dim')}]Executing query with agent: {agent or 'auto'}, model: {model or 'default'}[/{get_theme_color('text_dim')}]"
        )

    # Create styled panel for the query
    query_panel = Panel(
        Text(query, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('primary')}]Query[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
    )
    console.print(query_panel)

    # Process query with AI agent
    try:
        if agent:
            # Use explicitly requested agent
            selected_agent = manager_agent.get_agent_by_name(agent)
            response = selected_agent.run_sync(query)
            response_text = (
                response.data if hasattr(response, "data") else str(response)
            )
        else:
            # Route through manager
            response_text = manager_agent.process_request(query)

        response_panel = Panel(
            Text(response_text, style=get_theme_color("text")),
            title=f"[bold {get_theme_color('success')}]Response[/bold {get_theme_color('success')}]",
            border_style=get_theme_color("success"),
        )
    except Exception as e:
        response_panel = Panel(
            Text(f"Error: {str(e)}", style=get_theme_color("error")),
            title=f"[bold {get_theme_color('error')}]Error[/bold {get_theme_color('error')}]",
            border_style=get_theme_color("error"),
        )

    console.print(response_panel)


@app.command()
def chat(
    ctx: typer.Context,
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
        start_chat_mode_with_query(query, agent, verbose, ctx)
    else:
        start_chat_mode(agent, verbose, ctx)


@app.command("print")
def print_cmd(
    ctx: typer.Context,
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
    execute_print_mode(query, agent, model, verbose, ctx)


@app.command("agent")
def agent_cmd(
    action: str = typer.Argument(..., help="Action to perform (list, use, etc.)"),
):
    """Manage AI agents."""
    # TODO: Implement agent management
    console.print(
        f"[{get_theme_color('warning')}]🚧 Agent management not yet implemented. Action: {action}[/{get_theme_color('warning')}]"
    )


@app.command("config")
def config_cmd(
    ctx: typer.Context,
    subcommand: str = typer.Argument(..., help="Configuration area (theme, etc.)"),
    action: Optional[str] = typer.Argument(
        None, help="Action to perform or theme name"
    ),
):
    """Manage configuration."""
    if subcommand == "theme":
        handle_theme_config(action, ctx)
    else:
        console.print(
            f"[{_get_color('warning', ctx)}]🚧 Configuration area '{subcommand}' not yet implemented[/{_get_color('warning', ctx)}]"
        )


def handle_theme_config(action: Optional[str], ctx: typer.Context):
    """Handle theme configuration commands."""

    theme_manager = ctx.obj.theme

    if action is None or action == "list":
        # Show available themes
        show_theme_list(ctx)
    elif action == "current":
        # Show current theme
        current = get_current_theme_name(theme_manager)
        theme_info = get_theme_info(current, theme_manager)

        if theme_info:
            console.print(
                f"Current theme: [{_get_color('primary', ctx)}]{theme_info['name']}[/{_get_color('primary', ctx)}]"
            )
            console.print(
                f"Description: [{_get_color('text', ctx)}]{theme_info['description']}[/{_get_color('text', ctx)}]"
            )
        else:
            console.print(
                f"Current theme: [{_get_color('primary', ctx)}]{current}[/{_get_color('primary', ctx)}]"
            )
    else:
        # Try to switch to the specified theme
        # First load custom themes to make sure we have everything available
        themes_dir = "~/.cyro/themes"
        custom_count = load_custom_themes(theme_manager, themes_dir)
        if custom_count > 0:
            print_info(
                f"Loaded {custom_count} custom theme{'s' if custom_count != 1 else ''}"
            )

        if set_theme(theme_manager, action):
            theme_info = get_theme_info(action, theme_manager)
            if theme_info:
                print_success(f"Switched to '{theme_info['name']}' theme")
                console.print(
                    f"[{_get_color('text_dim', ctx)}]{theme_info['description']}[/{_get_color('text_dim', ctx)}]"
                )
            else:
                print_success(f"Switched to '{action}' theme")
        else:
            available_themes = list_themes(theme_manager)
            print_error(f"Theme '{action}' not found")
            console.print(
                f"Available themes: [{_get_color('info', ctx)}]{', '.join(available_themes)}[/{_get_color('info', ctx)}]"
            )


def show_theme_list(ctx: typer.Context):
    """Display a formatted list of available themes."""

    theme_manager = ctx.obj.theme

    # Load custom themes first
    themes_dir = "~/.cyro/themes"
    custom_count = load_custom_themes(theme_manager, themes_dir)

    all_themes = list_themes(theme_manager)
    current_theme = get_current_theme_name(theme_manager)

    # Prepare table data
    table_data = []
    for theme_name in all_themes:
        theme_info = get_theme_info(theme_name, theme_manager)
        is_current = "✓" if theme_name == current_theme else ""

        table_data.append(
            {
                "current": is_current,
                "name": theme_name,
                "description": theme_info["description"]
                if theme_info
                else "No description",
            }
        )

    # Print header info
    if custom_count > 0:
        console.print(
            f"[{_get_color('text_dim', ctx)}]Found {custom_count} custom theme{'s' if custom_count != 1 else ''} in {themes_dir}[/{_get_color('text_dim', ctx)}]"
        )
        console.print()

    # Print themes table
    table = Table(
        title="Available Themes",
        show_header=True,
        header_style=_get_color("table_header", ctx),
    )
    table.add_column("", style=_get_color("success", ctx), width=3)
    table.add_column("Theme", style=f"bold {_get_color('primary', ctx)}")
    table.add_column("Description", style=_get_color("table_row", ctx))

    for theme in table_data:
        table.add_row(theme["current"], theme["name"], theme["description"])

    console.print(table)
    console.print()
    console.print(
        f"[{_get_color('text_dim', ctx)}]Use 'cyro config theme <name>' to switch themes[/{_get_color('text_dim', ctx)}]"
    )


@app.command()
def status(ctx: typer.Context):
    """Show current Cyro setup and status."""
    status_text = Text.assemble(
        ("Cyro Status\n\n", f"bold {_get_color('primary', ctx)}"),
        ("Version: ", _get_color("text", ctx)),
        (__version__, f"bold {_get_color('success', ctx)}"),
        ("\nWorking Directory: ", _get_color("text", ctx)),
        (os.getcwd(), f"bold {_get_color('secondary', ctx)}"),
        ("\nConfig: ", _get_color("text", ctx)),
        ("Default", _get_color("info", ctx)),
        ("\nAgent: ", _get_color("text", ctx)),
        ("Auto-select", _get_color("info", ctx)),
        ("\nModel: ", _get_color("text", ctx)),
        ("Default (Ollama)", _get_color("info", ctx)),
    )

    panel = Panel(
        status_text,
        title=f"[bold {_get_color('primary', ctx)}]Status[/bold {_get_color('primary', ctx)}]",
        border_style=_get_color("border", ctx),
        padding=(1, 2),
    )
    console.print(panel)


if __name__ == "__main__":
    app()
