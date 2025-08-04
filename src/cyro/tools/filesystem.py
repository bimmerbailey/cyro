"""
Filesystem tools for Cyro agents.

This module provides a comprehensive set of filesystem tools by combining:
1. LangChain FileManagementToolkit for basic operations
2. Custom PydanticAI tools for advanced features like Edit, MultiEdit, Glob

Security: All file operations are scoped to configurable root directories.
"""

import shutil
import warnings
from pathlib import Path
from typing import List, Optional

from langchain_community.agent_toolkits import FileManagementToolkit
from pydantic import BaseModel, Field
from pydantic_ai import Agent
from pydantic_ai.ext.langchain import LangChainToolset
from pydantic_ai.toolsets import FunctionToolset

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


class _FilesystemOperations:
    """
    Core filesystem operations with enhanced security and error handling.
    
    This class centralizes all filesystem logic to eliminate code duplication.
    """

    def __init__(self, config: Optional[CyroConfig] = None):
        """
        Initialize filesystem operations.

        Args:
            config: Cyro configuration for security settings.
        """
        self.config = config or CyroConfig()
        self.root_dir = Path.cwd()  # Always use current working directory
        self.max_file_size_mb = 10  # Default max file size for reads
        self.max_write_size_mb = 100  # Default max file size for writes

    def _validate_path(self, file_path: str) -> Path:
        """
        Validate file path with enhanced security checks.

        Args:
            file_path: File path to validate

        Returns:
            Resolved Path object

        Raises:
            ValueError: If path is outside root directory or has security issues
        """
        path = Path(file_path)
        if not path.is_absolute():
            path = self.root_dir / path

        resolved_path = path.resolve()

        # Check for symlink attacks - limit to relative symlinks within root
        if resolved_path.is_symlink():
            symlink_target = resolved_path.readlink()
            if symlink_target.is_absolute():
                raise ValueError(f"Absolute symlinks not allowed: {file_path}")
            # Resolve the symlink and validate it's still within root
            final_path = (resolved_path.parent / symlink_target).resolve()
            try:
                final_path.relative_to(self.root_dir.resolve())
            except ValueError:
                raise ValueError(f"Symlink target outside allowed root directory: {file_path}")

        # Security check: ensure path is within root directory
        try:
            resolved_path.relative_to(self.root_dir.resolve())
        except ValueError:
            raise ValueError(
                f"Path {file_path} is outside allowed root directory {self.root_dir}"
            )

        return resolved_path

    def _validate_file_size(self, path: Path, max_size_mb: int, operation: str) -> None:
        """
        Validate file size is within limits.

        Args:
            path: Path to check
            max_size_mb: Maximum allowed size in MB
            operation: Operation name for error messages

        Raises:
            ValueError: If file is too large
        """
        if path.exists() and path.is_file():
            file_size = path.stat().st_size
            max_size_bytes = max_size_mb * 1024 * 1024
            if file_size > max_size_bytes:
                raise ValueError(
                    f"File {path} is too large for {operation} "
                    f"({file_size:,} bytes > {max_size_bytes:,} bytes)"
                )

    def read_file(self, file_path: str) -> str:
        """Read the contents of a file with size limits."""
        try:
            path = self._validate_path(file_path)
            if not path.exists():
                return f"Error: File {file_path} does not exist"
            if not path.is_file():
                return f"Error: {file_path} is not a file"
            
            self._validate_file_size(path, self.max_file_size_mb, "reading")
            return path.read_text(encoding="utf-8")
            
        except FileNotFoundError:
            return f"Error: File {file_path} does not exist"
        except PermissionError:
            return f"Error: Permission denied reading {file_path}"
        except UnicodeDecodeError:
            return f"Error: File {file_path} is not valid UTF-8 text"
        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error reading file: {str(e)}"

    def write_file(self, file_path: str, content: str) -> str:
        """Write content to a file with size validation."""
        try:
            path = self._validate_path(file_path)
            
            # Validate content size
            content_size = len(content.encode('utf-8'))
            max_size_bytes = self.max_write_size_mb * 1024 * 1024
            if content_size > max_size_bytes:
                return f"Error: Content too large ({content_size:,} bytes > {max_size_bytes:,} bytes)"
            
            # Create parent directories if they don't exist
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding="utf-8")
            return f"Successfully wrote to {file_path}"
            
        except PermissionError:
            return f"Error: Permission denied writing to {file_path}"
        except OSError as e:
            return f"Error: Could not write to {file_path}: {str(e)}"
        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error writing file: {str(e)}"

    def list_directory(self, directory_path: str = ".") -> str:
        """List the contents of a directory."""
        try:
            path = self._validate_path(directory_path)
            if not path.exists():
                return f"Error: Directory {directory_path} does not exist"
            if not path.is_dir():
                return f"Error: {directory_path} is not a directory"
            
            items = []
            for item in sorted(path.iterdir()):
                try:
                    item_type = "📁" if item.is_dir() else "📄"
                    items.append(f"{item_type} {item.name}")
                except (OSError, PermissionError):
                    # Skip items we can't access
                    items.append(f"❓ {item.name} (access denied)")
            
            if not items:
                return f"Directory {directory_path} is empty"
            
            return f"Contents of {directory_path}:\n" + "\n".join(items)
            
        except PermissionError:
            return f"Error: Permission denied accessing directory {directory_path}"
        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error listing directory: {str(e)}"

    def copy_file(self, source_path: str, destination_path: str) -> str:
        """Copy a file from source to destination."""
        try:
            source = self._validate_path(source_path)
            dest = self._validate_path(destination_path)
            
            if not source.exists():
                return f"Error: Source file {source_path} does not exist"
            if not source.is_file():
                return f"Error: {source_path} is not a file"
            
            self._validate_file_size(source, self.max_write_size_mb, "copying")
            
            # Create parent directories if they don't exist
            dest.parent.mkdir(parents=True, exist_ok=True)
            
            # Copy file
            shutil.copy2(source, dest)
            
            return f"Successfully copied {source_path} to {destination_path}"
            
        except FileNotFoundError:
            return f"Error: Source file {source_path} does not exist"
        except PermissionError:
            return "Error: Permission denied copying file"
        except OSError as e:
            return f"Error: Could not copy file: {str(e)}"
        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error copying file: {str(e)}"

    def delete_file(self, file_path: str) -> str:
        """Delete a file."""
        try:
            path = self._validate_path(file_path)
            if not path.exists():
                return f"Error: File {file_path} does not exist"
            if not path.is_file():
                return f"Error: {file_path} is not a file"
            
            path.unlink()
            return f"Successfully deleted {file_path}"
            
        except FileNotFoundError:
            return f"Error: File {file_path} does not exist"
        except PermissionError:
            return f"Error: Permission denied deleting {file_path}"
        except OSError as e:
            return f"Error: Could not delete file: {str(e)}"
        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error deleting file: {str(e)}"

    def edit_file(self, file_path: str, old_string: str, new_string: str, replace_all: bool = False) -> str:
        """Edit a file by replacing old_string with new_string."""
        try:
            path = self._validate_path(file_path)

            if not path.exists():
                return f"Error: File {file_path} does not exist"
            if not path.is_file():
                return f"Error: {file_path} is not a file"

            self._validate_file_size(path, self.max_file_size_mb, "editing")

            # Read current content
            content = path.read_text(encoding="utf-8")

            # Perform replacement
            if replace_all:
                new_content = content.replace(old_string, new_string)
                replacements = content.count(old_string)
            else:
                occurrences = content.count(old_string)
                if occurrences != 1:
                    return f"Error: old_string appears {occurrences} times, not exactly 1. Use replace_all=True for multiple replacements."
                new_content = content.replace(old_string, new_string, 1)
                replacements = 1

            if new_content == content:
                return f"Warning: No changes made. old_string not found in {file_path}"

            # Validate new content size
            new_content_size = len(new_content.encode('utf-8'))
            max_size_bytes = self.max_write_size_mb * 1024 * 1024
            if new_content_size > max_size_bytes:
                return f"Error: Edited content too large ({new_content_size:,} bytes > {max_size_bytes:,} bytes)"

            # Write updated content
            path.write_text(new_content, encoding="utf-8")

            return f"Successfully edited {file_path}: {replacements} replacement(s) made"

        except FileNotFoundError:
            return f"Error: File {file_path} does not exist"
        except PermissionError:
            return f"Error: Permission denied editing {file_path}"
        except UnicodeDecodeError:
            return f"Error: File {file_path} is not valid UTF-8 text"
        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error editing file: {str(e)}"

    def glob_search(self, pattern: str, root_dir: Optional[str] = None) -> str:
        """Search for files matching a glob pattern."""
        try:
            search_root = self.root_dir
            if root_dir:
                search_root = self._validate_path(root_dir)
                if not search_root.is_dir():
                    return f"Error: {root_dir} is not a directory"

            # Validate glob pattern (basic check)
            if '..' in pattern:
                return "Error: Parent directory references (..) not allowed in pattern"

            # Perform glob search
            matches = list(search_root.glob(pattern))

            if not matches:
                return f"No files found matching pattern '{pattern}' in {search_root}"

            # Return relative paths from search root
            relative_matches = []
            for match in sorted(matches):
                try:
                    rel_path = match.relative_to(search_root)
                    relative_matches.append(str(rel_path))
                except ValueError:
                    # Shouldn't happen with glob, but handle gracefully
                    relative_matches.append(str(match))

            result = f"Found {len(matches)} files matching '{pattern}':\n"
            result += "\n".join(f"  {path}" for path in relative_matches)

            return result

        except ValueError as e:
            return f"Error: {str(e)}"
        except Exception as e:
            return f"Unexpected error in glob search: {str(e)}"


