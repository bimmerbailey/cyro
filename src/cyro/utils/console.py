"""
Rich console utilities for Cyro CLI.

This module provides a centralized Rich console instance and utility functions
for consistent terminal output formatting across the application.
"""

from typing import Optional

from rich.console import Console
from rich.panel import Panel
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.syntax import Syntax
from rich.table import Table
from rich.text import Text

# Global console instance for consistent output
console = Console()


def print_info(message: str, title: str = "Info") -> None:
    """Print an informational message with yellow styling."""
    panel = Panel(
        Text(message, style="bright_white"),
        title=f"[bold yellow]{title}[/bold yellow]",
        border_style="yellow",
    )
    console.print(panel)


def print_success(message: str, title: str = "Success") -> None:
    """Print a success message with green styling."""
    panel = Panel(
        Text(message, style="bright_white"),
        title=f"[bold bright_green]{title}[/bold bright_green]",
        border_style="bright_green",
    )
    console.print(panel)


def print_warning(message: str, title: str = "Warning") -> None:
    """Print a warning message with bright yellow styling."""
    panel = Panel(
        Text(message, style="bright_white"),
        title=f"[bold bright_yellow]{title}[/bold bright_yellow]",
        border_style="bright_yellow",
    )
    console.print(panel)


def print_error(message: str, title: str = "Error") -> None:
    """Print an error message with red styling."""
    panel = Panel(
        Text(message, style="bright_white"),
        title=f"[bold bright_red]{title}[/bold bright_red]",
        border_style="bright_red",
    )
    console.print(panel)


def print_code(code: str, language: str = "python", title: str = "Code") -> None:
    """Print syntax-highlighted code."""
    syntax = Syntax(code, language, theme="monokai", line_numbers=True)
    panel = Panel(
        syntax,
        title=f"[bold orange]{title}[/bold orange]",
        border_style="orange",
    )
    console.print(panel)


def print_table(data: list[dict], title: str = "Table") -> None:
    """Print a formatted table from list of dictionaries."""
    if not data:
        print_info("No data to display", title)
        return

    table = Table(title=title, show_header=True, header_style="bold yellow")

    # Add columns based on first row keys
    for key in data[0].keys():
        table.add_column(str(key).title(), style="bright_white")

    # Add rows
    for row in data:
        table.add_row(*[str(value) for value in row.values()])

    console.print(table)


def create_progress() -> Progress:
    """Create a Rich progress bar with spinner."""
    return Progress(
        SpinnerColumn(),
        TextColumn("[progress.description]{task.description}"),
        console=console,
    )


def prompt_user(message: str, default: Optional[str] = None) -> str:
    """Prompt user for input with optional default value."""
    if default:
        prompt_text = f"[bold yellow]{message}[/bold yellow] [yellow]({default})[/yellow]: "
    else:
        prompt_text = f"[bold yellow]{message}[/bold yellow]: "

    try:
        response = console.input(prompt_text).strip()
        return response if response else (default or "")
    except (KeyboardInterrupt, EOFError):
        console.print("\n[dim]Operation cancelled[/dim]")
        return ""


def confirm(message: str, default: bool = False) -> bool:
    """Ask user for yes/no confirmation."""
    default_text = "Y/n" if default else "y/N"
    prompt_text = f"[bold yellow]{message}[/bold yellow] [yellow]({default_text})[/yellow]: "

    try:
        response = console.input(prompt_text).strip().lower()
        if not response:
            return default
        return response.startswith("y")
    except (KeyboardInterrupt, EOFError):
        console.print("\n[dim]Operation cancelled[/dim]")
        return False


def print_welcome():
    """Print the Cyro welcome message."""
    welcome = Panel(
        Text.assemble(
            ("Welcome to ", "bright_white"),
            ("Cyro", "bold yellow"),
            (" - Terminal-based AI coding agent\n\n", "bright_white"),
            ("🤖 Privacy-first AI assistant with local Ollama support\n", "bright_green"),
            ("🔧 Dynamic subagent creation through markdown configuration\n", "bright_green"),
            ("🚀 Extensible architecture for multiple AI providers\n\n", "bright_green"),
        ),
        title="[bold yellow]Cyro AI Coding Agent[/bold yellow]",
        border_style="yellow",
        padding=(1, 2),
    )
    console.print(welcome)


def print_help():
    """Print help information."""
    help_text = Text.assemble(
        ("Available commands:\n\n", "bright_white"),
        ("Interactive Mode:\n", "bold orange"),
        ("• Enter any prompt for direct execution\n", "bright_white"),
        ("• ", "bright_white"),
        ("chat", "bold bright_yellow"),
        (" - Start interactive chat mode\n", "bright_white"),
        ("• ", "bright_white"),
        ("agent list", "bold bright_yellow"),
        (" - Show available agents\n", "bright_white"),
        ("• ", "bright_white"),
        ("config show", "bold bright_yellow"),
        (" - Show configuration\n", "bright_white"),
        ("• ", "bright_white"),
        ("help", "bold bright_yellow"),
        (" - Show this help\n", "bright_white"),
        ("• ", "bright_white"),
        ("exit/quit/q", "bold bright_yellow"),
        (" - Exit Cyro\n\n", "bright_white"),
        ("Command Line:\n", "bold orange"),
        ("• ", "bright_white"),
        ('cyro "prompt"', "bold bright_yellow"),
        (" - Execute prompt directly\n", "bright_white"),
        ("• ", "bright_white"),
        ('cyro --agent <name> "prompt"', "bold bright_yellow"),
        (" - Use specific agent\n", "bright_white"),
        ("• ", "bright_white"),
        ("cyro chat", "bold bright_yellow"),
        (" - Start chat mode\n", "bright_white"),
        ("• ", "bright_white"),
        ("cyro agent list", "bold bright_yellow"),
        (" - List agents\n", "bright_white"),
        ("• ", "bright_white"),
        ("cyro config show", "bold bright_yellow"),
        (" - Show config\n", "bright_white"),
    )

    panel = Panel(
        help_text,
        title="[bold yellow]Cyro Help[/bold yellow]",
        border_style="yellow",
        padding=(1, 2),
    )
    console.print(panel)
