# Better Search MCP

An [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server written in Go that provides web search and content extraction capabilities through SearXNG and Byparr proxy.

## Features

- **Web Search**: Search the web using your local SearXNG instance
- **Web Page Fetching**: Extract clean, article-focused content from any URL
- **Batch Fetching**: Fetch multiple URLs in parallel with concurrent processing
- **HTML to Markdown**: Converts extracted content to clean Markdown format
- **Trafilatura-style Extraction**: Smart content extraction that identifies the main article content
- **Easy Installation**: Install with `go install`

## Prerequisites

- [SearXNG](https://docs.searxng.org/) - A self-hosted metasearch engine
- [Byparr](https://github.com/ThePhaseless/byparr) - Anti-captcha proxy for web scraping

## Quick Start

### 1. Install with Go

```bash
go install github.com/enrell/better-search-mcp@latest
```

The binary will be installed to `$GOPATH/bin/better-search-mcp` (usually `~/go/bin/`).

### 2. Configure your MCP client

Add to your MCP configuration file:

**For OpenCode:**
```json
{
  "mcp": {
    "better-search": {
      "type": "local",
      "command": ["$HOME/go/bin/better-search-mcp"],
      "environment": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191"
      }
    }
  }
}
```

**For Claude Code (.claude.json):**
```json
{
  "mcpServers": {
    "better-search": {
      "command": "$HOME/go/bin/better-search-mcp",
      "env": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191"
      }
    }
  }
}
```

> **Note:** Replace `$HOME` with your actual home directory path if your client doesn't expand environment variables.

## Build from Source

Requires [Go](https://go.dev/) 1.23+:

```bash
git clone https://github.com/enrell/better-search-mcp.git
cd better-search-mcp
go build -o better-search-mcp .
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARXNG_URL` | URL of your SearXNG instance | `http://localhost:8080` |
| `BYPARR_URL` | URL of your Byparr proxy | `http://localhost:8191` |
| `LOG_LEVEL` | Logging verbosity (DEBUG, INFO, WARN, ERROR) | `INFO` |
| `MCP_TIMEOUT` | Request timeout in seconds | `30` |
| `MAX_CONCURRENT_REQUESTS` | Max parallel requests for batch fetching | `30` |

## MCP Tools

### `searxng_web_search`

Search the web using SearXNG.

**Parameters:**
- `query` (required): The search query
- `num_results` (optional): Number of results (default: 10)
- `language` (optional): Search language (default: "en")

### `web_fetch`

Fetch and extract content from web pages. Supports single URL or batch fetching.

**Parameters:**
- `url` (optional): The URL to fetch
- `urls` (optional): Array of URLs to fetch in parallel
- `include_metadata` (optional): Include metadata (default: true)

**Batch Fetching Example:**
```json
{
  "urls": [
    "https://example.com/article1",
    "https://example.com/article2",
    "https://example.com/article3"
  ]
}
```

## License

MIT License - see [LICENSE](LICENSE) file
