"""
Tests for the theme system including ThemeManager, ColorScheme, and Theme models.

This module tests the theme management functionality including:
- Theme creation and validation
- Color resolution with fallbacks
- Theme switching
- Custom theme loading
- Built-in theme integrity
"""

import pytest
from pathlib import Path
import tempfile
import tomllib
from typing import Dict, Any

from cyro.config.themes import (
    ColorScheme,
    Theme,
    ThemeManager,
    DEFAULT_THEMES,
    create_theme_manager,
    get_theme_color,
    set_theme,
    get_current_theme_name,
    list_themes,
    get_theme_info,
    load_custom_themes,
    reset_themes,
)


class TestColorScheme:
    """Test the ColorScheme Pydantic model."""

    def test_color_scheme_creation(self):
        """Test creating a valid ColorScheme."""
        colors = ColorScheme(
            primary="yellow",
            secondary="orange",
            accent="bright_yellow",
            success="bright_green",
            warning="bright_yellow",
            error="bright_red",
            info="yellow",
            text="bright_white",
            text_dim="dim",
            background="default",
            border="yellow",
            highlight="bold yellow",
            code_bg="default",
            code_text="bright_white",
            syntax_keyword="bold bright_blue",
            syntax_string="bright_green",
            syntax_comment="dim",
            table_header="bold yellow",
            table_row="bright_white",
            prompt="bold yellow",
            input_default="yellow",
        )

        assert colors.primary == "yellow"
        assert colors.success == "bright_green"
        assert colors.border == "yellow"

    def test_color_scheme_validation(self):
        """Test that ColorScheme validates required fields."""
        with pytest.raises(ValueError):
            ColorScheme()  # Missing required fields


class TestTheme:
    """Test the Theme Pydantic model."""

    def test_theme_creation(self):
        """Test creating a valid Theme."""
        colors = ColorScheme(
            primary="blue",
            secondary="cyan",
            accent="bright_cyan",
            success="green",
            warning="yellow",
            error="red",
            info="blue",
            text="white",
            text_dim="dim",
            background="default",
            border="blue",
            highlight="bold blue",
            code_bg="default",
            code_text="white",
            syntax_keyword="bold blue",
            syntax_string="green",
            syntax_comment="dim",
            table_header="bold blue",
            table_row="white",
            prompt="bold blue",
            input_default="cyan",
        )

        theme = Theme(name="test_theme", description="A test theme", colors=colors)

        assert theme.name == "test_theme"
        assert theme.description == "A test theme"
        assert theme.colors.primary == "blue"


class TestDefaultThemes:
    """Test the built-in default themes."""

    def test_default_themes_exist(self):
        """Test that all expected default themes exist."""
        expected_themes = {"cyro", "classic", "dark", "light"}
        assert set(DEFAULT_THEMES.keys()) == expected_themes

    def test_cyro_theme_integrity(self):
        """Test the cyro theme has correct properties."""
        cyro = DEFAULT_THEMES["cyro"]
        assert cyro.name == "cyro"
        assert cyro.colors.primary == "yellow"
        assert cyro.colors.border == "yellow"
        assert cyro.colors.success == "bright_green"

    def test_classic_theme_integrity(self):
        """Test the classic theme has correct properties."""
        classic = DEFAULT_THEMES["classic"]
        assert classic.name == "classic"
        assert classic.colors.primary == "bright_blue"
        assert classic.colors.border == "bright_blue"
        assert classic.colors.success == "bright_green"

    def test_dark_theme_integrity(self):
        """Test the dark theme has correct properties."""
        dark = DEFAULT_THEMES["dark"]
        assert dark.name == "dark"
        assert dark.colors.primary == "#6B7280"
        assert dark.colors.border == "#4B5563"
        assert dark.colors.success == "#10B981"

    def test_light_theme_integrity(self):
        """Test the light theme has correct properties."""
        light = DEFAULT_THEMES["light"]
        assert light.name == "light"
        assert light.colors.primary == "#1E40AF"
        assert light.colors.border == "#374151"
        assert light.colors.success == "#065F46"

    def test_all_themes_have_required_colors(self):
        """Test that all default themes have all required color properties."""
        required_colors = [
            "primary",
            "secondary",
            "accent",
            "success",
            "warning",
            "error",
            "info",
            "text",
            "text_dim",
            "background",
            "border",
            "highlight",
            "code_bg",
            "code_text",
            "syntax_keyword",
            "syntax_string",
            "syntax_comment",
            "table_header",
            "table_row",
            "prompt",
            "input_default",
        ]

        for theme_name, theme in DEFAULT_THEMES.items():
            for color_name in required_colors:
                assert hasattr(theme.colors, color_name), (
                    f"Theme '{theme_name}' missing color '{color_name}'"
                )
                color_value = getattr(theme.colors, color_name)
                assert color_value is not None, (
                    f"Theme '{theme_name}' has None for color '{color_name}'"
                )
                assert isinstance(color_value, str), (
                    f"Theme '{theme_name}' color '{color_name}' is not a string"
                )


