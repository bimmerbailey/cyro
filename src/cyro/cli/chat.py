"""
Interactive chat command for Cyro CLI.

This module provides the chat interface for continuous conversation
with AI agents, supporting streaming responses and session management.
"""

import os
from typing import Optional

from rich.panel import Panel
from rich.text import Text

from cyro.config.themes import get_theme_color
from cyro.utils.console import console, print_info, print_success, print_warning


def _show_welcome_panels():
    """Show welcome and tip panels for chat mode."""
    # Welcome message for chat mode
    chat_welcome = Panel(
        Text.assemble(
            ("🤖 ", get_theme_color("success")),
            ("Welcome to Cyro!", f"bold {get_theme_color('text')}"),
            ("\n\n", get_theme_color("text")),
            ("/help", get_theme_color("info")),
            (" for help, ", get_theme_color("text")),
            ("/status", get_theme_color("info")),
            (" for your current setup\n\n", get_theme_color("text")),
            ("cwd: ", get_theme_color("text")),
            (os.getcwd(), get_theme_color("info")),
        ),
        border_style=get_theme_color("border"),
        padding=(0, 1),
    )
    console.print(chat_welcome)
    
    # Add tip panel
    tip_panel = Panel(
        Text.assemble(
            ("💡 ", get_theme_color("warning")),
            ("Tip: ", f"bold {get_theme_color('text')}"),
            ("Create custom slash commands by adding .md files to ", get_theme_color("text")),
            (".cyro/commands/", get_theme_color("secondary")),
            (" in your project or ", get_theme_color("text")),
            ("~/.cyro/commands/", get_theme_color("secondary")), 
            (" for commands that work in any project", get_theme_color("text")),
        ),
        border_style=get_theme_color("border"),
        padding=(0, 1),
    )
    console.print(tip_panel)


def _create_user_panel(content: str) -> Panel:
    """Create a standardized user message panel."""
    return Panel(
        Text(content, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('success')}]You[/bold {get_theme_color('success')}]",
        border_style=get_theme_color("success"),
    )


def _create_ai_panel(content: str, agent: Optional[str] = None) -> Panel:
    """Create a standardized AI response panel."""
    agent_suffix = f" ({agent})" if agent else ""
    return Panel(
        Text(content, style=get_theme_color("text")),
        title=f"[bold {get_theme_color('primary')}]Cyro{agent_suffix}[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("primary"),
    )


