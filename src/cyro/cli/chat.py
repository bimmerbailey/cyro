"""
Interactive chat command for Cyro CLI.

This module provides the chat interface for continuous conversation
with AI agents, supporting streaming responses and session management.
"""

from typing import Optional

import typer
from rich.panel import Panel
from rich.text import Text

from cyro.utils.console import console, print_info, print_success, print_warning

# Create chat subcommand app
chat_app = typer.Typer(
    name="chat",
    help="Interactive chat commands",
    rich_markup_mode="rich",
)


@chat_app.command()
def start(
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to chat with"
    ),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Enable verbose output"
    ),
):
    """Start an interactive chat session."""
    if verbose:
        console.print(f"[dim]Starting chat with agent: {agent or 'auto'}[/dim]")

    start_chat_mode(agent, verbose)


def start_chat_mode(agent: Optional[str] = None, verbose: bool = False):
    """Start the interactive chat mode."""
    import os
    
    # Welcome message for chat mode
    chat_welcome = Panel(
        Text.assemble(
            ("🤖 ", "bright_green"),
            ("Welcome to Cyro!", "bold bright_white"),
            ("\n\n", "white"),
            ("/help", "dim white"),
            (" for help, ", "dim white"),
            ("/status", "dim white"),
            (" for your current setup\n\n", "dim white"),
            ("cwd: ", "dim white"),
            (os.getcwd(), "dim white"),
        ),
        border_style="bright_white",
        padding=(0, 1),
    )
    console.print(chat_welcome)
    
    # Add tip panel
    tip_panel = Panel(
        Text.assemble(
            ("💡 ", "bright_yellow"),
            ("Tip: ", "bold bright_white"),
            ("Create custom slash commands by adding .md files to ", "white"),
            (".cyro/commands/", "bright_cyan"),
            (" in your project or ", "white"),
            ("~/.cyro/commands/", "bright_cyan"), 
            (" for commands that work in any project", "white"),
        ),
        border_style="bright_white",
        padding=(0, 1),
    )
    console.print(tip_panel)

    # Chat session state
    conversation_history = []
    current_agent = agent

    try:
        while True:
            try:
                # Get user input with agent indicator
                agent_indicator = f"[{current_agent}]" if current_agent else "[auto]"
                user_input = console.input(
                    f"\n[bold blue]you{agent_indicator}>[/bold blue] "
                ).strip()

                if not user_input:
                    continue

                # Handle chat commands
                if user_input.startswith("/"):
                    command_result = handle_chat_command(
                        user_input, conversation_history, current_agent, verbose
                    )

                    if command_result == "exit":
                        break
                    elif command_result == "clear":
                        conversation_history.clear()
                        print_success("Conversation history cleared.")
                        continue
                    elif command_result.startswith("agent:"):
                        current_agent = command_result.split(":", 1)[1]
                        print_info(f"Switched to agent: {current_agent or 'auto'}")
                        continue
                    else:
                        continue

                # Add user message to history
                conversation_history.append({"role": "user", "content": user_input})

                # Show user message
                user_panel = Panel(
                    Text(user_input, style="white"),
                    title="[bold green]You[/bold green]",
                    border_style="green",
                )
                console.print(user_panel)

                # Process message with AI agent
                response = process_chat_message(
                    user_input, conversation_history, current_agent, verbose
                )

                # Add AI response to history
                conversation_history.append({"role": "assistant", "content": response})

                # Show AI response
                ai_panel = Panel(
                    Text(response, style="white"),
                    title=f"[bold blue]Cyro{f' ({current_agent})' if current_agent else ''}[/bold blue]",
                    border_style="blue",
                )
                console.print(ai_panel)

            except KeyboardInterrupt:
                console.print("\n[dim]Use /exit or /quit to leave chat mode[/dim]")
                continue

    except (EOFError, KeyboardInterrupt):
        console.print("\n[dim]Exiting chat mode...[/dim]")

    print_success("Chat session ended.")


