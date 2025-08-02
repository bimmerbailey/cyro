"""
Interactive chat command for Cyro CLI.

This module provides the chat interface for continuous conversation
with AI agents, supporting streaming responses and session management.
"""

import os

import typer
from rich.panel import Panel
from rich.text import Text

from cyro.agents.manager import ManagerAgent
from cyro.cli.shared import (
    ChatCommandResult,
    get_themed_color,
    get_themes_directory,
    process_agent_request,
)
from cyro.config.themes import (
    get_current_theme_name,
    get_theme_color,
    get_theme_info,
    list_themes,
    load_custom_themes,
    set_theme,
    ThemeManager,
)
from cyro.utils.console import (
    console,
    print_error,
    print_info,
    print_success,
    print_warning,
)


def _show_welcome_panels(theme_manager=None):
    """Show welcome and tip panels for chat mode."""

    # Welcome message for chat mode
    chat_welcome = Panel(
        Text.assemble(
            ("🤖 ", get_themed_color("success", theme_manager)),
            ("Welcome to Cyro!", f"bold {get_themed_color('text', theme_manager)}"),
            ("\n\n", get_themed_color("text", theme_manager)),
            ("/help", get_themed_color("info", theme_manager)),
            (" for help, ", get_themed_color("text", theme_manager)),
            ("/status", get_themed_color("info", theme_manager)),
            (" for your current setup\n\n", get_themed_color("text", theme_manager)),
            ("cwd: ", get_themed_color("text", theme_manager)),
            (os.getcwd(), get_themed_color("info", theme_manager)),
        ),
        border_style=get_themed_color("border", theme_manager),
        padding=(0, 1),
    )
    console.print(chat_welcome)

    # Add tip panel
    tip_panel = Panel(
        Text.assemble(
            ("💡 ", get_themed_color("warning", theme_manager)),
            ("Tip: ", f"bold {get_themed_color('text', theme_manager)}"),
            (
                "Create custom slash commands by adding .md files to ",
                get_themed_color("text", theme_manager),
            ),
            (".cyro/commands/", get_themed_color("secondary", theme_manager)),
            (" in your project or ", get_themed_color("text", theme_manager)),
            ("~/.cyro/commands/", get_themed_color("secondary", theme_manager)),
            (
                " for commands that work in any project",
                get_themed_color("text", theme_manager),
            ),
        ),
        border_style=get_themed_color("border", theme_manager),
        padding=(0, 1),
    )
    console.print(tip_panel)


def _create_user_panel(content: str, theme_manager=None) -> Panel:
    """Create a standardized user message panel."""
    return Panel(
        Text(content, style=get_themed_color("text", theme_manager)),
        title=f"[bold {get_themed_color('success', theme_manager)}]You[/bold {get_themed_color('success', theme_manager)}]",
        border_style=get_themed_color("success", theme_manager),
    )


def _create_ai_panel(
    content: str, agent: str | None = None, theme_manager=None
) -> Panel:
    """Create a standardized AI response panel."""
    agent_suffix = f" ({agent})" if agent else ""
    return Panel(
        Text(content, style=get_themed_color("text", theme_manager)),
        title=f"[bold {get_themed_color('primary', theme_manager)}]Cyro{agent_suffix}[/bold {get_themed_color('primary', theme_manager)}]",
        border_style=get_themed_color("primary", theme_manager),
    )


