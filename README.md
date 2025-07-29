# Cyro

A terminal-based AI coding agent that enables dynamic creation and management of specialized subagents through markdown configuration files. Unlike vendor-locked solutions, Cyro connects to your local Ollama server by default, giving you flexibility to use any compatible model.

## Overview

Cyro is a powerful AI-driven development assistant that operates directly in your terminal. The system allows you to create custom subagents by defining them in markdown files stored in the `agents/` directory, providing flexibility and extensibility for various coding tasks and workflows.

## Features

- **Terminal-based Interface**: Clean, efficient command-line interaction
- **Model Vendor Flexibility**: Connect to Ollama by default, with support for multiple AI providers
- **Dynamic Subagent Creation**: Define specialized agents using markdown files
- **Extensible Architecture**: Easy to add new agent types and capabilities
- **AI-Powered Code Assistance**: Intelligent code generation, debugging, and optimization

## Subagent System

Cyro uses a subagent architecture similar to Claude Code's sub-agents. Create specialized AI assistants by adding markdown configuration files to the `agents/` directory.

### How Subagents Work

- **Task-Specific Expertise**: Each subagent is designed for specific types of work (code review, debugging, testing, etc.)
- **Isolated Context**: Subagents operate with clean context windows for focused problem-solving
- **Dynamic Delegation**: The main agent automatically delegates tasks to appropriate subagents based on context
- **Explicit Invocation**: You can also explicitly request specific subagents

### Configuration

Each subagent is defined by a markdown file containing:

- **Name**: Unique identifier for the subagent
- **Description**: What the subagent specializes in
- **System Prompt**: Detailed instructions defining behavior and expertise
- **Tool Access**: Specific tools the subagent can use
- **Scope**: Project-level or user-level availability

### Examples

Create subagents for:
- Code review and quality assurance
- Debugging and error analysis  
- Test generation and validation
- Documentation writing
- Security analysis
- Performance optimization

## Getting Started

1. Clone the repository
2. Install dependencies: `uv install`
3. Run the agent: `uv run cyro`
4. Create custom agents in the `agents/` directory

## Requirements

- Python 3.13+
- UV package manager
- Ollama server running locally (default AI provider)

## License

MIT