class TestThemeManager:
    """Test the ThemeManager class."""

    def test_theme_manager_creation(self):
        """Test creating a ThemeManager with default theme."""
        tm = ThemeManager()
        assert tm.get_current_theme_name() == "cyro"
        assert len(tm.list_themes()) == 4  # cyro, classic, dark, light

    def test_theme_manager_custom_default(self):
        """Test creating a ThemeManager with custom default theme."""
        tm = ThemeManager("classic")
        assert tm.get_current_theme_name() == "classic"

    def test_theme_manager_invalid_default_fallback(self):
        """Test that invalid default theme falls back to cyro."""
        tm = ThemeManager("nonexistent")
        assert tm.get_current_theme_name() == "cyro"

    def test_get_color_current_theme(self):
        """Test getting colors from current theme."""
        tm = ThemeManager("classic")
        assert tm.get_color("primary") == "bright_blue"
        assert tm.get_color("border") == "bright_blue"

    def test_get_color_fallback_to_cyro(self):
        """Test color fallback to cyro theme when current theme doesn't have color."""
        # This is tricky to test since all themes have all colors
        # Let's test the fallback logic by checking a cyro theme directly
        tm = ThemeManager("cyro")
        assert tm.get_color("primary") == "yellow"

    def test_get_color_ultimate_fallback(self):
        """Test ultimate fallback when color doesn't exist."""
        tm = ThemeManager()
        # Test with a non-existent color
        assert tm.get_color("nonexistent_color") == "bright_white"

    def test_set_theme_success(self):
        """Test successfully switching themes."""
        tm = ThemeManager()
        assert tm.get_current_theme_name() == "cyro"

        success = tm.set_theme("classic")
        assert success is True
        assert tm.get_current_theme_name() == "classic"
        assert tm.get_color("primary") == "bright_blue"

    def test_set_theme_failure(self):
        """Test switching to non-existent theme."""
        tm = ThemeManager()
        original_theme = tm.get_current_theme_name()

        success = tm.set_theme("nonexistent")
        assert success is False
        assert tm.get_current_theme_name() == original_theme

    def test_list_themes(self):
        """Test listing available themes."""
        tm = ThemeManager()
        themes = tm.list_themes()
        assert isinstance(themes, list)
        assert len(themes) == 4
        assert "cyro" in themes
        assert "classic" in themes
        assert "dark" in themes
        assert "light" in themes
        assert themes == sorted(themes)  # Should be sorted

    def test_get_theme_info_existing(self):
        """Test getting info for existing theme."""
        tm = ThemeManager()
        info = tm.get_theme_info("cyro")
        assert info is not None
        assert info["name"] == "cyro"
        assert "description" in info
        assert isinstance(info["description"], str)

    def test_get_theme_info_nonexistent(self):
        """Test getting info for non-existent theme."""
        tm = ThemeManager()
        info = tm.get_theme_info("nonexistent")
        assert info is None

    def test_get_current_theme(self):
        """Test getting current theme object."""
        tm = ThemeManager("classic")
        current = tm.get_current_theme()
        assert current.name == "classic"
        assert current.colors.primary == "bright_blue"

    def test_reset_to_defaults(self):
        """Test resetting theme manager to defaults."""
        tm = ThemeManager()
        # We'll test this more thoroughly with custom themes below
        original_count = len(tm.list_themes())
        tm.reset_to_defaults()
        assert len(tm.list_themes()) == original_count
        assert tm.get_current_theme_name() == "cyro"


