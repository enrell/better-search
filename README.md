# SearXNG Web Fetch MCP Server

An [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server written in Crystal that provides web search and content extraction capabilities through SearXNG and an anti-captcha proxy.

## Features

- **Web Search**: Search the web using your local SearXNG instance
- **Web Page Fetching**: Extract clean, article-focused content from any URL
- **HTML to Markdown**: Converts extracted content to clean Markdown format
- **Trafilatura-style Extraction**: Smart content extraction that identifies the main article content

## Prerequisites

- [SearXNG](https://docs.searxng.org/) - A self-hosted metasearch engine
- [Byparr](https://github.com/ThePhaseless/byparr) - Anti-captcha proxy for web scraping

## Quick Start (npx)

**The easiest way to use this MCP server:**

Add to your `.claude.json`:

```json
{
  "mcpServers": {
    "searxng-web": {
      "command": "npx",
      "args": ["searxng-web-fetch-mcp"],
      "env": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191"
      }
    }
  }
}
```

Then reload MCP servers with `/mcp reload`.

The binary will be automatically downloaded on first run. No building or Docker required.

## Installation

### Option 1: npx (Recommended)

Run directly without installing:

```bash
npx searxng-web-fetch-mcp
```

Or install globally:

```bash
npm install -g searxng-web-fetch-mcp
searxng-web-fetch-mcp
```

### Option 2: Build from Source

Requires [Crystal](https://crystal-lang.org/) 1.19.1+

1. Clone the repository:

```bash
git clone https://github.com/enrell/searxng-web-fetch-mcp.git
cd searxng-web-fetch-mcp
```

2. Install dependencies and build:

```bash
shards install --without development
crystal build src/searxng_web_fetch_mcp.cr -o searxng-web-fetch-mcp --release
```

The compiled binary `searxng-web-fetch-mcp` will be in the current directory.

## Usage

Ensure SearXNG and Byparr are running:

- SearXNG: `http://localhost:8888`
- Byparr: `http://localhost:8191`

Run the MCP server:

```bash
SEARXNG_URL=http://localhost:8888 \
BYPARR_URL=http://localhost:8191 \
./searxng-web-fetch-mcp
```

The server provides two MCP tools:

### `searxng_web_search`

Search the web using SearXNG.

**Parameters:**

- `query` (required): The search query
- `num_results` (optional): Number of results to return (default: 10)
- `language` (optional): Search language (default: "en")

**Returns:** Search results with title, URL, snippet, and source engine.

### `web_fetch`

Fetch and extract content from a web page.

**Parameters:**

- `url` (required): The URL to fetch
- `include_metadata` (optional): Include metadata like title, author, date (default: true)

**Returns:** Clean Markdown content with optional metadata (title, author, date, language).

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `SEARXNG_URL` | URL of your SearXNG instance | `http://localhost:8080` |
| `BYPARR_URL` | URL of your Byparr proxy | `http://localhost:8191` |
| `LOG_LEVEL` | Logging verbosity (DEBUG, INFO, WARN, ERROR) | `INFO` |

## Claude Code Configuration

Add to your `.claude.json`:

```json
{
  "mcpServers": {
    "searxng-web": {
      "command": "/home/kokoro/lab/searxng-web-fetch-mcp/searxng-web-fetch-mcp",
      "env": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191"
      }
    }
  }
}
```

Then reload MCP servers in Claude Code with `/mcp reload`.

## Architecture

- **Language**: Crystal
- **HTTP Client**: Uses `connect-proxy` for proxy support via Byparr
- **HTML Parsing**: Lexbor for fast HTML parsing
- **Content Extraction**: Trafilatura-style algorithm to identify main content
- **Protocol**: MCP stdio server

### Content Extraction Algorithm

The extractor identifies main content by:

1. Removing script, style, navigation, and advertisement tags
2. Scoring elements based on:
   - Text density (link text vs content length)
   - Class/ID patterns (boosts `content`, `article`, `main`; penalizes `comment`, `sidebar`, `footer`)
3. Extracting metadata from Open Graph tags, meta tags, and HTML structure
4. Converting cleaned HTML to Markdown

## Development

Run tests:

```bash
crystal spec
```

Lint with Ameba:

```bash
./bin/ameba
```

## License

MIT License - see [LICENSE](LICENSE) file
