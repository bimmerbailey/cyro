"""
Main CLI application for Cyro using Typer framework.

This module provides the primary CLI interface with command routing,
global options, and support for different interaction modes.
"""

import os
from typing import Optional

import typer
from rich.panel import Panel
from rich.text import Text

from cyro import __version__
from cyro.cli.chat import start_chat_mode, start_chat_mode_with_query
from cyro.config.themes import get_theme_color
from cyro.utils.console import console

# Create the main Typer application
app = typer.Typer(
    name="cyro",
    help="Cyro - Terminal-based AI coding agent with dynamic subagent creation",
    rich_markup_mode="rich",
    no_args_is_help=False,  # Allow running without args for interactive mode
)


def version_callback(value: bool):
    """Show version information."""
    if value:
        console.print(f"Cyro version {__version__}", style=f"bold {get_theme_color('primary')}")
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
            f"[{get_theme_color('text_dim')}]Executing query with agent: {agent or 'auto'}, model: {model or 'default'}[/{get_theme_color('text_dim')}]"
        )

    # Create styled panel for the query
    query_panel = Panel(
        Text(query, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('primary')}]Query[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
    )
    console.print(query_panel)

    # TODO: Implement actual AI agent execution
    response_panel = Panel(
        Text(
            f"🚧 AI execution not yet implemented.\n\nReceived: '{query}'",
            style=get_theme_color("warning"),
        ),
        title=f"[bold {get_theme_color('success')}]Response[/bold {get_theme_color('success')}]",
        border_style=get_theme_color("success"),
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
    console.print(f"[{get_theme_color('warning')}]🚧 Agent management not yet implemented. Action: {action}[/{get_theme_color('warning')}]")


@app.command("config")
def config_cmd(
    subcommand: str = typer.Argument(..., help="Configuration area (theme, etc.)"),
    action: Optional[str] = typer.Argument(None, help="Action to perform or theme name"),
):
    """Manage configuration."""
    if subcommand == "theme":
        handle_theme_config(action)
    else:
        console.print(f"[{get_theme_color('warning')}]🚧 Configuration area '{subcommand}' not yet implemented[/{get_theme_color('warning')}]")


def handle_theme_config(action: Optional[str]):
    """Handle theme configuration commands."""
    from cyro.config.themes import list_themes, get_current_theme_name, set_theme, get_theme_info, load_custom_themes
    from cyro.utils.console import print_info, print_success, print_error
    
    if action is None or action == "list":
        # Show available themes
        show_theme_list()
    elif action == "current":
        # Show current theme
        current = get_current_theme_name()
        theme_info = get_theme_info(current)
        
        if theme_info:
            console.print(f"Current theme: [{get_theme_color('primary')}]{theme_info['name']}[/{get_theme_color('primary')}]")
            console.print(f"Description: [{get_theme_color('text')}]{theme_info['description']}[/{get_theme_color('text')}]")
        else:
            console.print(f"Current theme: [{get_theme_color('primary')}]{current}[/{get_theme_color('primary')}]")
    else:
        # Try to switch to the specified theme
        # First load custom themes to make sure we have everything available
        themes_dir = "~/.config/cyro/themes"
        custom_count = load_custom_themes(themes_dir)
        if custom_count > 0:
            print_info(f"Loaded {custom_count} custom theme{'s' if custom_count != 1 else ''}")
        
        if set_theme(action):
            theme_info = get_theme_info(action)
            if theme_info:
                print_success(f"Switched to '{theme_info['name']}' theme")
                console.print(f"[{get_theme_color('text_dim')}]{theme_info['description']}[/{get_theme_color('text_dim')}]")
            else:
                print_success(f"Switched to '{action}' theme")
        else:
            available_themes = list_themes()
            print_error(f"Theme '{action}' not found")
            console.print(f"Available themes: [{get_theme_color('info')}]{', '.join(available_themes)}[/{get_theme_color('info')}]")


def show_theme_list():
    """Display a formatted list of available themes."""
    from cyro.config.themes import list_themes, get_current_theme_name, get_theme_info, load_custom_themes
    
    # Load custom themes first
    themes_dir = "~/.config/cyro/themes"
    custom_count = load_custom_themes(themes_dir)
    
    all_themes = list_themes()
    current_theme = get_current_theme_name()
    
    # Prepare table data
    table_data = []
    for theme_name in all_themes:
        theme_info = get_theme_info(theme_name)
        is_current = "✓" if theme_name == current_theme else ""
        
        table_data.append({
            "current": is_current,
            "name": theme_name,
            "description": theme_info["description"] if theme_info else "No description",
        })
    
    # Print header info
    if custom_count > 0:
        console.print(f"[{get_theme_color('text_dim')}]Found {custom_count} custom theme{'s' if custom_count != 1 else ''} in {themes_dir}[/{get_theme_color('text_dim')}]")
        console.print()
    
    # Print themes table
    from rich.table import Table
    
    table = Table(title="Available Themes", show_header=True, header_style=get_theme_color("table_header"))
    table.add_column("", style=get_theme_color("success"), width=3)
    table.add_column("Theme", style=f"bold {get_theme_color('primary')}")
    table.add_column("Description", style=get_theme_color("table_row"))
    
    for theme in table_data:
        table.add_row(theme["current"], theme["name"], theme["description"])
    
    console.print(table)
    console.print()
    console.print(f"[{get_theme_color('text_dim')}]Use 'cyro config theme <name>' to switch themes[/{get_theme_color('text_dim')}]")


@app.command()
def status():
    """Show current Cyro setup and status."""
    status_text = Text.assemble(
        ("Cyro Status\n\n", f"bold {get_theme_color('primary')}"),
        ("Version: ", get_theme_color("text")),
        (__version__, f"bold {get_theme_color('success')}"),
        ("\nWorking Directory: ", get_theme_color("text")),
        (os.getcwd(), f"bold {get_theme_color('secondary')}"),
        ("\nConfig: ", get_theme_color("text")),
        ("Default", get_theme_color("info")),
        ("\nAgent: ", get_theme_color("text")),
        ("Auto-select", get_theme_color("info")),
        ("\nModel: ", get_theme_color("text")),
        ("Default (Ollama)", get_theme_color("info")),
    )

    panel = Panel(
        status_text,
        title=f"[bold {get_theme_color('primary')}]Status[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
        padding=(1, 2),
    )
    console.print(panel)


if __name__ == "__main__":
    app()
