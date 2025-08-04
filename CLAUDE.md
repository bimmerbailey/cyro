# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cyro is a terminal-based AI coding agent with dynamic subagent creation through markdown configuration files. It
connects to local Ollama by default (privacy-first) with support for multiple AI providers.

## Development Commands

### Setup

- `make install` - Install production dependencies only
- `make install-all` - Install all dependencies (production + dev + extras)

### Code Quality

- `make format` - Format code (ruff + isort)
- `make lint` - Check code quality (ruff)
- `make type-check` - Type checking (pyright)
- `make check` - Run lint + type-check
- `make fix` - Auto-fix linting issues and format
- `make all` - Format, check, and test

### Development

- `make run` or `uv run cyro` - Run the application
- `make build` - Build the package
- `make clean` - Clean build artifacts and cache

## Architecture Overview

### Core System Design

Cyro is built on **PydanticAI** which provides:

- Type-safe agent creation with dependency injection
- Built-in conversation management and streaming
- Automatic tool registration and schema generation
- Support for OpenAI, Anthropic, Gemini, Vertex AI, Groq

### Key Components

**CLI Layer (Typer + Rich)**

- `cyro` - Interactive terminal UI
- `cyro "prompt"` - Direct task execution (auto-routes to best agent)
- `cyro --agent <name> "prompt"` - Explicit agent selection
- `cyro chat` - Conversational mode
- `cyro agent` - Agent management (list, use, etc.)
- `cyro config` - Configuration management

**Subagent Framework**

- Markdown-based agent definitions in `agents/` directory
- Manager agent automatically routes tasks to best subagent based on descriptions
- Dynamic loading and task delegation (automatic and explicit)
- Tool permissions per agent type
- Specialized contexts (code review, debugging, testing)

**Model Providers**

- Default: Custom Ollama provider (local, privacy-first)
- Built-in: OpenAI, Anthropic, Gemini, Vertex AI, Groq via PydanticAI
- Configuration: TOML-based profiles

**Tools System**

- Built on PydanticAI's `@agent.tool` decorators
- Automatic schema generation and validation
- Third-party integration: LangChain, ACI.dev, MCP servers
- Custom tools: filesystem, code execution, Git, web search

### Module Structure

```
src/cyro/
├── cli/           # Typer commands (main, chat, agent, config)
├── agents/        # Subagent system (base, loader, manager, delegation)
├── models/        # Custom Ollama provider + profile management
├── tools/         # Custom tools using @agent.tool decorators
├── config/        # Settings, security policies, TOML config
└── utils/         # Console utilities, errors, auth
```

## Agent Configuration Format

Subagents are defined in markdown files:

```markdown
# Agent Name: Code Reviewer

## Description

Specialized agent for code review and quality assurance. Reviews code for bugs,
performance issues, maintainability, and coding standards compliance.

## Best For

- Code quality reviews
- Bug detection and analysis
- Performance optimization suggestions
- Coding standards enforcement
- Refactoring recommendations

## System Prompt

You are an expert code reviewer...

## Tools

- filesystem
- git

## Dependencies

- User context
- Project context

## Output Type

ReviewResult

## Scope

project
```

## Privacy & Security Principles

1. **Privacy-First**: No user data persistence, all processing in-memory only
2. **Local-First**: Default to Ollama to minimize external data sharing
3. **Zero Data Retention**: No conversation logging by default
4. **Directory Isolation**: Controlled file system access and command execution
5. **Session-Only Context**: Conversation state management without persistence

## Development Notes

- **Python 3.13+** required
- **UV package manager** for dependency management
- **PydanticAI integration**: Extend Agent classes, use @agent.tool decorators
- **Tool development**: All tools should use PydanticAI's built-in validation and retry mechanisms
- **Markdown parsing**: Agent configurations are parsed from markdown files in `agents/`
- **Type safety**: Leverage PydanticAI's generics for dependencies and outputs
- **Provider system**: Custom Ollama provider extends PydanticAI's model abstraction
- Use absolute import, not relative
- All import statements should be at the top level unless absolutely needed