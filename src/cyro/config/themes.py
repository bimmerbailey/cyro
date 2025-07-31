"""
Theme models and default themes for Cyro CLI.

This module defines the theme system using Pydantic models with built-in
default themes that work out of the box without any configuration.

The theme system provides semantic color names that map to Rich color strings,
allowing consistent styling across the entire application while supporting
easy theme switching.

Usage:
    from cyro.config.themes import get_theme_color, set_theme, list_themes, get_current_theme_name

    # Primary interface - drop-in replacement for hardcoded colors
    panel = Panel(title=f"[bold {get_theme_color('primary')}]Title[/]")
    console.print(f"[{get_theme_color('success')}]Success![/]")

    # Theme management
    set_theme("dark")
    available_themes = list_themes()
    current_theme = get_current_theme_name()

    # Advanced usage - direct access to models
    from cyro.config.themes import DEFAULT_THEMES, Theme, ColorScheme, ThemeManager
    cyro_theme = DEFAULT_THEMES["cyro"]
    theme_manager = ThemeManager()
"""

from pathlib import Path
from typing import Dict, List, Optional

import tomllib
from pydantic import BaseModel, Field, ValidationError


class ColorScheme(BaseModel):
    """
    Semantic color scheme for Cyro CLI theming.

    All colors should be valid Rich color strings (e.g., "yellow", "#FFFF00",
    "bright_green", etc.). This model ensures every UI element has a semantic
    color mapping for consistent theming.
    """

    # Primary brand colors
    primary: str = Field(description="Primary brand color (main UI elements)")
    secondary: str = Field(description="Secondary accent color")
    accent: str = Field(description="Accent color for highlights")

    # Status colors for different message types
    success: str = Field(description="Success messages and positive feedback")
    warning: str = Field(description="Warning messages and cautions")
    error: str = Field(description="Error messages and failures")
    info: str = Field(description="Informational messages and tips")

    # UI element colors
    text: str = Field(description="Primary text color")
    text_dim: str = Field(description="Dimmed/secondary text color")
    background: str = Field(description="Background color for panels")
    border: str = Field(description="Border colors for panels and tables")
    highlight: str = Field(description="Highlighted text and selections")

    # Code and syntax colors
    code_bg: str = Field(description="Code block background")
    code_text: str = Field(description="Code block text")
    syntax_keyword: str = Field(description="Syntax highlighting for keywords")
    syntax_string: str = Field(description="Syntax highlighting for strings")
    syntax_comment: str = Field(description="Syntax highlighting for comments")

    # Table and data colors
    table_header: str = Field(description="Table header styling")
    table_row: str = Field(description="Table row text color")

    # Interactive elements
    prompt: str = Field(description="User input prompts")
    input_default: str = Field(description="Default values in prompts")


class Theme(BaseModel):
    """
    Complete theme definition with metadata and color scheme.

    A theme combines a descriptive name and color scheme into a complete
    theming solution that can be easily switched and applied across the CLI.
    """

    name: str = Field(description="Unique theme identifier")
    description: str = Field(description="Human-readable theme description")
    colors: ColorScheme = Field(description="Complete color scheme for this theme")


# Default "cyro" theme - maintains current yellow/orange visual appearance
_CYRO_COLORS = ColorScheme(
    # Primary brand colors (current yellow/orange scheme)
    primary="yellow",
    secondary="orange",
    accent="bright_yellow",
    # Status colors (from current console.py)
    success="bright_green",
    warning="bright_yellow",
    error="bright_red",
    info="yellow",
    # UI colors
    text="bright_white",
    text_dim="dim",
    background="default",
    border="yellow",
    highlight="bold yellow",
    # Code colors
    code_bg="default",
    code_text="bright_white",
    syntax_keyword="bold bright_blue",
    syntax_string="bright_green",
    syntax_comment="dim",
    # Table colors
    table_header="bold yellow",
    table_row="bright_white",
    # Interactive colors
    prompt="bold yellow",
    input_default="yellow",
)

