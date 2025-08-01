"""
Filesystem tools for Cyro agents.

This module provides a comprehensive set of filesystem tools by combining:
1. LangChain FileManagementToolkit for basic operations
2. Custom PydanticAI tools for advanced features like Edit, MultiEdit, Glob

Security: All file operations are scoped to configurable root directories.
"""

from pathlib import Path
from typing import Any, List, Optional

from langchain_community.agent_toolkits import FileManagementToolkit
from pydantic import BaseModel, Field
from pydantic_ai import Agent
from pydantic_ai.ext.langchain import LangChainToolset

from cyro.config.settings import CyroConfig


class FileEditRequest(BaseModel):
    """Request model for file editing operations."""

    file_path: str = Field(description="Path to the file to edit")
    old_string: str = Field(description="Text to replace")
    new_string: str = Field(description="Replacement text")
    replace_all: bool = Field(default=False, description="Replace all occurrences")


class MultiEditRequest(BaseModel):
    """Request model for multiple file edits."""

    file_path: str = Field(description="Path to the file to edit")
    edits: List[FileEditRequest] = Field(description="List of edit operations")


class GlobSearchRequest(BaseModel):
    """Request model for glob pattern searches."""

    pattern: str = Field(description="Glob pattern to match")
    root_dir: Optional[str] = Field(
        default=None, description="Root directory to search in"
    )


