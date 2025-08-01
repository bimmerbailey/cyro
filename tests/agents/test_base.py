"""
Tests for AgentConfig markdown parsing functionality.
"""

import pytest
from pathlib import Path
from tempfile import NamedTemporaryFile
from pydantic import BaseModel

from cyro.agents.base import AgentConfig, AgentMetadata


class MockResult(BaseModel):
    """Mock result type for AgentConfig testing."""
    answer: str
    confidence: float


class TestAgentConfigMarkdownParsing:
    """Test cases for AgentConfig markdown parsing."""

    def test_valid_yaml_frontmatter_parsing(self):
        """Test parsing valid YAML frontmatter with system prompt."""
        markdown_content = """---
name: test-agent
description: A test agent for unit testing
version: 2.0
tools: filesystem, git, web
---

You are a test agent. Your role is to help with testing and validation tasks.

## Instructions
- Follow test protocols
- Validate inputs carefully
- Provide clear feedback
"""
        
        config = AgentConfig.from_markdown(markdown_content)
        
        assert config.metadata.name == "test-agent"
        assert config.metadata.description == "A test agent for unit testing"
        assert config.metadata.version == "2.0"
        assert config.tools == ["filesystem", "git", "web"]
        assert "You are a test agent" in config.system_prompt
        assert "## Instructions" in config.system_prompt

    def test_minimal_valid_markdown(self):
        """Test parsing minimal valid markdown with required fields only."""
        markdown_content = """---
name: minimal-agent
description: Minimal test agent
---

Basic system prompt content."""
        
        config = AgentConfig.from_markdown(markdown_content)
        
        assert config.metadata.name == "minimal-agent"
        assert config.metadata.description == "Minimal test agent"  
        assert config.metadata.version == "1.0"  # default version
        assert config.tools is None
        assert config.system_prompt == "Basic system prompt content."

    def test_missing_required_name_field(self):
        """Test error handling when required 'name' field is missing."""
        markdown_content = """---
description: Agent without name
---

System prompt content."""
        
        with pytest.raises(ValueError, match="Agent missing required 'name' field"):
            AgentConfig.from_markdown(markdown_content)

    def test_missing_required_description_field(self):
        """Test error handling when required 'description' field is missing."""
        markdown_content = """---
name: no-description-agent
---

System prompt content."""
        
        with pytest.raises(ValueError, match="Agent missing required 'description' field"):
            AgentConfig.from_markdown(markdown_content)

    def test_invalid_yaml_frontmatter_format(self):
        """Test error handling for invalid YAML frontmatter format."""
        invalid_content = """# Not YAML frontmatter

This doesn't have proper frontmatter."""
        
        with pytest.raises(ValueError, match="Invalid agent format: missing YAML frontmatter"):
            AgentConfig.from_markdown(invalid_content)

    def test_empty_tools_field(self):
        """Test handling of empty tools field."""
        markdown_content = """---
name: empty-tools-agent
description: Agent with empty tools
tools: 
---

System prompt."""
        
        config = AgentConfig.from_markdown(markdown_content)
        # Empty tools field becomes empty string (not parsed as list)
        assert config.tools == ""

    def test_single_tool_parsing(self):
        """Test parsing single tool (no commas)."""
        markdown_content = """---
name: single-tool-agent
description: Agent with single tool
tools: filesystem
---

System prompt."""
        
        config = AgentConfig.from_markdown(markdown_content)
        assert config.tools == ["filesystem"]

    def test_tools_with_spaces(self):
        """Test parsing tools with spaces around commas."""
        markdown_content = """---
name: spaced-tools-agent
description: Agent with spaced tools
tools: filesystem , git,  web  , execution
---

System prompt."""
        
        config = AgentConfig.from_markdown(markdown_content)
        assert config.tools == ["filesystem", "git", "web", "execution"]

    def test_result_type_parameter(self):
        """Test passing result_type parameter to from_markdown."""
        markdown_content = """---
name: typed-agent
description: Agent with result type
---

System prompt."""
        
        config = AgentConfig.from_markdown(markdown_content, result_type=MockResult)
        
        assert config.result_type == MockResult
        assert config.metadata.name == "typed-agent"

    def test_bytes_input(self):
        """Test parsing markdown content passed as bytes."""
        markdown_content = b"""---
name: bytes-agent
description: Agent from bytes input
---

System prompt from bytes."""
        
        config = AgentConfig.from_markdown(markdown_content)
        
        assert config.metadata.name == "bytes-agent"
        assert config.system_prompt == "System prompt from bytes."

    def test_multiline_system_prompt(self):
        """Test parsing multiline system prompt with various formatting."""
        markdown_content = """---
name: multiline-agent
description: Agent with multiline prompt
---

You are a specialized agent with multiple responsibilities:

1. **Primary Function**: Handle complex analysis tasks
2. **Secondary Function**: Provide detailed explanations  
3. **Constraints**: 
   - Always validate inputs
   - Provide structured responses
   - Maintain professional tone

## Examples

Here are some example interactions:

- Question: "How do I optimize this?"
- Answer: "First analyze the current state..."

Remember to be thorough and accurate."""
        
        config = AgentConfig.from_markdown(markdown_content)
        
        prompt = config.system_prompt
        assert "You are a specialized agent" in prompt
        assert "**Primary Function**" in prompt
        assert "## Examples" in prompt
        assert "Remember to be thorough" in prompt

    def test_from_file_valid_path(self):
        """Test from_file method with valid file path."""
        markdown_content = """---
name: file-agent
description: Agent loaded from file
---

File-based system prompt."""
        
        with NamedTemporaryFile(mode='w', suffix='.md', delete=False) as tmp_file:
            tmp_file.write(markdown_content)
            tmp_file.flush()
            
            try:
                config = AgentConfig.from_file(Path(tmp_file.name))
                
                assert config.metadata.name == "file-agent"
                assert config.system_prompt == "File-based system prompt."
            finally:
                Path(tmp_file.name).unlink()

    def test_from_file_nonexistent_path(self):
        """Test from_file method with nonexistent file path."""
        nonexistent_path = Path("/nonexistent/path/agent.md")
        
        with pytest.raises(FileNotFoundError, match="Agent file not found"):
            AgentConfig.from_file(nonexistent_path)

    def test_from_file_with_result_type(self):
        """Test from_file method with result_type parameter."""
        markdown_content = """---
name: typed-file-agent
description: Typed agent from file
---

Typed system prompt."""
        
        with NamedTemporaryFile(mode='w', suffix='.md', delete=False) as tmp_file:
            tmp_file.write(markdown_content)
            tmp_file.flush()
            
            try:
                config = AgentConfig.from_file(Path(tmp_file.name), result_type=MockResult)
                
                assert config.result_type == MockResult
                assert config.metadata.name == "typed-file-agent"
            finally:
                Path(tmp_file.name).unlink()

    def test_complex_yaml_values(self):
        """Test parsing complex YAML values in frontmatter."""
        markdown_content = """---
name: complex-agent
description: Agent with complex YAML values
version: 1.5
tools: filesystem, git, web, execution, search
custom_field: some_value
---

Complex agent system prompt with multiple sections."""
        
        config = AgentConfig.from_markdown(markdown_content)
        
        assert config.metadata.name == "complex-agent"
        assert config.metadata.version == "1.5"
        assert len(config.tools) == 5
        assert "filesystem" in config.tools
        assert "search" in config.tools