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

from cyro.config.themes import get_theme_color

# Global console instance for consistent output
console = Console()


def print_info(message: str, title: str = "Info") -> None:
    """Print an informational message with themed styling."""
    panel = Panel(
        Text(message, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('info')}]{title}[/bold {get_theme_color('info')}]",
        border_style=get_theme_color("border"),
    )
    console.print(panel)


def print_success(message: str, title: str = "Success") -> None:
    """Print a success message with themed styling."""
    panel = Panel(
        Text(message, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('success')}]{title}[/bold {get_theme_color('success')}]",
        border_style=get_theme_color("success"),
    )
    console.print(panel)


def print_warning(message: str, title: str = "Warning") -> None:
    """Print a warning message with themed styling."""
    panel = Panel(
        Text(message, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('warning')}]{title}[/bold {get_theme_color('warning')}]",
        border_style=get_theme_color("warning"),
    )
    console.print(panel)


def print_error(message: str, title: str = "Error") -> None:
    """Print an error message with themed styling."""
    panel = Panel(
        Text(message, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('error')}]{title}[/bold {get_theme_color('error')}]",
        border_style=get_theme_color("error"),
    )
    console.print(panel)


def print_code(code: str, language: str = "python", title: str = "Code") -> None:
    """Print syntax-highlighted code."""
    syntax = Syntax(code, language, theme="monokai", line_numbers=True)
    panel = Panel(
        syntax,
        title=f"[bold {get_theme_color('secondary')}]{title}[/bold {get_theme_color('secondary')}]",
        border_style=get_theme_color("secondary"),
    )
    console.print(panel)


def print_table(data: list[dict], title: str = "Table") -> None:
    """Print a formatted table from list of dictionaries."""
    if not data:
        print_info("No data to display", title)
        return

    table = Table(
        title=title, show_header=True, header_style=get_theme_color("table_header")
    )

    # Add columns based on first row keys
    for key in data[0].keys():
        table.add_column(str(key).title(), style=get_theme_color("table_row"))

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
        prompt_text = f"[bold {get_theme_color('prompt')}]{message}[/bold {get_theme_color('prompt')}] [{get_theme_color('input_default')}]({default})[/{get_theme_color('input_default')}]: "
    else:
        prompt_text = f"[bold {get_theme_color('prompt')}]{message}[/bold {get_theme_color('prompt')}]: "

    try:
        response = console.input(prompt_text).strip()
        return response if response else (default or "")
    except (KeyboardInterrupt, EOFError):
        console.print(
            f"\n[{get_theme_color('text_dim')}]Operation cancelled[/{get_theme_color('text_dim')}]"
        )
        return ""


def confirm(message: str, default: bool = False) -> bool:
    """Ask user for yes/no confirmation."""
    default_text = "Y/n" if default else "y/N"
    prompt_text = f"[bold {get_theme_color('prompt')}]{message}[/bold {get_theme_color('prompt')}] [{get_theme_color('input_default')}]({default_text})[/{get_theme_color('input_default')}]: "

    try:
        response = console.input(prompt_text).strip().lower()
        if not response:
            return default
        return response.startswith("y")
    except (KeyboardInterrupt, EOFError):
        console.print(
            f"\n[{get_theme_color('text_dim')}]Operation cancelled[/{get_theme_color('text_dim')}]"
        )
        return False


def print_welcome():
    """Print the Cyro welcome message."""
    welcome = Panel(
        Text.assemble(
            ("Welcome to ", get_theme_color("text")),
            ("Cyro", f"bold {get_theme_color('primary')}"),
            (" - Terminal-based AI coding agent\n\n", get_theme_color("text")),
            (
                "🤖 Privacy-first AI assistant with local Ollama support\n",
                get_theme_color("success"),
            ),
            (
                "🔧 Dynamic subagent creation through markdown configuration\n",
                get_theme_color("success"),
            ),
            (
                "🚀 Extensible architecture for multiple AI providers\n\n",
                get_theme_color("success"),
            ),
        ),
        title=f"[bold {get_theme_color('primary')}]Cyro AI Coding Agent[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
        padding=(1, 2),
    )
    console.print(welcome)


def print_help():
    """Print help information."""
    help_text = Text.assemble(
        ("Available commands:\n\n", get_theme_color("text")),
        ("Interactive Mode:\n", f"bold {get_theme_color('secondary')}"),
        ("• Enter any prompt for direct execution\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("chat", f"bold {get_theme_color('accent')}"),
        (" - Start interactive chat mode\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("agent list", f"bold {get_theme_color('accent')}"),
        (" - Show available agents\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("config show", f"bold {get_theme_color('accent')}"),
        (" - Show configuration\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("help", f"bold {get_theme_color('accent')}"),
        (" - Show this help\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("exit/quit/q", f"bold {get_theme_color('accent')}"),
        (" - Exit Cyro\n\n", get_theme_color("text")),
        ("Command Line:\n", f"bold {get_theme_color('secondary')}"),
        ("• ", get_theme_color("text")),
        ('cyro "prompt"', f"bold {get_theme_color('accent')}"),
        (" - Execute prompt directly\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ('cyro --agent <name> "prompt"', f"bold {get_theme_color('accent')}"),
        (" - Use specific agent\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("cyro chat", f"bold {get_theme_color('accent')}"),
        (" - Start chat mode\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("cyro agent list", f"bold {get_theme_color('accent')}"),
        (" - List agents\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("cyro config show", f"bold {get_theme_color('accent')}"),
        (" - Show config\n", get_theme_color("text")),
    )

    panel = Panel(
        help_text,
        title=f"[bold {get_theme_color('primary')}]Cyro Help[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
        padding=(1, 2),
    )
    console.print(panel)
