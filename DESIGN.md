# Cyro CLI Application Design

## System Overview

Cyro is a terminal-based AI coding agent with dynamic subagent creation through markdown configuration files. It
provides flexible AI model connectivity (Ollama default) and extensible agent architecture.

## Core Architecture Components

### 1. CLI Framework Layer

- **Technology**: Typer + Rich
- **Responsibility**: Command routing, terminal UX, user interaction
- **Command Modes**:
    - `cyro`: Interactive terminal UI
    - `cyro "prompt"`: Direct task execution (auto-routes to best agent)
    - `cyro --agent <name> "prompt"`: Explicit agent selection
    - `cyro chat`: Conversational mode
    - `cyro agent`: Agent management (list, use, etc.)
    - `cyro config`: Configuration management

### 2. AI Agent System

- **Technology**: PydanticAI (provides built-in conversation management, streaming, type safety)
- **Built-in Features**:
    - Type-safe agent creation with dependency injection
    - Conversation state management (session-only)
    - Structured response handling and validation
    - Automatic tool schema generation
- **Custom Components**:
    - Manager agent with intelligent routing based on agent descriptions
    - Task delegation logic (automatic and explicit)
    - Subagent orchestration

### 3. Subagent Framework

- **Configuration**: Markdown-based agent definitions
- **Discovery**: Dynamic loading from `agents/` directory
- **Specialization**: Task-specific contexts (code review, debugging, testing)
- **Access Control**: Tool permissions per agent type
- **Routing**: Manager agent automatically routes tasks to best subagent based on descriptions
- **User Control**: Users can explicitly select agents or let system auto-route

### 4. Model Provider System

- **Built-in Support**: PydanticAI provides OpenAI, Anthropic, Gemini, Vertex AI, Groq out of the box
- **Default**: Ollama (local, privacy-first) - we'll add custom provider
- **Configuration**: Profile-based provider management (TOML config)
- **Privacy**: Zero Data Retention by default, no conversation logging, local-first operation

### 5. Tool System

- **Built-in Capabilities**: PydanticAI provides tool registration, schema generation, validation, retry mechanisms
- **Third-party Integration**: LangChain tools, ACI.dev tools, MCP server tools (built into PydanticAI)
- **Custom Tools**: File operations, code execution, Git, web search
- **Security**: Directory isolation, controlled file access, command approval
- **Context Awareness**: PydanticAI's RunContext for dynamic tool behavior

## Proposed Module Structure

```
src/cyro/
├── __init__.py              # Main entry point and version
├── __main__.py              # CLI bootstrap
├── cli/
│   ├── __init__.py
│   ├── main.py              # Main CLI app with Typer
│   ├── chat.py              # Interactive chat commands
│   ├── agent.py             # Agent management commands
│   └── config.py            # Configuration commands
├── agents/
│   ├── __init__.py
│   ├── base.py              # Base agent classes (extends PydanticAI Agent)
│   ├── loader.py            # Markdown-based agent discovery and loading
│   ├── manager.py           # Agent lifecycle management
│   └── delegation.py        # Task routing logic
├── models/
│   ├── __init__.py
│   ├── ollama.py            # Custom Ollama provider for PydanticAI
│   └── profiles.py          # Provider profile management
├── tools/
│   ├── __init__.py
│   ├── filesystem.py        # File operations (using @agent.tool decorator)
│   ├── code.py              # Code execution tools
│   ├── git.py               # Git integration tools
│   ├── web.py               # Web search integration
│   └── sandbox.py           # Sandboxed execution
├── config/
│   ├── __init__.py
│   ├── settings.py          # Configuration models
│   ├── loader.py            # Config file handling
│   └── security.py          # Security policies and approval settings
└── utils/
    ├── __init__.py
    ├── console.py           # Rich console utilities
    ├── errors.py            # Custom exceptions
    └── auth.py              # API key management for cloud providers
```

## Agent Configuration Format

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

## What PydanticAI Provides Out of the Box

### Core Agent Framework

- **Type-safe agent creation** with generics for dependencies and outputs
- **Conversation management** with run/stream methods
- **Dependency injection** system for runtime context
- **Structured response handling** with automatic validation
- **Retry mechanisms** and error handling

### Model Provider Support

- **Built-in providers**: OpenAI, Anthropic, Gemini, Vertex AI, Groq
- **Model-agnostic design** - easy to switch between providers
- **Configurable at runtime** - no need to rebuild custom abstractions

### Tool Integration

- **Automatic tool registration** with `@agent.tool` decorators
- **Schema generation** from function signatures and docstrings
- **Tool validation** and retry mechanisms
- **Third-party tool support**: LangChain, ACI.dev, MCP servers
- **Context-aware tools** with RunContext

### What We Need to Build

- **Custom Ollama provider** (PydanticAI doesn't include this yet)
- **Markdown-based agent configuration** parser
- **CLI interface** with Typer and Rich
- **Task delegation system** for routing between subagents
- **Security/sandboxing** layer for safe code execution
- **Profile management** system for different configurations

## Implementation Phases

### Phase 1: Foundation (Core Infrastructure)

1. **CLI Framework Setup**
    - Implement Typer-based command structure
    - Set up Rich console for beautiful terminal output
    - Create basic command scaffolding (chat, agent, config)

2. **Configuration System**
    - Design settings models with Pydantic
    - Implement config file loading/saving
    - Add model provider configuration

3. **Basic AI Integration**
    - Create custom Ollama provider for PydanticAI
    - Set up basic PydanticAI agent with chat interface
    - Configure provider profiles and authentication

### Phase 2: Agent Architecture

1. **Agent Framework**
    - Extend PydanticAI Agent classes for subagent system
    - Implement markdown-based agent configuration parser
    - Create agent discovery and loading system

2. **Tool System Foundation**
    - Implement custom tools using PydanticAI's @agent.tool decorators
    - Build core tools (filesystem, basic code execution)
    - Add tool access control and permissions

3. **Agent Management**
    - Build agent lifecycle management
    - Create CLI commands for agent operations
    - Implement agent listing and status reporting

### Phase 3: Advanced Features

1. **Task Delegation**
    - Implement manager agent with intelligent routing based on agent descriptions
    - Add automatic agent selection with transparent routing
    - Support explicit agent selection via CLI flags
    - Create agent communication protocols

2. **Extended Tool Integration**
    - Add Git integration tools
    - Implement code analysis and execution tools
    - Create project-aware context understanding

3. **User Experience**
    - Enhance interactive chat with PydanticAI's streaming capabilities
    - Add agent marketplace/sharing capabilities (markdown-based)
    - Implement rich error handling and help system

## Key Design Principles

1. **Privacy-First**: No user data persistence, all processing in-memory only
2. **Modularity**: Each component should be independently testable and replaceable
3. **Extensibility**: Easy to add new agents, tools, and model providers
4. **Security**: Configurable sandboxing and granular approval controls
5. **User Experience**: Clean terminal interface with rich feedback and multimodal support
6. **Flexibility**: Support for different AI providers and deployment scenarios
7. **Local-First**: Default to local models (Ollama) to minimize external data sharing