class FilesystemTools:
    """
    Filesystem tools wrapper for backward compatibility.

    Provides secure file operations with configurable root directory restrictions.
    """

    def __init__(self, config: Optional[CyroConfig] = None):
        """
        Initialize filesystem tools using current working directory.

        Args:
            config: Cyro configuration for security settings.
        """
        self.config = config or CyroConfig()
        self._operations = _FilesystemOperations(config)
        self.root_dir = self._operations.root_dir

        # Legacy LangChain support (deprecated - will be removed in future version)
        # Only initialized if actually needed to reduce startup overhead
        self._langchain_toolkit = None
        self._langchain_toolset = None
    
    @property
    def langchain_toolkit(self) -> FileManagementToolkit:
        """Lazy initialization of LangChain toolkit (deprecated)."""
        if self._langchain_toolkit is None:
            self._langchain_toolkit = FileManagementToolkit(
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
        return self._langchain_toolkit
    
    @property
    def langchain_toolset(self) -> LangChainToolset:
        """Lazy initialization of LangChain toolset (deprecated)."""
        if self._langchain_toolset is None:
            self._langchain_toolset = LangChainToolset(self.langchain_toolkit.get_tools())  # type: ignore
        return self._langchain_toolset

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
        instance._operations.root_dir = root_path
        instance.root_dir = root_path
        
        # Reset lazy-loaded LangChain components so they use the new root directory
        instance._langchain_toolkit = None
        instance._langchain_toolset = None
        
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
        """Legacy method - use _operations._validate_path instead."""
        return self._operations._validate_path(file_path)

    def get_langchain_toolset(self) -> LangChainToolset:
        """
        Get the LangChain toolset for PydanticAI integration.
        
        .. deprecated:: 
            Use create_filesystem_toolset() instead for better performance and security.
        """
        warnings.warn(
            "get_langchain_toolset() is deprecated. Use create_filesystem_toolset() instead.",
            DeprecationWarning,
            stacklevel=2
        )
        return self.langchain_toolset

    def create_custom_tools(self, agent: Agent) -> None:
        """
        Add custom filesystem tools to a PydanticAI agent using centralized operations.

        Args:
            agent: PydanticAI agent to add tools to
        """

        @agent.tool_plain
        def edit_file(request: FileEditRequest) -> str:
            """Edit a file by replacing old_string with new_string."""
            return self._operations.edit_file(
                request.file_path, 
                request.old_string, 
                request.new_string, 
                request.replace_all
            )

        @agent.tool_plain
        def multi_edit_file(request: MultiEditRequest) -> str:
            """Perform multiple edits on a single file in sequence."""
            # Convert MultiEditRequest to individual edit operations
            try:
                for i, edit in enumerate(request.edits):
                    result = self._operations.edit_file(
                        request.file_path,
                        edit.old_string,
                        edit.new_string,
                        edit.replace_all
                    )
                    if result.startswith("Error"):
                        return f"Error at edit {i + 1}: {result}"
                
                return f"Successfully applied {len(request.edits)} edits to {request.file_path}"
            except Exception as e:
                return f"Error in multi-edit: {str(e)}"

        @agent.tool_plain
        def glob_search(request: GlobSearchRequest) -> str:
            """Search for files matching a glob pattern."""
            return self._operations.glob_search(request.pattern, request.root_dir)


def create_filesystem_toolset(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """
    Create a complete filesystem toolset for PydanticAI agents using current working directory.

    Args:
        config: Cyro configuration

    Returns:
        FunctionToolset containing filesystem tools
    """
    # Use centralized operations to eliminate code duplication
    operations = _FilesystemOperations(config=config)
    
    # Create FunctionToolset for all file operations
    toolset = FunctionToolset()
    
    # Add filesystem tools using centralized operations
    @toolset.tool
    def read_file(file_path: str) -> str:
        """
        Read the contents of a file with security validation and size limits.

        Args:
            file_path: Path to the file to read

        Returns:
            File contents or error description
        """
        return operations.read_file(file_path)

    @toolset.tool
    def write_file(file_path: str, content: str) -> str:
        """
        Write content to a file with size validation and directory creation.

        Args:
            file_path: Path to the file to write
            content: Content to write to the file

        Returns:
            Success message or error description
        """
        return operations.write_file(file_path, content)

    @toolset.tool
    def list_directory(directory_path: str = ".") -> str:
        """
        List the contents of a directory with permission handling.

        Args:
            directory_path: Path to the directory to list (default: current directory)

        Returns:
            Directory listing or error description
        """
        return operations.list_directory(directory_path)

    @toolset.tool
    def copy_file(source_path: str, destination_path: str) -> str:
        """
        Copy a file from source to destination with size validation.

        Args:
            source_path: Path to the source file
            destination_path: Path to the destination

        Returns:
            Success message or error description
        """
        return operations.copy_file(source_path, destination_path)

    @toolset.tool
    def delete_file(file_path: str) -> str:
        """
        Delete a file with security validation.

        Args:
            file_path: Path to the file to delete

        Returns:
            Success message or error description
        """
        return operations.delete_file(file_path)

    @toolset.tool
    def edit_file(request: FileEditRequest) -> str:
        """
        Edit a file by replacing old_string with new_string.

        Args:
            request: File edit request with path, old/new strings, and options

        Returns:
            Success message or error description
        """
        return operations.edit_file(
            request.file_path,
            request.old_string,
            request.new_string,
            request.replace_all
        )

    @toolset.tool
    def multi_edit_file(request: MultiEditRequest) -> str:
        """
        Perform multiple edits on a single file in sequence.

        Args:
            request: Multi-edit request with file path and list of edits

        Returns:
            Success message with edit summary or error description
        """
        # Apply edits sequentially using centralized operations
        try:
            for i, edit in enumerate(request.edits):
                result = operations.edit_file(
                    request.file_path,
                    edit.old_string,
                    edit.new_string,
                    edit.replace_all
                )
                if result.startswith("Error"):
                    return f"Error at edit {i + 1}: {result}"
            
            return f"Successfully applied {len(request.edits)} edits to {request.file_path}"
        except Exception as e:
            return f"Error in multi-edit: {str(e)}"

    @toolset.tool
    def glob_search(request: GlobSearchRequest) -> str:
        """
        Search for files matching a glob pattern with security validation.

        Args:
            request: Glob search request with pattern and optional root directory

        Returns:
            List of matching file paths or error description
        """
        return operations.glob_search(request.pattern, request.root_dir)
    
    return toolset


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