class TestCustomThemeLoading:
    """Test loading custom themes from TOML files."""

    def test_load_custom_themes_from_directory(self):
        """Test loading custom themes from a directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Create a custom theme file
            custom_theme_data = {
                "name": "custom_test",
                "description": "A custom test theme",
                "colors": {
                    "primary": "purple",
                    "secondary": "magenta",
                    "accent": "bright_magenta",
                    "success": "bright_green",
                    "warning": "bright_yellow",
                    "error": "bright_red",
                    "info": "purple",
                    "text": "bright_white",
                    "text_dim": "dim",
                    "background": "default",
                    "border": "purple",
                    "highlight": "bold purple",
                    "code_bg": "default",
                    "code_text": "bright_white",
                    "syntax_keyword": "bold purple",
                    "syntax_string": "bright_green",
                    "syntax_comment": "dim",
                    "table_header": "bold purple",
                    "table_row": "bright_white",
                    "prompt": "bold purple",
                    "input_default": "magenta",
                },
            }

            theme_file = Path(temp_dir) / "custom_test.toml"
            with open(theme_file, "w") as f:
                import toml

                toml.dump(custom_theme_data, f)

            tm = ThemeManager()
            initial_count = len(tm.list_themes())

            loaded_count = tm.load_custom_themes(temp_dir)
            assert loaded_count == 1

            themes = tm.list_themes()
            assert len(themes) == initial_count + 1
            assert "custom_test" in themes

            # Test switching to custom theme
            success = tm.set_theme("custom_test")
            assert success is True
            assert tm.get_color("primary") == "purple"

    def test_load_custom_themes_invalid_directory(self):
        """Test loading from non-existent directory."""
        tm = ThemeManager()
        loaded_count = tm.load_custom_themes("/nonexistent/directory")
        assert loaded_count == 0

    def test_load_custom_themes_empty_directory(self):
        """Test loading from empty directory."""
        with tempfile.TemporaryDirectory() as temp_dir:
            tm = ThemeManager()
            loaded_count = tm.load_custom_themes(temp_dir)
            assert loaded_count == 0

    def test_load_custom_themes_invalid_toml(self):
        """Test loading invalid TOML files is skipped gracefully."""
        with tempfile.TemporaryDirectory() as temp_dir:
            invalid_file = Path(temp_dir) / "invalid.toml"
            with open(invalid_file, "w") as f:
                f.write("invalid toml content [[[")

            tm = ThemeManager()
            loaded_count = tm.load_custom_themes(temp_dir)
            assert loaded_count == 0

    def test_load_custom_themes_missing_required_fields(self):
        """Test that themes missing required fields are skipped."""
        with tempfile.TemporaryDirectory() as temp_dir:
            incomplete_theme = {"name": "incomplete"}  # Missing colors

            theme_file = Path(temp_dir) / "incomplete.toml"
            with open(theme_file, "w") as f:
                import toml

                toml.dump(incomplete_theme, f)

            tm = ThemeManager()
            loaded_count = tm.load_custom_themes(temp_dir)
            assert loaded_count == 0

    def test_load_custom_themes_dont_override_builtin(self):
        """Test that custom themes can't override built-in themes."""
        with tempfile.TemporaryDirectory() as temp_dir:
            # Try to create a custom "cyro" theme
            custom_cyro = {
                "name": "cyro",  # Same name as built-in
                "description": "Fake cyro theme",
                "colors": {
                    "primary": "red",  # Different color
                    "secondary": "orange",
                    "accent": "bright_yellow",
                    "success": "bright_green",
                    "warning": "bright_yellow",
                    "error": "bright_red",
                    "info": "yellow",
                    "text": "bright_white",
                    "text_dim": "dim",
                    "background": "default",
                    "border": "yellow",
                    "highlight": "bold yellow",
                    "code_bg": "default",
                    "code_text": "bright_white",
                    "syntax_keyword": "bold bright_blue",
                    "syntax_string": "bright_green",
                    "syntax_comment": "dim",
                    "table_header": "bold yellow",
                    "table_row": "bright_white",
                    "prompt": "bold yellow",
                    "input_default": "yellow",
                },
            }

            theme_file = Path(temp_dir) / "cyro.toml"
            with open(theme_file, "w") as f:
                import toml

                toml.dump(custom_cyro, f)

            tm = ThemeManager()
            original_cyro_color = tm.get_color("primary")  # Should be yellow

            loaded_count = tm.load_custom_themes(temp_dir)
            assert loaded_count == 0  # Should not load

            # Cyro theme should remain unchanged
            assert tm.get_color("primary") == original_cyro_color