# Classic terminal theme with traditional blue/cyan colors
_CLASSIC_COLORS = ColorScheme(
    primary="bright_blue",
    secondary="cyan",
    accent="bright_cyan",
    success="bright_green",
    warning="bright_yellow",
    error="bright_red",
    info="bright_blue",
    text="bright_white",
    text_dim="dim",
    background="default",
    border="bright_blue",
    highlight="bold bright_blue",
    code_bg="default",
    code_text="bright_white",
    syntax_keyword="bold bright_blue",
    syntax_string="bright_green",
    syntax_comment="dim",
    table_header="bold bright_blue",
    table_row="bright_white",
    prompt="bold bright_blue",
    input_default="cyan",
)

# Dark theme with muted colors for comfortable dark terminal use
_DARK_COLORS = ColorScheme(
    primary="#6B7280",  # Gray-500
    secondary="#4B5563",  # Gray-600
    accent="#9CA3AF",  # Gray-400
    success="#10B981",  # Emerald-500
    warning="#F59E0B",  # Amber-500
    error="#EF4444",  # Red-500
    info="#3B82F6",  # Blue-500
    text="#F3F4F6",  # Gray-100
    text_dim="#6B7280",  # Gray-500
    background="default",
    border="#4B5563",  # Gray-600
    highlight="#9CA3AF",  # Gray-400
    code_bg="default",
    code_text="#F3F4F6",  # Gray-100
    syntax_keyword="#8B5CF6",  # Violet-500
    syntax_string="#10B981",  # Emerald-500
    syntax_comment="#6B7280",  # Gray-500
    table_header="#9CA3AF",  # Gray-400
    table_row="#F3F4F6",  # Gray-100
    prompt="#9CA3AF",  # Gray-400
    input_default="#6B7280",  # Gray-500
)

# Light theme with high contrast for light terminals
_LIGHT_COLORS = ColorScheme(
    primary="#1E40AF",  # Blue-800
    secondary="#7C3AED",  # Violet-600
    accent="#059669",  # Emerald-600
    success="#065F46",  # Emerald-800
    warning="#D97706",  # Amber-600
    error="#DC2626",  # Red-600
    info="#1D4ED8",  # Blue-700
    text="#111827",  # Gray-900
    text_dim="#6B7280",  # Gray-500
    background="default",
    border="#374151",  # Gray-700
    highlight="#1F2937",  # Gray-800
    code_bg="default",
    code_text="#111827",  # Gray-900
    syntax_keyword="#7C3AED",  # Violet-600
    syntax_string="#065F46",  # Emerald-800
    syntax_comment="#6B7280",  # Gray-500
    table_header="#374151",  # Gray-700
    table_row="#111827",  # Gray-900
    prompt="#1E40AF",  # Blue-800
    input_default="#6B7280",  # Gray-500
)

# Built-in default themes dictionary
DEFAULT_THEMES: Dict[str, Theme] = {
    "cyro": Theme(
        name="cyro",
        description="Default Cyro theme with warm yellow and orange colors",
        colors=_CYRO_COLORS,
    ),
    "classic": Theme(
        name="classic",
        description="Traditional terminal theme with blue and cyan colors",
        colors=_CLASSIC_COLORS,
    ),
    "dark": Theme(
        name="dark",
        description="Muted theme optimized for dark terminals",
        colors=_DARK_COLORS,
    ),
    "light": Theme(
        name="light",
        description="High contrast theme for light terminals",
        colors=_LIGHT_COLORS,
    ),
}