def _run_chat_loop(
    conversation_history: list,
    current_agent: str | None,
    verbose: bool,
    ctx: typer.Context,
):
    """Run the main chat interaction loop."""
    theme_manager: ThemeManager = ctx.obj.theme

    try:
        while True:
            try:
                # Get user input with agent indicator
                agent_indicator = f"[{current_agent}]" if current_agent else "[auto]"
                user_input = console.input(
                    f"\n[bold {get_themed_color('primary', theme_manager)}]you{agent_indicator}>[/bold {get_themed_color('primary', theme_manager)}] "
                ).strip()

                if not user_input:
                    continue

                # Handle chat commands
                if user_input.startswith("/"):
                    command_result = handle_chat_command(
                        user_input,
                        conversation_history,
                        current_agent,
                        verbose,
                        ctx,
                    )

                    if command_result.action == "exit":
                        break
                    elif command_result.action == "clear":
                        conversation_history.clear()
                        print_success("Conversation history cleared.")
                        continue
                    elif command_result.action == "agent_switch":
                        current_agent = command_result.value
                        print_info(f"Switched to agent: {current_agent or 'auto'}")
                        continue
                    else:
                        continue

                # Add user message to history
                conversation_history.append({"role": "user", "content": user_input})

                # Show user message
                console.print(_create_user_panel(user_input, theme_manager))

                # Process message with AI agent
                response = process_chat_message(
                    user_input,
                    conversation_history,
                    current_agent,
                    verbose,
                    ctx,
                )

                # Add AI response to history
                conversation_history.append({"role": "assistant", "content": response})

                # Show AI response
                console.print(_create_ai_panel(response, current_agent, theme_manager))

            except KeyboardInterrupt:
                console.print(
                    f"\n[{get_themed_color('text_dim', theme_manager)}]Use /exit or /quit to leave chat mode[/{get_themed_color('text_dim', theme_manager)}]"
                )
                continue

    except (EOFError, KeyboardInterrupt):
        console.print(
            f"\n[{get_themed_color('text_dim', theme_manager)}]Exiting chat mode...[/{get_themed_color('text_dim', theme_manager)}]"
        )


def start_chat_mode(
    ctx: typer.Context,
    agent: str | None = None,
    verbose: bool = False,
):
    """Start the interactive chat mode."""
    theme_manager = ctx.obj.theme
    _show_welcome_panels(theme_manager)

    # Chat session state
    conversation_history = []
    current_agent = agent

    _run_chat_loop(conversation_history, current_agent, verbose, ctx)

    print_success("Chat session ended.")


def start_chat_mode_with_query(
    initial_query: str,
    agent: str | None = None,
    verbose: bool = False,
    ctx: typer.Context = None,  # type: ignore
):
    """Start chat mode with an initial query."""
    theme_manager = ctx.obj.theme
    _show_welcome_panels(theme_manager)

    # Chat session state
    conversation_history = []
    current_agent = agent

    # Process the initial query
    try:
        # Add initial query to history
        conversation_history.append({"role": "user", "content": initial_query})

        # Show initial query
        console.print(_create_user_panel(initial_query, theme_manager))

        # Process initial message with AI agent
        response = process_chat_message(
            initial_query, conversation_history, current_agent, verbose, ctx
        )

        # Add AI response to history
        conversation_history.append({"role": "assistant", "content": response})

        # Show AI response
        console.print(_create_ai_panel(response, current_agent, theme_manager))

        # Continue with normal chat loop
        _run_chat_loop(conversation_history, current_agent, verbose, ctx)

    except (EOFError, KeyboardInterrupt):
        console.print(
            f"\n[{get_themed_color('text_dim', theme_manager)}]Exiting chat mode...[/{get_themed_color('text_dim', theme_manager)}]"
        )

    print_success("Chat session ended.")


def handle_chat_command(
    command: str,
    history: list,
    current_agent: str | None,
    verbose: bool,
    ctx: typer.Context,
) -> ChatCommandResult:
    """Handle special chat commands."""
    theme_manager = ctx.obj.theme
    cmd_parts = command[1:].split()  # Remove leading '/'

    if not cmd_parts:
        return ChatCommandResult("unknown", error_message="Empty command")

    cmd = cmd_parts[0].lower()

    # TODO: Consider switch statement
    if cmd in ["exit", "quit", "q"]:
        return ChatCommandResult("exit")

    elif cmd == "clear":
        return ChatCommandResult("clear")

    elif cmd == "help":
        show_chat_help(theme_manager)
        return ChatCommandResult("help")

    elif cmd == "agent":
        if len(cmd_parts) > 1:
            new_agent = cmd_parts[1] if cmd_parts[1] != "auto" else None
            return ChatCommandResult("agent_switch", value=new_agent)
        else:
            print_warning("Usage: /agent <name> or /agent auto")
            return ChatCommandResult("error", error_message="Invalid command")

    elif cmd == "history":
        show_conversation_history(history, theme_manager)
        return ChatCommandResult("history")

    elif cmd == "status":
        show_chat_status(current_agent, len(history), theme_manager)
        return ChatCommandResult("status")

    elif cmd == "config":
        if len(cmd_parts) > 1 and cmd_parts[1] == "theme":
            if len(cmd_parts) > 2:
                handle_chat_theme_config(cmd_parts[2], theme_manager)
            else:
                handle_chat_theme_config("list", theme_manager)
            return ChatCommandResult("config")
        else:
            print_warning("Usage: /config theme [list|current|<theme_name>]")
            return ChatCommandResult("error", error_message="Invalid command")

    else:
        print_warning(f"Unknown command: /{cmd}. Type /help for available commands.")
        return ChatCommandResult("unknown", error_message="Empty command")