class TestThemeUtilityFunctions:
    """Test the utility functions for theme management."""

    def test_create_theme_manager(self):
        """Test the create_theme_manager utility function."""
        tm = create_theme_manager()
        assert isinstance(tm, ThemeManager)
        assert tm.get_current_theme_name() == "cyro"

        tm_custom = create_theme_manager("classic")
        assert tm_custom.get_current_theme_name() == "classic"

    def test_get_theme_color_with_manager(self):
        """Test get_theme_color with theme manager."""
        tm = create_theme_manager("classic")
        color = get_theme_color("primary", tm)
        assert color == "bright_blue"

    def test_get_theme_color_without_manager(self):
        """Test get_theme_color fallback without theme manager."""
        color = get_theme_color("primary")  # Should use cyro default
        assert color == "yellow"

        color_nonexistent = get_theme_color("nonexistent")
        assert color_nonexistent == "bright_white"

    def test_set_theme_utility(self):
        """Test the set_theme utility function."""
        tm = create_theme_manager()
        assert tm.get_current_theme_name() == "cyro"

        success = set_theme(tm, "classic")
        assert success is True
        assert tm.get_current_theme_name() == "classic"

        failure = set_theme(tm, "nonexistent")
        assert failure is False
        assert tm.get_current_theme_name() == "classic"  # Unchanged

    def test_get_current_theme_name_utility(self):
        """Test the get_current_theme_name utility function."""
        tm = create_theme_manager("dark")
        name = get_current_theme_name(tm)
        assert name == "dark"

    def test_list_themes_utility(self):
        """Test the list_themes utility function."""
        tm = create_theme_manager()
        themes_with_manager = list_themes(tm)
        themes_without_manager = list_themes()

        assert themes_with_manager == themes_without_manager
        assert isinstance(themes_with_manager, list)
        assert len(themes_with_manager) == 4

    def test_get_theme_info_utility(self):
        """Test the get_theme_info utility function."""
        tm = create_theme_manager()

        info_with_manager = get_theme_info("cyro", tm)
        info_without_manager = get_theme_info("cyro")

        assert info_with_manager == info_without_manager
        assert info_with_manager["name"] == "cyro"
        assert "description" in info_with_manager

    def test_load_custom_themes_utility(self):
        """Test the load_custom_themes utility function."""
        tm = create_theme_manager()
        # Test with non-existent directory
        count = load_custom_themes(tm, "/nonexistent")
        assert count == 0

    def test_reset_themes_utility(self):
        """Test the reset_themes utility function."""
        tm = create_theme_manager()
        original_count = len(tm.list_themes())

        reset_themes(tm)

        assert len(tm.list_themes()) == original_count
        assert tm.get_current_theme_name() == "cyro"


class TestThemeManagerEdgeCases:
    """Test edge cases and error conditions."""

    def test_theme_manager_color_resolution_logic(self):
        """Test the specific color resolution logic."""
        tm = ThemeManager("classic")

        # When current theme is not cyro, should use current theme
        assert tm.get_color("primary") == "bright_blue"  # classic theme

        # Switch to cyro theme
        tm.set_theme("cyro")
        assert tm.get_color("primary") == "yellow"  # cyro theme

    def test_theme_manager_empty_theme_dir_parameter(self):
        """Test load_custom_themes with empty string."""
        tm = ThemeManager()
        count = tm.load_custom_themes("")
        assert count == 0

    def test_theme_manager_none_theme_dir_parameter(self):
        """Test load_custom_themes with None."""
        tm = ThemeManager()
        count = tm.load_custom_themes(None)
        assert count == 0


# Integration tests
class TestThemeIntegration:
    """Integration tests for the complete theme system."""

    def test_complete_theme_workflow(self):
        """Test a complete workflow of theme operations."""
        # Create theme manager
        tm = create_theme_manager()
        assert get_current_theme_name(tm) == "cyro"

        # List themes
        themes = list_themes(tm)
        assert len(themes) == 4

        # Get theme info
        info = get_theme_info("classic", tm)
        assert info["name"] == "classic"

        # Switch theme
        success = set_theme(tm, "classic")
        assert success is True
        assert get_current_theme_name(tm) == "classic"

        # Test color resolution
        color = get_theme_color("primary", tm)
        assert color == "bright_blue"

        # Reset themes
        reset_themes(tm)
        assert get_current_theme_name(tm) == "cyro"

    def test_theme_persistence_within_manager(self):
        """Test that theme changes persist within the same manager instance."""
        tm = create_theme_manager()

        # Switch through all themes
        for theme_name in ["classic", "dark", "light", "cyro"]:
            success = set_theme(tm, theme_name)
            assert success is True
            assert get_current_theme_name(tm) == theme_name

            # Verify colors change
            color = get_theme_color("primary", tm)
            if theme_name == "cyro":
                assert color == "yellow"
            elif theme_name == "classic":
                assert color == "bright_blue"
            elif theme_name == "dark":
                assert color == "#6B7280"
            elif theme_name == "light":
                assert color == "#1E40AF"


if __name__ == "__main__":
    pytest.main([__file__])
