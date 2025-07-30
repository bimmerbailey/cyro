"""
Custom exception hierarchy for Cyro.

This module defines custom exceptions for different error conditions
with proper error codes and user-friendly messages.
"""

from typing import Optional


class CyroError(Exception):
    """Base exception class for all Cyro errors."""

    def __init__(self, message: str, exit_code: int = 1):
        super().__init__(message)
        self.message = message
        self.exit_code = exit_code


class ConfigurationError(CyroError):
    """Raised when there are configuration-related errors."""

    def __init__(self, message: str, config_key: Optional[str] = None):
        super().__init__(message, exit_code=2)
        self.config_key = config_key


class AgentError(CyroError):
    """Raised when there are agent-related errors."""

    def __init__(self, message: str, agent_name: Optional[str] = None):
        super().__init__(message, exit_code=3)
        self.agent_name = agent_name


class ProviderError(CyroError):
    """Raised when there are AI provider-related errors."""

    def __init__(self, message: str, provider: Optional[str] = None):
        super().__init__(message, exit_code=4)
        self.provider = provider


class SecurityError(CyroError):
    """Raised when security policies are violated."""

    def __init__(self, message: str, operation: Optional[str] = None):
        super().__init__(message, exit_code=5)
        self.operation = operation


class ValidationError(CyroError):
    """Raised when input validation fails."""

    def __init__(self, message: str, field: Optional[str] = None):
        super().__init__(message, exit_code=6)
        self.field = field


class FileSystemError(CyroError):
    """Raised when file system operations fail."""

    def __init__(self, message: str, path: Optional[str] = None):
        super().__init__(message, exit_code=7)
        self.path = path


class NetworkError(CyroError):
    """Raised when network operations fail."""

    def __init__(self, message: str, url: Optional[str] = None):
        super().__init__(message, exit_code=8)
        self.url = url


class AuthenticationError(CyroError):
    """Raised when authentication fails."""

    def __init__(self, message: str, provider: Optional[str] = None):
        super().__init__(message, exit_code=9)
        self.provider = provider


class CLIError(CyroError):
    """Raised for CLI-specific errors."""

    def __init__(self, message: str, command: Optional[str] = None):
        super().__init__(message, exit_code=10)
        self.command = command


# Error code mappings for programmatic handling
ERROR_CODES = {
    1: "General Error",
    2: "Configuration Error",
    3: "Agent Error",
    4: "Provider Error",
    5: "Security Error",
    6: "Validation Error",
    7: "File System Error",
    8: "Network Error",
    9: "Authentication Error",
    10: "CLI Error",
}


def get_error_description(exit_code: int) -> str:
    """Get human-readable description for an error code."""
    return ERROR_CODES.get(exit_code, "Unknown Error")


def format_error_message(error: CyroError) -> str:
    """Format an error for display to users."""
    error_type = get_error_description(error.exit_code)

    message_parts = [f"[red]{error_type}:[/red] {error.message}"]

    # Add specific context based on error type
    if isinstance(error, ConfigurationError) and error.config_key:
        message_parts.append(f"Configuration key: {error.config_key}")
    elif isinstance(error, AgentError) and error.agent_name:
        message_parts.append(f"Agent: {error.agent_name}")
    elif isinstance(error, ProviderError) and error.provider:
        message_parts.append(f"Provider: {error.provider}")
    elif isinstance(error, SecurityError) and error.operation:
        message_parts.append(f"Operation: {error.operation}")
    elif isinstance(error, ValidationError) and error.field:
        message_parts.append(f"Field: {error.field}")
    elif isinstance(error, FileSystemError) and error.path:
        message_parts.append(f"Path: {error.path}")
    elif isinstance(error, NetworkError) and error.url:
        message_parts.append(f"URL: {error.url}")
    elif isinstance(error, AuthenticationError) and error.provider:
        message_parts.append(f"Provider: {error.provider}")
    elif isinstance(error, CLIError) and error.command:
        message_parts.append(f"Command: {error.command}")

    return "\n".join(message_parts)


# Common error factory functions
def config_not_found(config_path: str) -> ConfigurationError:
    """Create a configuration not found error."""
    return ConfigurationError(
        f"Configuration file not found: {config_path}", config_key="config_file"
    )


def agent_not_found(agent_name: str) -> AgentError:
    """Create an agent not found error."""
    return AgentError(
        f"Agent '{agent_name}' not found. Use 'cyro agent list' to see available agents.",
        agent_name=agent_name,
    )


def provider_unavailable(provider: str, reason: str) -> ProviderError:
    """Create a provider unavailable error."""
    return ProviderError(
        f"Provider '{provider}' is unavailable: {reason}", provider=provider
    )


def insufficient_permissions(operation: str) -> SecurityError:
    """Create an insufficient permissions error."""
    return SecurityError(
        f"Operation '{operation}' requires additional permissions. "
        f"Check security settings or use --force flag.",
        operation=operation,
    )


def invalid_input(field: str, value: str, expected: str) -> ValidationError:
    """Create an invalid input error."""
    return ValidationError(
        f"Invalid value for {field}: '{value}'. Expected: {expected}", field=field
    )


def file_not_accessible(path: str, reason: str) -> FileSystemError:
    """Create a file not accessible error."""
    return FileSystemError(f"Cannot access file '{path}': {reason}", path=path)


def connection_failed(url: str, reason: str) -> NetworkError:
    """Create a connection failed error."""
    return NetworkError(f"Failed to connect to '{url}': {reason}", url=url)


def auth_failed(provider: str, reason: str) -> AuthenticationError:
    """Create an authentication failed error."""
    return AuthenticationError(
        f"Authentication failed for provider '{provider}': {reason}", provider=provider
    )


def command_failed(command: str, reason: str) -> CLIError:
    """Create a command failed error."""
    return CLIError(f"Command '{command}' failed: {reason}", command=command)