def show_chat_help(theme_manager=None):
    """Show available chat commands."""
    help_text = Text.assemble(
        ("Chat Commands:\n\n", f"bold {get_themed_color('text', theme_manager)}"),
        ("• ", get_themed_color("text", theme_manager)),
        ("/exit, /quit, /q", f"bold {get_themed_color('secondary', theme_manager)}"),
        (" - Exit chat mode\n", get_themed_color("text", theme_manager)),
        ("• ", get_themed_color("text", theme_manager)),
        ("/clear", f"bold {get_themed_color('secondary', theme_manager)}"),
        (" - Clear conversation history\n", get_themed_color("text", theme_manager)),
        ("• ", get_themed_color("text", theme_manager)),
        ("/help", f"bold {get_themed_color('secondary', theme_manager)}"),
        (" - Show this help\n", get_themed_color("text", theme_manager)),
        ("• ", get_themed_color("text", theme_manager)),
        ("/agent <name>", f"bold {get_themed_color('secondary', theme_manager)}"),
        (" - Switch to specific agent\n", get_themed_color("text", theme_manager)),
        ("• ", get_themed_color("text", theme_manager)),
        ("/agent auto", f"bold {get_themed_color('secondary', theme_manager)}"),
        (
            " - Use automatic agent selection\n",
            get_themed_color("text", theme_manager),
        ),
        ("• ", get_themed_color("text", theme_manager)),
        ("/history", f"bold {get_themed_color('secondary', theme_manager)}"),
        (" - Show conversation history\n", get_themed_color("text", theme_manager)),
        ("• ", get_themed_color("text", theme_manager)),
        ("/status", f"bold {get_themed_color('secondary', theme_manager)}"),
        (" - Show chat session status\n", get_themed_color("text", theme_manager)),
        ("• ", get_themed_color("text", theme_manager)),
        ("/config theme", f"bold {get_themed_color('secondary', theme_manager)}"),
        (
            " - Manage themes (list, current, <name>)\n",
            get_themed_color("text", theme_manager),
        ),
    )

    panel = Panel(
        help_text,
        title=f"[bold {get_themed_color('primary', theme_manager)}]Chat Help[/bold {get_themed_color('primary', theme_manager)}]",
        border_style=get_themed_color("border", theme_manager),
        padding=(1, 2),
    )
    console.print(panel)


def show_conversation_history(history: list, theme_manager=None):
    """Show the conversation history."""
    if not history:
        print_info("No conversation history yet.")
        return

    history_text = Text()
    for i, message in enumerate(history, 1):
        role = "You" if message["role"] == "user" else "Cyro"
        role_style = (
            get_themed_color("success", theme_manager)
            if message["role"] == "user"
            else get_themed_color("primary", theme_manager)
        )

        history_text.append(f"{i}. ", style=get_themed_color("info", theme_manager))
        history_text.append(f"{role}: ", style=f"bold {role_style}")
        history_text.append(
            f"{message['content']}\n\n", style=get_themed_color("text", theme_manager)
        )

    panel = Panel(
        history_text,
        title=f"[bold {get_themed_color('primary', theme_manager)}]Conversation History[/bold {get_themed_color('primary', theme_manager)}]",
        border_style=get_themed_color("border", theme_manager),
    )
    console.print(panel)


