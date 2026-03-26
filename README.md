# SearXNG Web Fetch MCP Server

An [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server written in Crystal that provides web search and content extraction capabilities through SearXNG and Byparr proxy.

## Features

- **Web Search**: Search the web using your local SearXNG instance
- **Web Page Fetching**: Extract clean, article-focused content from any URL
- **HTML to Markdown**: Converts extracted content to clean Markdown format
- **Trafilatura-style Extraction**: Smart content extraction that identifies the main article content

## Prerequisites

- [SearXNG](https://docs.searxng.org/) - A self-hosted metasearch engine
- [Byparr](https://github.com/ThePhaseless/byparr) - Anti-captcha proxy for web scraping

## Quick Start

### 1. Install the binary

```bash
curl -sL https://raw.githubusercontent.com/enrell/searxng-web-fetch-mcp/main/install.sh | bash
```

This downloads the latest release binary to `~/.local/bin/searxng-web-fetch-mcp`.

### 2. Configure your MCP client

Add to your MCP configuration file:

**For OpenCode:**
```json
{
  "mcp": {
    "searxng-web": {
      "type": "local",
      "command": ["~/.local/bin/searxng-web-fetch-mcp"],
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
    "searxng-web": {
      "command": "~/.local/bin/searxng-web-fetch-mcp",
      "env": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191"
      }
    }
  }
}
```

## Install Script

The install script automatically detects your platform and downloads the appropriate binary:

```bash
#!/bin/bash
set -e

REPO="enrell/searxng-web-fetch-mcp"
BIN_DIR="${HOME}/.local/bin"
BIN_NAME="searxng-web-fetch-mcp"
INSTALL_PATH="${BIN_DIR}/${BIN_NAME}"

mkdir -p "${BIN_DIR}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "${OS}" in
  linux)
    PLATFORM="linux-x86_64"
    ;;
  darwin)
    if [ "${ARCH}" = "arm64" ]; then
      PLATFORM="darwin-arm64"
    else
      PLATFORM="darwin-x86_64"
    fi
    ;;
  *)
    echo "Unsupported platform: ${OS}"
    exit 1
    ;;
esac

echo "Downloading searxng-web-fetch-mcp for ${PLATFORM}..."
curl -sL "https://github.com/${REPO}/releases/latest/download/searxng-web-fetch-mcp-${PLATFORM}" -o "${INSTALL_PATH}"
chmod +x "${INSTALL_PATH}"

echo "Installed to: ${INSTALL_PATH}"
```

## Usage

Ensure SearXNG and Byparr are running, then use the MCP as configured above.

**Environment Variables:**

| Variable | Description | Default |
|----------|-------------|---------|
| `SEARXNG_URL` | URL of your SearXNG instance | `http://localhost:8080` |
| `BYPARR_URL` | URL of your Byparr proxy | `http://localhost:8191` |
| `LOG_LEVEL` | Logging verbosity (DEBUG, INFO, WARN, ERROR) | `INFO` |

## MCP Tools

### `searxng_web_search`

Search the web using SearXNG.

**Parameters:**
- `query` (required): The search query
- `num_results` (optional): Number of results (default: 10)
- `language` (optional): Search language (default: "en")

### `web_fetch`

Fetch and extract content from a web page.

**Parameters:**
- `url` (required): The URL to fetch
- `include_metadata` (optional): Include metadata (default: true)

## Build from Source

Requires [Crystal](https://crystal-lang.org/) 1.19.1+:

```bash
git clone https://github.com/enrell/searxng-web-fetch-mcp.git
cd searxng-web-fetch-mcp
shards install --without development
crystal build src/searxng_web_fetch_mcp.cr -o searxng-web-fetch-mcp --release
```

## License

MIT License - see [LICENSE](LICENSE) file