def _run_chat_loop(conversation_history: list, current_agent: Optional[str], verbose: bool):
    """Run the main chat interaction loop."""
    try:
        while True:
            try:
                # Get user input with agent indicator
                agent_indicator = f"[{current_agent}]" if current_agent else "[auto]"
                user_input = console.input(
                    f"\n[bold {get_theme_color('primary')}]you{agent_indicator}>[/bold {get_theme_color('primary')}] "
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
                console.print(_create_user_panel(user_input))

                # Process message with AI agent
                response = process_chat_message(
                    user_input, conversation_history, current_agent, verbose
                )

                # Add AI response to history
                conversation_history.append({"role": "assistant", "content": response})

                # Show AI response
                console.print(_create_ai_panel(response, current_agent))

            except KeyboardInterrupt:
                console.print(f"\n[{get_theme_color('text_dim')}]Use /exit or /quit to leave chat mode[/{get_theme_color('text_dim')}]")
                continue

    except (EOFError, KeyboardInterrupt):
        console.print(f"\n[{get_theme_color('text_dim')}]Exiting chat mode...[/{get_theme_color('text_dim')}]")


def start_chat_mode(agent: Optional[str] = None, verbose: bool = False):
    """Start the interactive chat mode."""
    _show_welcome_panels()

    # Chat session state
    conversation_history = []
    current_agent = agent

    _run_chat_loop(conversation_history, current_agent, verbose)

    print_success("Chat session ended.")


def start_chat_mode_with_query(initial_query: str, agent: Optional[str] = None, verbose: bool = False):
    """Start chat mode with an initial query."""
    _show_welcome_panels()

    # Chat session state
    conversation_history = []
    current_agent = agent

    # Process the initial query
    try:
        # Add initial query to history
        conversation_history.append({"role": "user", "content": initial_query})

        # Show initial query
        console.print(_create_user_panel(initial_query))

        # Process initial message with AI agent
        response = process_chat_message(
            initial_query, conversation_history, current_agent, verbose
        )

        # Add AI response to history
        conversation_history.append({"role": "assistant", "content": response})

        # Show AI response
        console.print(_create_ai_panel(response, current_agent))
        
        # Continue with normal chat loop
        _run_chat_loop(conversation_history, current_agent, verbose)

    except (EOFError, KeyboardInterrupt):
        console.print(f"\n[{get_theme_color('text_dim')}]Exiting chat mode...[/{get_theme_color('text_dim')}]")

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

    elif cmd == "config":
        if len(cmd_parts) > 1 and cmd_parts[1] == "theme":
            if len(cmd_parts) > 2:
                handle_chat_theme_config(cmd_parts[2])
            else:
                handle_chat_theme_config("list")
            return "config"
        else:
            print_warning("Usage: /config theme [list|current|<theme_name>]")
            return "error"

    else:
        print_warning(f"Unknown command: /{cmd}. Type /help for available commands.")
        return "unknown"


def show_chat_help():
    """Show available chat commands."""
    help_text = Text.assemble(
        ("Chat Commands:\n\n", f"bold {get_theme_color('text')}"),
        ("• ", get_theme_color("text")),
        ("/exit, /quit, /q", f"bold {get_theme_color('secondary')}"),
        (" - Exit chat mode\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/clear", f"bold {get_theme_color('secondary')}"),
        (" - Clear conversation history\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/help", f"bold {get_theme_color('secondary')}"),
        (" - Show this help\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/agent <name>", f"bold {get_theme_color('secondary')}"),
        (" - Switch to specific agent\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/agent auto", f"bold {get_theme_color('secondary')}"),
        (" - Use automatic agent selection\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/history", f"bold {get_theme_color('secondary')}"),
        (" - Show conversation history\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/status", f"bold {get_theme_color('secondary')}"),
        (" - Show chat session status\n", get_theme_color("text")),
        ("• ", get_theme_color("text")),
        ("/config theme", f"bold {get_theme_color('secondary')}"),
        (" - Manage themes (list, current, <name>)\n", get_theme_color("text")),
    )

    panel = Panel(
        help_text,
        title=f"[bold {get_theme_color('primary')}]Chat Help[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
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
        role_style = get_theme_color("success") if message["role"] == "user" else get_theme_color("primary")

        history_text.append(f"{i}. ", style=get_theme_color("info"))
        history_text.append(f"{role}: ", style=f"bold {role_style}")
        history_text.append(f"{message['content']}\n\n", style=get_theme_color("text"))

    panel = Panel(
        history_text,
        title=f"[bold {get_theme_color('primary')}]Conversation History[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
    )
    console.print(panel)


def show_chat_status(agent: Optional[str], message_count: int):
    """Show current chat session status."""
    status_text = Text.assemble(
        ("Current Agent: ", get_theme_color("text")),
        (agent or "auto", f"bold {get_theme_color('info')}"),
        ("\nMessages in History: ", get_theme_color("text")),
        (str(message_count), f"bold {get_theme_color('success')}"),
        ("\nSession Status: ", get_theme_color("text")),
        ("Active", f"bold {get_theme_color('success')}"),
    )

    panel = Panel(
        status_text,
        title=f"[bold {get_theme_color('primary')}]Chat Status[/bold {get_theme_color('primary')}]",
        border_style=get_theme_color("border"),
    )
    console.print(panel)


def handle_chat_theme_config(action: str):
    """Handle theme configuration commands in chat mode."""
    from cyro.config.themes import list_themes, get_current_theme_name, set_theme, get_theme_info, load_custom_themes
    from cyro.utils.console import print_info, print_success, print_error
    
    if action == "list":
        # Load custom themes first
        themes_dir = "~/.cyro/themes"
        custom_count = load_custom_themes(themes_dir)
        
        all_themes = list_themes()
        current_theme = get_current_theme_name()
        
        # Simple list format for chat
        themes_text = Text()
        themes_text.append("Available themes:\n\n", style=get_theme_color("text"))
        
        for theme_name in all_themes:
            is_current = " (current)" if theme_name == current_theme else ""
            themes_text.append(f"• {theme_name}{is_current}\n", 
                             style=get_theme_color("success") if is_current else get_theme_color("text"))
        
        if custom_count > 0:
            themes_text.append(f"\n{custom_count} custom theme{'s' if custom_count != 1 else ''} loaded from {themes_dir}", 
                             style=get_theme_color("text_dim"))
        
        panel = Panel(
            themes_text,
            title=f"[bold {get_theme_color('primary')}]Themes[/bold {get_theme_color('primary')}]",
            border_style=get_theme_color("border"),
        )
        console.print(panel)
        
    elif action == "current":
        current = get_current_theme_name()
        theme_info = get_theme_info(current)
        
        if theme_info:
            console.print(f"Current theme: [{get_theme_color('primary')}]{theme_info['name']}[/{get_theme_color('primary')}]")
            console.print(f"[{get_theme_color('text_dim')}]{theme_info['description']}[/{get_theme_color('text_dim')}]")
        else:
            console.print(f"Current theme: [{get_theme_color('primary')}]{current}[/{get_theme_color('primary')}]")
    else:
        # Try to switch to the specified theme
        themes_dir = "~/.cyro/themes"
        custom_count = load_custom_themes(themes_dir)
        
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
            console.print(f"Available: [{get_theme_color('info')}]{', '.join(available_themes)}[/{get_theme_color('info')}]")


def process_chat_message(
    message: str, history: list, agent: Optional[str], verbose: bool
) -> str:
    """Process a chat message through the AI agent."""
    if verbose:
        console.print(f"[{get_theme_color('text_dim')}]Processing message with agent: {agent or 'auto'}[/{get_theme_color('text_dim')}]")

    # TODO: Implement actual AI agent processing
    return f"🚧 AI processing not yet implemented.\n\nReceived: '{message}'"


