"""
Git operations tools for Cyro agents - Refactored for Factory Integration

This module provides comprehensive git operations by combining:
1. PydanticAI FunctionToolset for local git commands (status, commit, etc.)
2. Optional LangChain GitHub Toolkit for remote repository operations

Security: Git operations are scoped to the current working directory and include validation.
Pattern: Follows filesystem.py implementation pattern for consistency.
"""

import subprocess
from pathlib import Path
from typing import List, Optional

from pydantic import BaseModel, Field
from pydantic_ai.toolsets import FunctionToolset

from cyro.config.settings import CyroConfig

# Optional GitHub integration
try:
    from langchain_community.agent_toolkits import GitHubToolkit
    from pydantic_ai.ext.langchain import LangChainToolset
    GITHUB_AVAILABLE = True
except ImportError:
    GITHUB_AVAILABLE = False


class GitStatusRequest(BaseModel):
    """Request model for git status operations."""

    include_untracked: bool = Field(default=True, description="Include untracked files")
    include_ignored: bool = Field(default=False, description="Include ignored files")


class GitStatusResult(BaseModel):
    """Result model for git status operations."""

    modified_files: List[str] = Field(description="List of modified files")
    added_files: List[str] = Field(description="List of added/staged files")
    deleted_files: List[str] = Field(description="List of deleted files")
    untracked_files: List[str] = Field(description="List of untracked files")
    current_branch: str = Field(description="Current branch name")
    is_clean: bool = Field(description="True if working directory is clean")


class GitCommitRequest(BaseModel):
    """Request model for git commit operations."""

    message: str = Field(description="Commit message")
    add_all: bool = Field(
        default=False, description="Add all changes before committing"
    )
    files: Optional[List[str]] = Field(
        default=None, description="Specific files to add and commit"
    )


class GitCommitResult(BaseModel):
    """Result model for git commit operations."""

    commit_hash: str = Field(description="SHA hash of the new commit")
    message: str = Field(description="Commit message")
    files_changed: int = Field(description="Number of files changed")
    insertions: int = Field(description="Number of insertions")
    deletions: int = Field(description="Number of deletions")


class GitBranchRequest(BaseModel):
    """Request model for git branch operations."""

    branch_name: str = Field(description="Name of the branch")
    checkout: bool = Field(
        default=True, description="Switch to the new branch after creation"
    )


class GitBranchResult(BaseModel):
    """Result model for git branch operations."""

    branch_name: str = Field(description="Branch name")
    created: bool = Field(description="True if branch was created")
    switched: bool = Field(description="True if switched to the branch")
    current_branch: str = Field(description="Current active branch")


class GitLogRequest(BaseModel):
    """Request model for git log operations."""

    max_commits: int = Field(
        default=10, description="Maximum number of commits to return"
    )
    since: Optional[str] = Field(
        default=None, description="Show commits since date (e.g., '2024-01-01')"
    )
    author: Optional[str] = Field(default=None, description="Filter by author")
    grep: Optional[str] = Field(
        default=None, description="Filter by commit message pattern"
    )


class GitLogResult(BaseModel):
    """Result model for git log operations."""

    commits: List[dict] = Field(description="List of commit information")
    total_commits: int = Field(description="Total number of commits returned")