class ThemeManager:
    """
    Theme manager for Cyro CLI with safe defaults and optional custom themes.

    The ThemeManager provides a centralized way to access theme colors with
    automatic fallbacks and seamless theme switching. It works perfectly
    without any configuration and supports optional custom theme loading.

    Features:
    - Zero configuration required - works with built-in themes
    - Always returns valid Rich color strings
    - Seamless theme switching
    - Safe custom theme loading with fallbacks
    - Current theme state management
    """

    def __init__(self, default_theme: str = "cyro"):
        """
        Initialize ThemeManager with built-in themes.

        Args:
            default_theme: Name of the default theme to use
        """
        self._themes: Dict[str, Theme] = DEFAULT_THEMES.copy()
        self._current_theme_name: str = default_theme

        # Ensure default theme exists
        if default_theme not in self._themes:
            self._current_theme_name = "cyro"

    def get_color(self, semantic_name: str) -> str:
        """
        Get a color by semantic name from the current theme.

        Always returns a valid Rich color string. If the color is not found
        in the current theme, falls back to the cyro theme, and if still not
        found, returns a safe default color.

        Args:
            semantic_name: Semantic color name (e.g., "primary", "success")

        Returns:
            Rich color string that is guaranteed to be valid
        """
        current_theme = self._themes.get(self._current_theme_name)
        
        # Try current theme first (if it's not already cyro)
        if (
            current_theme
            and current_theme.name != "cyro"
            and hasattr(current_theme.colors, semantic_name)
        ):
            color = getattr(current_theme.colors, semantic_name)
            if color:
                return color

        # Fallback to cyro theme (guaranteed to exist in DEFAULT_THEMES)
        cyro_theme = DEFAULT_THEMES["cyro"]
        if hasattr(cyro_theme.colors, semantic_name):
            return getattr(cyro_theme.colors, semantic_name)

        # Ultimate fallback if semantic name doesn't exist in any theme
        return "bright_white"

    def get_current_theme(self) -> Theme:
        """
        Get the currently active theme.

        Returns:
            The current Theme object
        """
        return self._themes.get(self._current_theme_name, self._themes["cyro"])

    def get_current_theme_name(self) -> str:
        """
        Get the name of the currently active theme.

        Returns:
            Name of the current theme
        """
        return self._current_theme_name

    def set_theme(self, theme_name: str) -> bool:
        """
        Switch to a different theme.

        Args:
            theme_name: Name of the theme to switch to

        Returns:
            True if theme was successfully switched, False if theme not found
        """
        if theme_name in self._themes:
            self._current_theme_name = theme_name
            return True
        return False

    def list_themes(self) -> List[str]:
        """
        Get a list of all available theme names.

        Returns:
            List of theme names sorted alphabetically
        """
        return sorted(self._themes.keys())

    def get_theme_info(self, theme_name: str) -> Optional[Dict[str, str]]:
        """
        Get information about a specific theme.

        Args:
            theme_name: Name of the theme

        Returns:
            Dictionary with theme info (name, description) or None if not found
        """
        theme = self._themes.get(theme_name)
        if theme:
            return {"name": theme.name, "description": theme.description}
        return None

    def load_custom_themes(self, themes_dir: str) -> int:
        """
        Load custom themes from TOML files in a directory.

        This is a fail-safe operation - if any theme fails to load, it is
        skipped and the operation continues. The built-in themes are never
        affected by custom theme loading failures.

        Args:
            themes_dir: Directory path containing .toml theme files

        Returns:
            Number of custom themes successfully loaded
        """
        if not themes_dir:
            return 0

        themes_path = Path(themes_dir).expanduser()
        if not themes_path.exists() or not themes_path.is_dir():
            return 0

        loaded_count = 0

        try:
            for theme_file in themes_path.glob("*.toml"):
                try:
                    with open(theme_file, "rb") as f:
                        theme_data = tomllib.load(f)

                    # Validate theme structure
                    if "name" not in theme_data or "colors" not in theme_data:
                        continue

                    # Create theme objects with validation
                    colors = ColorScheme(**theme_data["colors"])
                    theme = Theme(
                        name=theme_data["name"],
                        description=theme_data.get(
                            "description", f"Custom theme: {theme_data['name']}"
                        ),
                        colors=colors,
                    )

                    # Add to available themes (don't override built-ins)
                    if theme.name not in DEFAULT_THEMES:
                        self._themes[theme.name] = theme
                        loaded_count += 1

                except (OSError, ValidationError, KeyError, tomllib.TOMLDecodeError):
                    # Skip invalid theme files silently
                    continue

        except OSError:
            # Directory access issues - fail silently
            pass

        return loaded_count

    def reset_to_defaults(self) -> None:
        """
        Reset theme manager to only include built-in themes.

        This removes all custom themes and resets to the default theme.
        """
        self._themes = DEFAULT_THEMES.copy()
        if self._current_theme_name not in self._themes:
            self._current_theme_name = "cyro"


