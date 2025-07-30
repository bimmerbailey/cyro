"""
Interactive chat command for Cyro CLI.

This module provides the chat interface for continuous conversation
with AI agents, supporting streaming responses and session management.
"""

from typing import Optional

from rich.panel import Panel
from rich.text import Text

from cyro.utils.console import console, print_info, print_success, print_warning


def start_chat_mode(agent: Optional[str] = None, verbose: bool = False):
    """Start the interactive chat mode."""
    import os
    
    # Welcome message for chat mode
    chat_welcome = Panel(
        Text.assemble(
            ("🤖 ", "bright_green"),
            ("Welcome to Cyro!", "bold bright_white"),
            ("\n\n", "bright_white"),
            ("/help", "yellow"),
            (" for help, ", "bright_white"),
            ("/status", "yellow"),
            (" for your current setup\n\n", "bright_white"),
            ("cwd: ", "bright_white"),
            (os.getcwd(), "yellow"),
        ),
        border_style="yellow",
        padding=(0, 1),
    )
    console.print(chat_welcome)
    
    # Add tip panel
    tip_panel = Panel(
        Text.assemble(
            ("💡 ", "bright_yellow"),
            ("Tip: ", "bold bright_white"),
            ("Create custom slash commands by adding .md files to ", "bright_white"),
            (".cyro/commands/", "orange"),
            (" in your project or ", "bright_white"),
            ("~/.cyro/commands/", "orange"), 
            (" for commands that work in any project", "bright_white"),
        ),
        border_style="yellow",
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
                    Text(user_input, style="bright_white"),
                    title="[bold bright_green]You[/bold bright_green]",
                    border_style="bright_green",
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


def start_chat_mode_with_query(initial_query: str, agent: Optional[str] = None, verbose: bool = False):
    """Start chat mode with an initial query."""
    import os
    
    # Welcome message for chat mode
    chat_welcome = Panel(
        Text.assemble(
            ("🤖 ", "bright_green"),
            ("Welcome to Cyro!", "bold bright_white"),
            ("\n\n", "bright_white"),
            ("/help", "yellow"),
            (" for help, ", "bright_white"),
            ("/status", "yellow"),
            (" for your current setup\n\n", "bright_white"),
            ("cwd: ", "bright_white"),
            (os.getcwd(), "yellow"),
        ),
        border_style="yellow",
        padding=(0, 1),
    )
    console.print(chat_welcome)
    
    # Add tip panel
    tip_panel = Panel(
        Text.assemble(
            ("💡 ", "bright_yellow"),
            ("Tip: ", "bold bright_white"),
            ("Create custom slash commands by adding .md files to ", "bright_white"),
            (".cyro/commands/", "orange"),
            (" in your project or ", "bright_white"),
            ("~/.cyro/commands/", "orange"), 
            (" for commands that work in any project", "bright_white"),
        ),
        border_style="yellow",
        padding=(0, 1),
    )
    console.print(tip_panel)

    # Chat session state
    conversation_history = []
    current_agent = agent

    # Process the initial query
    try:
        # Add initial query to history
        conversation_history.append({"role": "user", "content": initial_query})

        # Show initial query
        user_panel = Panel(
            Text(initial_query, style="bright_white"),
            title="[bold bright_green]You[/bold bright_green]",
            border_style="bright_green",
        )
        console.print(user_panel)

        # Process initial message with AI agent
        response = process_chat_message(
            initial_query, conversation_history, current_agent, verbose
        )

        # Add AI response to history
        conversation_history.append({"role": "assistant", "content": response})

        # Show AI response
        ai_panel = Panel(
            Text(response, style="bright_white"),
            title=f"[bold yellow]Cyro{f' ({current_agent})' if current_agent else ''}[/bold yellow]",
            border_style="yellow",
        )
        console.print(ai_panel)
        
        # Continue with normal chat loop
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
                        Text(response, style="bright_white"),
                        title=f"[bold yellow]Cyro{f' ({current_agent})' if current_agent else ''}[/bold yellow]",
                        border_style="yellow",
                    )
                    console.print(ai_panel)

                except KeyboardInterrupt:
                    console.print("\n[dim]Use /exit or /quit to leave chat mode[/dim]")
                    continue

        except (EOFError, KeyboardInterrupt):
            console.print("\n[dim]Exiting chat mode...[/dim]")

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
        ("Chat Commands:\n\n", "bold bright_white"),
        ("• ", "bright_white"),
        ("/exit, /quit, /q", "bold orange"),
        (" - Exit chat mode\n", "bright_white"),
        ("• ", "bright_white"),
        ("/clear", "bold orange"),
        (" - Clear conversation history\n", "bright_white"),
        ("• ", "bright_white"),
        ("/help", "bold orange"),
        (" - Show this help\n", "bright_white"),
        ("• ", "bright_white"),
        ("/agent <name>", "bold orange"),
        (" - Switch to specific agent\n", "bright_white"),
        ("• ", "bright_white"),
        ("/agent auto", "bold orange"),
        (" - Use automatic agent selection\n", "bright_white"),
        ("• ", "bright_white"),
        ("/history", "bold orange"),
        (" - Show conversation history\n", "bright_white"),
        ("• ", "bright_white"),
        ("/status", "bold orange"),
        (" - Show chat session status\n", "bright_white"),
    )

    panel = Panel(
        help_text,
        title="[bold yellow]Chat Help[/bold yellow]",
        border_style="yellow",
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
        role_style = "bright_green" if message["role"] == "user" else "yellow"

        history_text.append(f"{i}. ", style="yellow")
        history_text.append(f"{role}: ", style=f"bold {role_style}")
        history_text.append(f"{message['content']}\n\n", style="bright_white")

    panel = Panel(
        history_text,
        title="[bold yellow]Conversation History[/bold yellow]",
        border_style="yellow",
    )
    console.print(panel)


def show_chat_status(agent: Optional[str], message_count: int):
    """Show current chat session status."""
    status_text = Text.assemble(
        ("Current Agent: ", "bright_white"),
        (agent or "auto", "bold yellow"),
        ("\nMessages in History: ", "bright_white"),
        (str(message_count), "bold bright_green"),
        ("\nSession Status: ", "bright_white"),
        ("Active", "bold bright_green"),
    )

    panel = Panel(
        status_text,
        title="[bold yellow]Chat Status[/bold yellow]",
        border_style="yellow",
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