class _GitOperations:
    """Internal git operations class following filesystem.py pattern."""
    
    # TODO: Implement dangerous command detection and warning system
    # TODO: Add dry-run mode for destructive operations

    def __init__(self, config: Optional[CyroConfig] = None):
        """Initialize git operations with configuration."""
        self.config = config or CyroConfig()
        self.working_dir = Path.cwd()

    def _run_git_command(
        self, command: str, cwd: Optional[Path] = None
    ) -> tuple[str, str, int]:
        """Run a git command and return output, error, and return code."""
        working_dir = cwd or self.working_dir

        try:
            result = subprocess.run(
                command.split(),
                cwd=working_dir,
                capture_output=True,
                text=True,
                timeout=30,
            )
            return result.stdout.strip(), result.stderr.strip(), result.returncode
        except subprocess.TimeoutExpired:
            return "", "Command timed out", 124
        except Exception as e:
            return "", str(e), 1

    def git_status(self, request: GitStatusRequest) -> GitStatusResult:
        """Get git repository status."""
        stdout, stderr, returncode = self._run_git_command("git status --porcelain")

        if returncode != 0:
            raise RuntimeError(f"Git status failed: {stderr}")

        # Parse git status output
        modified_files = []
        added_files = []
        deleted_files = []
        untracked_files = []

        for line in stdout.split("\n"):
            if not line:
                continue
            status = line[:2]
            filename = line[3:]

            if status.startswith("M") or status.endswith("M"):
                modified_files.append(filename)
            elif status.startswith("A"):
                added_files.append(filename)
            elif status.startswith("D"):
                deleted_files.append(filename)
            elif status.startswith("??"):
                untracked_files.append(filename)

        # Get current branch
        branch_stdout, _, branch_returncode = self._run_git_command(
            "git branch --show-current"
        )
        current_branch = branch_stdout if branch_returncode == 0 else "unknown"

        return GitStatusResult(
            modified_files=modified_files,
            added_files=added_files,
            deleted_files=deleted_files,
            untracked_files=untracked_files if request.include_untracked else [],
            current_branch=current_branch,
            is_clean=not any(
                [modified_files, added_files, deleted_files, untracked_files]
            ),
        )

    def git_commit(self, request: GitCommitRequest) -> GitCommitResult:
        """Create a git commit."""
        # TODO: Add confirmation dialog for git commit operations (prevents accidental commits)
        # TODO: Add extra confirmation when add_all=True (stages all changes)
        # Add files if specified
        if request.add_all:
            stdout, stderr, returncode = self._run_git_command("git add .")
            if returncode != 0:
                raise RuntimeError(f"Git add failed: {stderr}")
        elif request.files:
            for file_path in request.files:
                stdout, stderr, returncode = self._run_git_command(
                    f"git add {file_path}"
                )
                if returncode != 0:
                    raise RuntimeError(f"Git add failed for {file_path}: {stderr}")

        # Create commit
        commit_cmd = f'git commit -m "{request.message}"'
        stdout, stderr, returncode = self._run_git_command(commit_cmd)

        if returncode != 0:
            raise RuntimeError(f"Git commit failed: {stderr}")

        # Get commit details
        hash_stdout, _, _ = self._run_git_command("git rev-parse HEAD")
        commit_hash = hash_stdout[:8] if hash_stdout else "unknown"

        # Parse commit statistics
        stats_stdout, _, _ = self._run_git_command(f"git show --stat {commit_hash}")
        files_changed = 0
        insertions = 0
        deletions = 0

        # Simple parsing of git show output
        if stats_stdout:
            lines = stats_stdout.split("\n")
            for line in lines:
                if "changed" in line:
                    parts = line.split()
                    for i, part in enumerate(parts):
                        if part.isdigit():
                            if "file" in parts[i + 1 : i + 2]:
                                files_changed = int(part)
                            elif "insertion" in " ".join(parts[i + 1 : i + 3]):
                                insertions = int(part)
                            elif "deletion" in " ".join(parts[i + 1 : i + 3]):
                                deletions = int(part)

        return GitCommitResult(
            commit_hash=commit_hash,
            message=request.message,
            files_changed=files_changed,
            insertions=insertions,
            deletions=deletions,
        )

    def git_branch(self, request: GitBranchRequest) -> GitBranchResult:
        """Create and optionally switch to a git branch."""
        # TODO: Add confirmation dialog when checkout=True (prevents accidental branch switches that could lose uncommitted work)
        # Check if branch already exists
        list_stdout, _, _ = self._run_git_command("git branch --list")
        branch_exists = request.branch_name in list_stdout

        created = False
        switched = False

        if not branch_exists:
            # Create branch
            stdout, stderr, returncode = self._run_git_command(
                f"git branch {request.branch_name}"
            )
            if returncode != 0:
                raise RuntimeError(
                    f"Failed to create branch {request.branch_name}: {stderr}"
                )
            created = True

        if request.checkout:
            # Switch to branch
            stdout, stderr, returncode = self._run_git_command(
                f"git checkout {request.branch_name}"
            )
            if returncode != 0:
                raise RuntimeError(
                    f"Failed to checkout branch {request.branch_name}: {stderr}"
                )
            switched = True

        # Get current branch
        branch_stdout, _, _ = self._run_git_command("git branch --show-current")
        current_branch = branch_stdout if branch_stdout else "unknown"

        return GitBranchResult(
            branch_name=request.branch_name,
            created=created,
            switched=switched,
            current_branch=current_branch,
        )

    def git_log(self, request: GitLogRequest) -> GitLogResult:
        """Get git commit history."""
        # Build git log command
        cmd_parts = [
            "git",
            "log",
            f"--max-count={request.max_commits}",
            "--oneline",
            "--format=%H|%an|%ad|%s",
        ]

        if request.since:
            cmd_parts.append(f"--since={request.since}")
        if request.author:
            cmd_parts.append(f"--author={request.author}")
        if request.grep:
            cmd_parts.append(f"--grep={request.grep}")

        stdout, stderr, returncode = self._run_git_command(" ".join(cmd_parts))

        if returncode != 0:
            raise RuntimeError(f"Git log failed: {stderr}")

        commits = []
        for line in stdout.split("\n"):
            if not line:
                continue
            parts = line.split("|", 3)
            if len(parts) >= 4:
                commits.append(
                    {
                        "hash": parts[0][:8],
                        "author": parts[1],
                        "date": parts[2],
                        "message": parts[3],
                    }
                )

        return GitLogResult(commits=commits, total_commits=len(commits))

    # TODO: Implement git_reset with mandatory confirmation (DANGEROUS - can destroy uncommitted work)
    # def git_reset(self, request: GitResetRequest) -> GitResetResult:
    #     """Reset current branch to specified commit."""
    #     # TODO: Add confirmation dialog - this is a destructive operation
    #     pass

    # TODO: Implement git_clean with confirmation (DANGEROUS - permanently deletes untracked files)
    # def git_clean(self, request: GitCleanRequest) -> GitCleanResult:
    #     """Remove untracked files from working directory."""
    #     # TODO: Add confirmation dialog - this permanently deletes files
    #     pass

    # TODO: Implement git_stash_drop with confirmation (DANGEROUS - permanent data loss)
    # def git_stash_drop(self, request: GitStashDropRequest) -> GitStashDropResult:
    #     """Drop a specific stash entry."""
    #     # TODO: Add confirmation dialog - this permanently deletes stashed changes
    #     pass

    # TODO: Implement git_rebase with confirmation (DANGEROUS - rewrites commit history)
    # def git_rebase(self, request: GitRebaseRequest) -> GitRebaseResult:
    #     """Rebase current branch onto another branch."""
    #     # TODO: Add confirmation dialog - this rewrites commit history
    #     pass

    # TODO: Implement git_push with confirmation for force pushes (DANGEROUS - can overwrite remote history)
    # def git_push(self, request: GitPushRequest) -> GitPushResult:
    #     """Push commits to remote repository."""
    #     # TODO: Add confirmation dialog when force=True - this can overwrite remote history
    #     pass


