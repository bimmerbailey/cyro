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
    """Print an informational message with blue styling."""
    panel = Panel(
        Text(message, style="white"),
        title=f"[bold blue]{title}[/bold blue]",
        border_style="blue",
    )
    console.print(panel)


def print_success(message: str, title: str = "Success") -> None:
    """Print a success message with green styling."""
    panel = Panel(
        Text(message, style="white"),
        title=f"[bold green]{title}[/bold green]",
        border_style="green",
    )
    console.print(panel)


def print_warning(message: str, title: str = "Warning") -> None:
    """Print a warning message with yellow styling."""
    panel = Panel(
        Text(message, style="white"),
        title=f"[bold yellow]{title}[/bold yellow]",
        border_style="yellow",
    )
    console.print(panel)


def print_error(message: str, title: str = "Error") -> None:
    """Print an error message with red styling."""
    panel = Panel(
        Text(message, style="white"),
        title=f"[bold red]{title}[/bold red]",
        border_style="red",
    )
    console.print(panel)


def print_code(code: str, language: str = "python", title: str = "Code") -> None:
    """Print syntax-highlighted code."""
    syntax = Syntax(code, language, theme="monokai", line_numbers=True)
    panel = Panel(
        syntax,
        title=f"[bold cyan]{title}[/bold cyan]",
        border_style="cyan",
    )
    console.print(panel)


def print_table(data: list[dict], title: str = "Table") -> None:
    """Print a formatted table from list of dictionaries."""
    if not data:
        print_info("No data to display", title)
        return

    table = Table(title=title, show_header=True, header_style="bold blue")

    # Add columns based on first row keys
    for key in data[0].keys():
        table.add_column(str(key).title(), style="white")

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
        prompt_text = f"[bold blue]{message}[/bold blue] [dim]({default})[/dim]: "
    else:
        prompt_text = f"[bold blue]{message}[/bold blue]: "

    try:
        response = console.input(prompt_text).strip()
        return response if response else (default or "")
    except (KeyboardInterrupt, EOFError):
        console.print("\n[dim]Operation cancelled[/dim]")
        return ""


def confirm(message: str, default: bool = False) -> bool:
    """Ask user for yes/no confirmation."""
    default_text = "Y/n" if default else "y/N"
    prompt_text = f"[bold blue]{message}[/bold blue] [dim]({default_text})[/dim]: "

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
            ("Welcome to ", "white"),
            ("Cyro", "bold blue"),
            (" - Terminal-based AI coding agent\n\n", "white"),
            ("🤖 Privacy-first AI assistant with local Ollama support\n", "green"),
            ("🔧 Dynamic subagent creation through markdown configuration\n", "green"),
            ("🚀 Extensible architecture for multiple AI providers\n\n", "green"),
        ),
        title="[bold blue]Cyro AI Coding Agent[/bold blue]",
        border_style="blue",
        padding=(1, 2),
    )
    console.print(welcome)


def print_help():
    """Print help information."""
    help_text = Text.assemble(
        ("Available commands:\n\n", "white"),
        ("Interactive Mode:\n", "bold cyan"),
        ("• Enter any prompt for direct execution\n", "white"),
        ("• ", "white"),
        ("chat", "bold yellow"),
        (" - Start interactive chat mode\n", "white"),
        ("• ", "white"),
        ("agent list", "bold yellow"),
        (" - Show available agents\n", "white"),
        ("• ", "white"),
        ("config show", "bold yellow"),
        (" - Show configuration\n", "white"),
        ("• ", "white"),
        ("help", "bold yellow"),
        (" - Show this help\n", "white"),
        ("• ", "white"),
        ("exit/quit/q", "bold yellow"),
        (" - Exit Cyro\n\n", "white"),
        ("Command Line:\n", "bold cyan"),
        ("• ", "white"),
        ('cyro "prompt"', "bold yellow"),
        (" - Execute prompt directly\n", "white"),
        ("• ", "white"),
        ('cyro --agent <name> "prompt"', "bold yellow"),
        (" - Use specific agent\n", "white"),
        ("• ", "white"),
        ("cyro chat", "bold yellow"),
        (" - Start chat mode\n", "white"),
        ("• ", "white"),
        ("cyro agent list", "bold yellow"),
        (" - List agents\n", "white"),
        ("• ", "white"),
        ("cyro config show", "bold yellow"),
        (" - Show config\n", "white"),
    )

    panel = Panel(
        help_text,
        title="[bold blue]Cyro Help[/bold blue]",
        border_style="blue",
        padding=(1, 2),
    )
    console.print(panel)