def handle_chat_command(
    command: str, history: list, current_agent: Optional[str], verbose: bool
) -> str:
    """Handle special chat commands."""
    cmd_parts = command[1:].split()  # Remove leading '/'

    if not cmd_parts:
        return "unknown"

    cmd = cmd_parts[0].lower()

    if cmd in ["exit", "quit", "q"]:
        return "exit"

    elif cmd == "clear":
        return "clear"

    elif cmd == "help":
        show_chat_help()
        return "help"

    elif cmd == "agent":
        if len(cmd_parts) > 1:
            new_agent = cmd_parts[1] if cmd_parts[1] != "auto" else None
            return f"agent:{new_agent}"
        else:
            print_warning("Usage: /agent <name> or /agent auto")
            return "error"

    elif cmd == "history":
        show_conversation_history(history)
        return "history"

    elif cmd == "status":
        show_chat_status(current_agent, len(history))
        return "status"

    else:
        print_warning(f"Unknown command: /{cmd}. Type /help for available commands.")
        return "unknown"


def show_chat_help():
    """Show available chat commands."""
    help_text = Text.assemble(
        ("Chat Commands:\n\n", "bold white"),
        ("• ", "white"),
        ("/exit, /quit, /q", "bold cyan"),
        (" - Exit chat mode\n", "white"),
        ("• ", "white"),
        ("/clear", "bold cyan"),
        (" - Clear conversation history\n", "white"),
        ("• ", "white"),
        ("/help", "bold cyan"),
        (" - Show this help\n", "white"),
        ("• ", "white"),
        ("/agent <name>", "bold cyan"),
        (" - Switch to specific agent\n", "white"),
        ("• ", "white"),
        ("/agent auto", "bold cyan"),
        (" - Use automatic agent selection\n", "white"),
        ("• ", "white"),
        ("/history", "bold cyan"),
        (" - Show conversation history\n", "white"),
        ("• ", "white"),
        ("/status", "bold cyan"),
        (" - Show chat session status\n", "white"),
    )

    panel = Panel(
        help_text,
        title="[bold blue]Chat Help[/bold blue]",
        border_style="blue",
        padding=(1, 2),
    )
    console.print(panel)


def show_conversation_history(history: list):
    """Show the conversation history."""
    if not history:
        print_info("No conversation history yet.")
        return

    history_text = Text()
    for i, message in enumerate(history, 1):
        role = "You" if message["role"] == "user" else "Cyro"
        role_style = "green" if message["role"] == "user" else "blue"

        history_text.append(f"{i}. ", style="dim")
        history_text.append(f"{role}: ", style=f"bold {role_style}")
        history_text.append(f"{message['content']}\n\n", style="white")

    panel = Panel(
        history_text,
        title="[bold blue]Conversation History[/bold blue]",
        border_style="blue",
    )
    console.print(panel)


def show_chat_status(agent: Optional[str], message_count: int):
    """Show current chat session status."""
    status_text = Text.assemble(
        ("Current Agent: ", "white"),
        (agent or "auto", "bold blue"),
        ("\nMessages in History: ", "white"),
        (str(message_count), "bold green"),
        ("\nSession Status: ", "white"),
        ("Active", "bold green"),
    )

    panel = Panel(
        status_text,
        title="[bold blue]Chat Status[/bold blue]",
        border_style="blue",
    )
    console.print(panel)


def process_chat_message(
    message: str, history: list, agent: Optional[str], verbose: bool
) -> str:
    """Process a chat message through the AI agent."""
    if verbose:
        console.print(f"[dim]Processing message with agent: {agent or 'auto'}[/dim]")

    # TODO: Implement actual AI agent processing
    return f"🚧 AI processing not yet implemented.\n\nReceived: '{message}'"


# Default command when just running 'cyro chat'
@chat_app.callback(invoke_without_command=True)
def chat_main(
    ctx: typer.Context,
    agent: Optional[str] = typer.Option(
        None, "--agent", "-a", help="Specify which agent to chat with"
    ),
    verbose: bool = typer.Option(
        False, "--verbose", "-v", help="Enable verbose output"
    ),
):
    """Start interactive chat mode."""
    if ctx.invoked_subcommand is None:
        start_chat_mode(agent, verbose)