def create_git_toolset(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """
    Create a complete git toolset for PydanticAI agents following filesystem.py pattern.

    Args:
        config: Cyro configuration

    Returns:
        FunctionToolset containing git tools
    """
    # Use centralized operations to eliminate code duplication
    operations = _GitOperations(config=config)

    # Create FunctionToolset for all git operations
    toolset = FunctionToolset()

    @toolset.tool
    def git_status(request: GitStatusRequest) -> GitStatusResult:
        """
        Get current git repository status including modified, added, deleted, and untracked files.

        Args:
            request: Git status request with options for untracked/ignored files

        Returns:
            GitStatusResult with file lists and repository state
        """
        return operations.git_status(request)

    @toolset.tool
    def git_commit(request: GitCommitRequest) -> GitCommitResult:
        """
        Create a git commit with the specified message and files.

        Args:
            request: Git commit request with message, files, and options

        Returns:
            GitCommitResult with commit details and statistics
        """
        return operations.git_commit(request)

    @toolset.tool
    def git_branch(request: GitBranchRequest) -> GitBranchResult:
        """
        Create a new git branch and optionally switch to it.

        Args:
            request: Git branch request with branch name and checkout option

        Returns:
            GitBranchResult with branch creation and checkout status
        """
        return operations.git_branch(request)

    @toolset.tool
    def git_log(request: GitLogRequest) -> GitLogResult:
        """
        Get git commit history with optional filtering.

        Args:
            request: Git log request with filtering options (author, date, message)

        Returns:
            GitLogResult with list of commits and metadata
        """
        return operations.git_log(request)

    return toolset


def create_github_toolset(
    github_app_id: str,
    github_private_key: str,
    github_installation_id: str,
):
    """
    Create GitHub remote operations toolset for advanced git workflows.

    Args:
        github_app_id: GitHub App ID for remote operations
        github_private_key: GitHub App private key
        github_installation_id: GitHub App installation ID

    Returns:
        LangChainToolset with GitHub remote operations

    Raises:
        ImportError: If LangChain GitHub toolkit is not available
        ValueError: If GitHub credentials are invalid
    """
    if not GITHUB_AVAILABLE:
        raise ImportError("GitHub toolkit not available. Install langchain-community with GitHub support.")
        
    try:
        github_toolkit = GitHubToolkit(
            github_app_id=github_app_id,
            github_private_key=github_private_key,
            github_installation_id=github_installation_id,
        )
        return LangChainToolset(github_toolkit.get_tools())
    except Exception as e:
        raise ValueError(f"Failed to initialize GitHub toolkit: {e}")


def create_full_git_toolset(
    config: Optional[CyroConfig] = None,
    github_app_id: Optional[str] = None,
    github_private_key: Optional[str] = None,
    github_installation_id: Optional[str] = None,
) -> FunctionToolset:
    """
    Create a complete git toolset with both local and GitHub operations.

    Args:
        config: Cyro configuration
        github_app_id: GitHub App ID for remote operations (optional)
        github_private_key: GitHub App private key (optional)
        github_installation_id: GitHub App installation ID (optional)

    Returns:
        FunctionToolset containing local git tools and GitHub tools (if configured)
    """
    # Start with local git toolset
    toolset = create_git_toolset(config)

    # Add GitHub tools if credentials provided and available
    if github_app_id and github_private_key and github_installation_id:
        if GITHUB_AVAILABLE:
            try:
                github_toolset = create_github_toolset(
                    github_app_id, github_private_key, github_installation_id
                )
                # Add GitHub tools to the main toolset
                for tool in github_toolset.tools.values():
                    toolset.add_tool(tool)
            except Exception as e:
                # Log warning but don't fail - local git still works
                print(f"Warning: Could not initialize GitHub toolkit: {e}")
        else:
            print("Warning: GitHub integration requested but LangChain GitHub toolkit not available")

    return toolset


# Convenience functions for common use cases
def get_basic_git_tools(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """
    Get basic local git tools without GitHub integration.

    Args:
        config: Cyro configuration

    Returns:
        FunctionToolset with basic git tools
    """
    return create_git_toolset(config)


def get_safe_git_tools(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """
    Get safe git tools (read-only operations like status and log).

    Args:
        config: Cyro configuration

    Returns:
        FunctionToolset with read-only git tools
    """
    # Use centralized operations
    operations = _GitOperations(config=config)
    toolset = FunctionToolset()

    @toolset.tool
    def git_status(request: GitStatusRequest) -> GitStatusResult:
        """Get current git repository status including modified, added, deleted, and untracked files."""
        return operations.git_status(request)

    @toolset.tool
    def git_log(request: GitLogRequest) -> GitLogResult:
        """Get git commit history with optional filtering."""
        return operations.git_log(request)

    return toolset
