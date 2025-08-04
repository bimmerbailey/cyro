"""
Web and search tools for Cyro agents.

This module provides web interaction capabilities:
1. DuckDuckGo search via LangChain
2. Custom web fetching with BeautifulSoup
3. Rate limiting and security controls

Simplified: Focus on DuckDuckGo only, no browser automation.
"""

import time
from typing import Dict, List, Optional
from urllib.parse import urlparse

from pydantic import BaseModel, Field
from pydantic_ai.toolsets import FunctionToolset
from pydantic_ai.common_tools.duckduckgo import duckduckgo_search_tool
import requests
from bs4 import BeautifulSoup

from cyro.config.settings import CyroConfig


class WebSearchRequest(BaseModel):
    """Request model for web search operations."""

    query: str = Field(description="Search query")
    max_results: int = Field(default=5, description="Maximum number of results")
    domain_filter: Optional[str] = Field(
        default=None, description="Limit results to specific domain"
    )


class WebSearchResult(BaseModel):
    """Result model for web search operations."""

    results: List[Dict[str, str]] = Field(description="List of search results")
    query: str = Field(description="Original search query")
    total_results: int = Field(description="Number of results returned")


class WebFetchRequest(BaseModel):
    """Request model for web page fetching."""

    url: str = Field(description="URL to fetch")
    include_links: bool = Field(
        default=False, description="Include links in the content"
    )
    max_content_length: int = Field(
        default=10000, description="Maximum content length to return"
    )
    timeout: int = Field(default=30, description="Request timeout in seconds")


class WebFetchResult(BaseModel):
    """Result model for web page fetching."""

    content: str = Field(description="Page content")
    title: str = Field(description="Page title")
    url: str = Field(description="Final URL (after redirects)")
    status_code: int = Field(description="HTTP status code")
    content_type: str = Field(description="Content type")


class RateLimiter:
    """Simple rate limiter for web requests."""

    def __init__(self, calls_per_minute: int = 30):
        """Initialize rate limiter.

        Args:
            calls_per_minute: Maximum calls per minute
        """
        self.calls_per_minute = calls_per_minute
        self.calls = []

    def wait_if_needed(self):
        """Wait if rate limit would be exceeded."""
        now = time.time()
        # Remove calls older than 1 minute
        self.calls = [call_time for call_time in self.calls if now - call_time < 60]

        if len(self.calls) >= self.calls_per_minute:
            # Calculate wait time
            oldest_call = min(self.calls)
            wait_time = 60 - (now - oldest_call) + 1  # Add 1 second buffer
            if wait_time > 0:
                time.sleep(wait_time)
                # Clean up again after waiting
                now = time.time()
                self.calls = [
                    call_time for call_time in self.calls if now - call_time < 60
                ]

        self.calls.append(now)


class WebTools:
    """Web tools with custom web fetching (DuckDuckGo handled by PydanticAI)."""

    def __init__(self, config: Optional[CyroConfig] = None):
        """Initialize web tools.

        Args:
            config: Cyro configuration
        """
        self.config = config or CyroConfig()
        self.rate_limiter = RateLimiter(calls_per_minute=30)

    @staticmethod
    def _validate_url(url: str) -> bool:
        """Validate URL format and domain restrictions."""
        try:
            parsed = urlparse(url)
            if not parsed.scheme or not parsed.netloc:
                return False

            # Basic security checks
            if parsed.scheme not in ["http", "https"]:
                return False

            # TODO: What about local dev container?
            # Block localhost and private IPs (basic check)
            if parsed.hostname in ["localhost", "127.0.0.1", "0.0.0.0"]:
                return False

            return True
        except Exception:
            return False

    # Note: DuckDuckGo search is now handled by PydanticAI's native tool
    # We'll remove the custom web_search method since PydanticAI handles it better

    def web_fetch(self, request: WebFetchRequest) -> WebFetchResult:
        """Fetch content from a web page."""
        if not self._validate_url(request.url):
            raise ValueError(f"Invalid or restricted URL: {request.url}")

        self.rate_limiter.wait_if_needed()

        try:
            # Fetch the page
            headers = {"User-Agent": "Mozilla/5.0 (compatible; Cyro-Agent/1.0)"}

            response = requests.get(
                request.url,
                headers=headers,
                timeout=request.timeout,
                allow_redirects=True,
            )

            response.raise_for_status()

            # Parse content
            soup = BeautifulSoup(response.content, "html.parser")

            # Extract title
            title = soup.title.string.strip() if soup.title else "No title"

            # Extract text content
            # Remove script and style elements
            for script in soup(["script", "style"]):
                script.decompose()

            # Get text
            text = soup.get_text()

            # Clean up whitespace
            lines = (line.strip() for line in text.splitlines())
            chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
            content = " ".join(chunk for chunk in chunks if chunk)

            # Truncate if needed
            if len(content) > request.max_content_length:
                content = content[: request.max_content_length] + "..."

            # Extract links if requested
            if request.include_links:
                links = []
                for link in soup.find_all("a", href=True):
                    links.append({"text": link.get_text().strip(), "url": link["href"]})

                if links:
                    content += "\n\nLinks found:\n"
                    for link in links[:20]:  # Limit to first 20 links
                        content += f"- {link['text']}: {link['url']}\n"

            return WebFetchResult(
                content=content,
                title=title,
                url=response.url,
                status_code=response.status_code,
                content_type=response.headers.get("content-type", "unknown"),
            )

        except Exception as e:
            raise RuntimeError(f"Web fetch failed for {request.url}: {str(e)}")


def create_web_toolset(config: Optional[CyroConfig] = None) -> FunctionToolset:
    """Create a web toolset with PydanticAI DuckDuckGo and custom tools.

    Args:
        config: Cyro configuration

    Returns:
        FunctionToolset containing PydanticAI DuckDuckGo + custom web fetch
    """
    web_tools = WebTools(config)

    # Create FunctionToolset for all web operations
    toolset = FunctionToolset()

    # Add PydanticAI's native DuckDuckGo tool
    ddg_tool = duckduckgo_search_tool()
    toolset.add_tool(ddg_tool)

    # Add our custom web fetch tool
    @toolset.tool
    def web_fetch(request: WebFetchRequest) -> WebFetchResult:
        """Fetch and extract content from a web page with optional link extraction."""
        return web_tools.web_fetch(request)

    return toolset