def show_chat_status(agent: str | None, message_count: int, theme_manager=None):
    """Show current chat session status."""
    status_text = Text.assemble(
        ("Current Agent: ", get_themed_color("text", theme_manager)),
        (agent or "auto", f"bold {get_themed_color('info', theme_manager)}"),
        ("\nMessages in History: ", get_themed_color("text", theme_manager)),
        (str(message_count), f"bold {get_themed_color('success', theme_manager)}"),
        ("\nSession Status: ", get_themed_color("text", theme_manager)),
        ("Active", f"bold {get_themed_color('success', theme_manager)}"),
    )

    panel = Panel(
        status_text,
        title=f"[bold {get_themed_color('primary', theme_manager)}]Chat Status[/bold {get_themed_color('primary', theme_manager)}]",
        border_style=get_themed_color("border", theme_manager),
    )
    console.print(panel)


def handle_chat_theme_config(action: str, theme_manager: ThemeManager):
    """Handle theme configuration commands in chat mode."""

    if action == "list":
        # Load custom themes first
        themes_dir = get_themes_directory()
        custom_count = load_custom_themes(theme_manager, themes_dir)

        all_themes = list_themes(theme_manager)
        current_theme = get_current_theme_name(theme_manager)

        # Simple list format for chat
        themes_text = Text()
        themes_text.append(
            "Available themes:\n\n", style=get_theme_color("text", theme_manager)
        )

        for theme_name in all_themes:
            is_current = " (current)" if theme_name == current_theme else ""
            themes_text.append(
                f"• {theme_name}{is_current}\n",
                style=get_theme_color("success", theme_manager)
                if is_current
                else get_theme_color("text", theme_manager),
            )

        if custom_count > 0:
            themes_text.append(
                f"\n{custom_count} custom theme{'s' if custom_count != 1 else ''} loaded from {themes_dir}",
                style=get_theme_color("text_dim", theme_manager),
            )

        panel = Panel(
            themes_text,
            title=f"[bold {get_theme_color('primary', theme_manager)}]Themes[/bold {get_theme_color('primary', theme_manager)}]",
            border_style=get_theme_color("border", theme_manager),
        )
        console.print(panel)

    elif action == "current":
        current = get_current_theme_name(theme_manager)
        theme_info = get_theme_info(current, theme_manager)

        if theme_info:
            console.print(
                f"Current theme: [{get_theme_color('primary', theme_manager)}]{theme_info['name']}[/{get_theme_color('primary', theme_manager)}]"
            )
            console.print(
                f"[{get_theme_color('text_dim', theme_manager)}]{theme_info['description']}[/{get_theme_color('text_dim', theme_manager)}]"
            )
        else:
            console.print(
                f"Current theme: [{get_theme_color('primary', theme_manager)}]{current}[/{get_theme_color('primary', theme_manager)}]"
            )
    else:
        # Try to switch to the specified theme
        themes_dir = get_themes_directory()
        custom_count = load_custom_themes(theme_manager, themes_dir)

        if set_theme(theme_manager, action):
            theme_info = get_theme_info(action, theme_manager)
            if theme_info:
                print_success(f"Switched to '{theme_info['name']}' theme")
                console.print(
                    f"[{get_theme_color('text_dim', theme_manager)}]{theme_info['description']}[/{get_theme_color('text_dim', theme_manager)}]"
                )
            else:
                print_success(f"Switched to '{action}' theme")
        else:
            available_themes = list_themes(theme_manager)
            print_error(f"Theme '{action}' not found")
            console.print(
                f"Available: [{get_theme_color('info', theme_manager)}]{', '.join(available_themes)}[/{get_theme_color('info', theme_manager)}]"
            )


def process_chat_message(
    message: str,
    history: list[dict[str, str]],
    agent: str | None,
    verbose: bool,
    ctx: typer.Context,
) -> str:
    """Process a chat message through the AI agent."""
    manager_agent: ManagerAgent = ctx.obj.manager
    theme_manager: ThemeManager = ctx.obj.theme

    if verbose:
        console.print(
            f"[{get_themed_color('text_dim', theme_manager)}]Processing message with agent: {agent or 'auto'}[/{get_themed_color('text_dim', theme_manager)}]"
        )

    try:
        return process_agent_request(message, manager_agent, agent)
    except Exception as e:
        return f"Error: {str(e)}"