class FilesystemTools:
    """
    Filesystem tools combining LangChain and custom PydanticAI tools.

    Provides secure file operations with configurable root directory restrictions.
    """

    def __init__(self, config: Optional[CyroConfig] = None):
        """
        Initialize filesystem tools using current working directory.

        Args:
            config: Cyro configuration for security settings.
        """
        self.config = config or CyroConfig()
        self.root_dir = Path.cwd()  # Always use current working directory

        # Initialize LangChain FileManagementToolkit
        self.langchain_toolkit = FileManagementToolkit(
            root_dir=str(self.root_dir),
            selected_tools=[
                "read_file",
                "write_file",
                "list_directory",
                "file_search",
                "copy_file",
                "move_file",
                "file_delete",
            ],
        )

        # Get LangChain tools as PydanticAI toolset
        self.langchain_toolset = LangChainToolset(self.langchain_toolkit.get_tools())  # type: ignore

    @classmethod
    def for_directory(cls, root_dir: str, config: Optional[CyroConfig] = None) -> "FilesystemTools":
        """
        Create filesystem tools for a specific directory.

        Args:
            root_dir: Root directory that must exist
            config: Cyro configuration for security settings.

        Returns:
            FilesystemTools instance

        Raises:
            ValueError: If directory doesn't exist
        """
        root_path = Path(root_dir).resolve()
        if not root_path.exists():
            raise ValueError(f"Directory does not exist: {root_dir}")
        if not root_path.is_dir():
            raise ValueError(f"Path is not a directory: {root_dir}")

        instance = cls(config=config)
        instance.root_dir = root_path
        
        # Recreate toolkit with new root directory
        instance.langchain_toolkit = FileManagementToolkit(
            root_dir=str(root_path),
            selected_tools=[
                "read_file",
                "write_file", 
                "list_directory",
                "file_search",
                "copy_file",
                "move_file",
                "file_delete",
            ],
        )
        instance.langchain_toolset = LangChainToolset(instance.langchain_toolkit.get_tools())  # type: ignore
        
        return instance

    @classmethod
    def for_testing(cls, temp_dir: str, config: Optional[CyroConfig] = None) -> "FilesystemTools":
        """
        Create filesystem tools for testing with a temporary directory.

        Args:
            temp_dir: Temporary directory path (will be created if needed)
            config: Cyro configuration for security settings.

        Returns:
            FilesystemTools instance
        """
        temp_path = Path(temp_dir).resolve()
        temp_path.mkdir(parents=True, exist_ok=True)  # Only create for testing
        
        return cls.for_directory(str(temp_path), config=config)

    def _validate_path(self, file_path: str) -> Path:
        """
        Validate file path is within root directory.

        Args:
            file_path: File path to validate

        Returns:
            Resolved Path object

        Raises:
            ValueError: If path is outside root directory
        """
        path = Path(file_path)
        if not path.is_absolute():
            path = self.root_dir / path

        resolved_path = path.resolve()

        # Security check: ensure path is within root directory
        try:
            resolved_path.relative_to(self.root_dir.resolve())
        except ValueError:
            raise ValueError(
                f"Path {file_path} is outside allowed root directory {self.root_dir}"
            )

        return resolved_path

    def get_langchain_toolset(self) -> LangChainToolset:
        """Get the LangChain toolset for PydanticAI integration."""
        return self.langchain_toolset

    def create_custom_tools(self, agent: Agent) -> None:
        """
        Add custom filesystem tools to a PydanticAI agent.

        Args:
            agent: PydanticAI agent to add tools to
        """

        @agent.tool_plain
        def edit_file(request: FileEditRequest) -> str:
            """
            Edit a file by replacing old_string with new_string.

            Args:
                request: File edit request with path, old/new strings, and options

            Returns:
                Success message or error description
            """
            try:
                file_path = self._validate_path(request.file_path)

                if not file_path.exists():
                    return f"Error: File {request.file_path} does not exist"

                # Read current content
                content = file_path.read_text(encoding="utf-8")

                # Perform replacement
                if request.replace_all:
                    new_content = content.replace(
                        request.old_string, request.new_string
                    )
                    replacements = content.count(request.old_string)
                else:
                    if content.count(request.old_string) != 1:
                        return f"Error: old_string appears {content.count(request.old_string)} times, not exactly 1. Use replace_all=True for multiple replacements."
                    new_content = content.replace(
                        request.old_string, request.new_string, 1
                    )
                    replacements = 1

                if new_content == content:
                    return f"Warning: No changes made. old_string not found in {request.file_path}"

                # Write updated content
                file_path.write_text(new_content, encoding="utf-8")

                return f"Successfully edited {request.file_path}: {replacements} replacement(s) made"

            except Exception as e:
                return f"Error editing file: {str(e)}"

        @agent.tool_plain
        def multi_edit_file(request: MultiEditRequest) -> str:
            """
            Perform multiple edits on a single file in sequence.

            Args:
                request: Multi-edit request with file path and list of edits

            Returns:
                Success message with edit summary or error description
            """
            try:
                file_path = self._validate_path(request.file_path)

                if not file_path.exists():
                    return f"Error: File {request.file_path} does not exist"

                # Read initial content
                content = file_path.read_text(encoding="utf-8")
                original_content = content
                total_replacements = 0

                # Apply edits sequentially
                for i, edit in enumerate(request.edits):
                    if edit.replace_all:
                        count = content.count(edit.old_string)
                        content = content.replace(edit.old_string, edit.new_string)
                        total_replacements += count
                    else:
                        if content.count(edit.old_string) != 1:
                            return f"Error at edit {i + 1}: old_string appears {content.count(edit.old_string)} times, not exactly 1"
                        content = content.replace(edit.old_string, edit.new_string, 1)
                        total_replacements += 1

                if content == original_content:
                    return f"Warning: No changes made to {request.file_path}"

                # Write final content
                file_path.write_text(content, encoding="utf-8")

                return f"Successfully applied {len(request.edits)} edits to {request.file_path}: {total_replacements} total replacement(s)"

            except Exception as e:
                return f"Error in multi-edit: {str(e)}"

        @agent.tool_plain
        def glob_search(request: GlobSearchRequest) -> str:
            """
            Search for files matching a glob pattern.

            Args:
                request: Glob search request with pattern and optional root directory

            Returns:
                List of matching file paths or error description
            """
            try:
                search_root = self.root_dir
                if request.root_dir:
                    search_root = self._validate_path(request.root_dir)
                    if not search_root.is_dir():
                        return f"Error: {request.root_dir} is not a directory"

                # Perform glob search
                matches = list(search_root.glob(request.pattern))

                if not matches:
                    return f"No files found matching pattern '{request.pattern}' in {search_root}"

                # Return relative paths from search root
                relative_matches = []
                for match in sorted(matches):
                    try:
                        rel_path = match.relative_to(search_root)
                        relative_matches.append(str(rel_path))
                    except ValueError:
                        # Shouldn't happen with glob, but handle gracefully
                        relative_matches.append(str(match))

                result = f"Found {len(matches)} files matching '{request.pattern}':\n"
                result += "\n".join(f"  {path}" for path in relative_matches)

                return result

            except Exception as e:
                return f"Error in glob search: {str(e)}"


def create_filesystem_toolset(config: Optional[CyroConfig] = None) -> tuple[LangChainToolset, Any]:
    """
    Create a complete filesystem toolset for PydanticAI agents using current working directory.

    Args:
        config: Cyro configuration

    Returns:
        Tuple of (LangChain toolset, custom tools creator function)
    """
    filesystem_tools = FilesystemTools(config=config)

    return (
        filesystem_tools.get_langchain_toolset(),
        filesystem_tools.create_custom_tools,
    )


# Convenience function for common use cases
def get_basic_file_tools() -> LangChainToolset:
    """
    Get basic file tools (read, write, list, search) as a LangChain toolset using current working directory.

    Returns:
        LangChain toolset with basic file operations
    """
    toolkit = FileManagementToolkit(
        root_dir=str(Path.cwd()),
        selected_tools=["read_file", "write_file", "list_directory", "file_search"],
    )
    return LangChainToolset(toolkit.get_tools())  # type: ignore