def create_theme_manager(default_theme: str = "cyro") -> ThemeManager:
    """Create a new ThemeManager instance."""
    return ThemeManager(default_theme)


def get_theme_color(
    semantic_name: str, theme_manager: Optional[ThemeManager] = None
) -> str:
    """
    Get a theme color by semantic name.

    Args:
        semantic_name: Semantic color name (e.g., "primary", "success", "error")
        theme_manager: Optional ThemeManager instance. If None, uses cyro theme defaults.

    Returns:
        Rich color string that is guaranteed to be valid

    Examples:
        # With theme manager (recommended)
        tm = create_theme_manager()
        panel = Panel(title=f"[bold {get_theme_color('primary', tm)}]Title[/]")

        # Without theme manager (uses cyro defaults)
        console.print(f"[{get_theme_color('success')}]Success![/]")
    """
    if theme_manager is None:
        # Fallback to cyro theme colors directly
        cyro_theme = DEFAULT_THEMES["cyro"]
        if hasattr(cyro_theme.colors, semantic_name):
            return getattr(cyro_theme.colors, semantic_name)
        return "bright_white"

    return theme_manager.get_color(semantic_name)


# Convenience functions that work with ThemeManager instances
def set_theme(theme_manager: ThemeManager, theme_name: str) -> bool:
    """
    Switch to a different theme on a specific theme manager.

    Args:
        theme_manager: ThemeManager instance to modify
        theme_name: Name of the theme to switch to

    Returns:
        True if theme was successfully switched, False if theme not found

    Example:
        tm = create_theme_manager()
        if set_theme(tm, "dark"):
            print("Switched to dark theme")
    """
    return theme_manager.set_theme(theme_name)


def get_current_theme_name(theme_manager: ThemeManager) -> str:
    """
    Get the name of the currently active theme from a theme manager.

    Args:
        theme_manager: ThemeManager instance to query

    Returns:
        Name of the current theme (e.g., "cyro", "dark", "classic")
    """
    return theme_manager.get_current_theme_name()


def list_themes(theme_manager: Optional[ThemeManager] = None) -> List[str]:
    """
    Get a list of all available theme names.

    Args:
        theme_manager: Optional ThemeManager instance. If None, returns built-in themes only.

    Returns:
        List of theme names sorted alphabetically

    Example:
        tm = create_theme_manager()
        available = list_themes(tm)
        print(f"Available themes: {', '.join(available)}")
    """
    if theme_manager is None:
        return sorted(DEFAULT_THEMES.keys())
    return theme_manager.list_themes()


def get_theme_info(
    theme_name: str, theme_manager: Optional[ThemeManager] = None
) -> Optional[Dict[str, str]]:
    """
    Get information about a specific theme.

    Args:
        theme_name: Name of the theme
        theme_manager: Optional ThemeManager instance. If None, searches built-in themes only.

    Returns:
        Dictionary with theme info (name, description) or None if not found

    Example:
        tm = create_theme_manager()
        info = get_theme_info("cyro", tm)
        if info:
            print(f"{info['name']}: {info['description']}")
    """
    if theme_manager is None:
        theme = DEFAULT_THEMES.get(theme_name)
        if theme:
            return {"name": theme.name, "description": theme.description}
        return None
    return theme_manager.get_theme_info(theme_name)


def load_custom_themes(theme_manager: ThemeManager, themes_dir: str) -> int:
    """
    Load custom themes from TOML files into a theme manager.

    Args:
        theme_manager: ThemeManager instance to load themes into
        themes_dir: Directory path containing .toml theme files

    Returns:
        Number of custom themes successfully loaded

    Example:
        tm = create_theme_manager()
        count = load_custom_themes(tm, "~/.cyro/themes/")
        if count > 0:
            print(f"Loaded {count} custom themes")
    """
    return theme_manager.load_custom_themes(themes_dir)


def reset_themes(theme_manager: ThemeManager) -> None:
    """
    Reset a theme manager to only built-in default themes.

    Args:
        theme_manager: ThemeManager instance to reset

    Example:
        tm = create_theme_manager()
        reset_themes(tm)  # Removes custom themes, resets to "cyro"
    """
    theme_manager.reset_to_defaults()
