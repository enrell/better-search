# Better Search MCP

An [MCP (Model Context Protocol)](https://modelcontextprotocol.io) server written in Go that provides web search and article-oriented page fetching through SearXNG and Byparr.

## Features

- Web search via a local SearXNG instance
- Single and batch page fetching through Byparr
- Article-focused extraction with heuristic content scoring
- HTML to Markdown conversion using a DOM-based renderer
- Structured MCP tool responses with `structuredContent` and `_meta`
- Config validation, request logging, and automated tests

## Prerequisites

- [SearXNG](https://docs.searxng.org/) running locally or remotely
- [Byparr](https://github.com/ThePhaseless/byparr) available as an HTTP service
- Go 1.23+ if you want to build from source

## Install

```bash
go install github.com/enrell/better-search-mcp@latest
```

The binary is installed as `better-search-mcp` in `$GOPATH/bin` or `$HOME/go/bin`.

## MCP Client Configuration

### OpenCode

```json
{
  "mcp": {
    "better-search": {
      "type": "local",
      "command": ["$HOME/go/bin/better-search-mcp"],
      "environment": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191",
        "LOG_LEVEL": "INFO"
      }
    }
  }
}
```

### Claude Code

```json
{
  "mcpServers": {
    "better-search": {
      "command": "$HOME/go/bin/better-search-mcp",
      "env": {
        "SEARXNG_URL": "http://localhost:8888",
        "BYPARR_URL": "http://localhost:8191",
        "LOG_LEVEL": "INFO"
      }
    }
  }
}
```

## Build From Source

```bash
git clone https://github.com/enrell/better-search-mcp.git
cd better-search-mcp
go build -o better-search-mcp ./cmd/server
```

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `SEARXNG_URL` | Base URL of your SearXNG instance. Must be `http` or `https`. | `http://localhost:8080` |
| `BYPARR_URL` | Base URL of your Byparr instance. Must be `http` or `https`. | `http://localhost:8191` |
| `LOG_LEVEL` | `DEBUG`, `INFO`, `WARN`, or `ERROR`. | `INFO` |
| `MCP_TIMEOUT` | Default timeout in seconds for outbound requests. | `30` |
| `MAX_CONCURRENT_REQUESTS` | Max parallel requests for batch fetch mode. | `30` |

Invalid configuration fails fast during startup.

## MCP Tools

### `searxng_web_search`

Searches SearXNG and returns:

- `success`
- `query`
- `results[]` with `title`, `url`, `snippet`, and `engine`

Parameters:

- `query` required string
- `num_results` optional number between `1` and `50`
- `language` optional string, defaults to `en`

Example:

```json
{
  "query": "golang mcp server",
  "num_results": 5,
  "language": "en"
}
```

### `web_fetch`

Fetches a single URL or a batch of URLs, extracts readable content, and converts it to Markdown.

Parameters:

- `url` optional string
- `urls` optional string array up to `25` items
- `include_metadata` optional boolean, defaults to `true`
- `timeout_seconds` optional number between `1` and `120`
- `max_content_chars` optional number for truncation
- `preserve_links` optional boolean, defaults to `true`
- `raw_html` optional boolean, defaults to `false`
- `prefer_readable_text` optional boolean, defaults to `true`
- `fail_fast` optional boolean for batch mode, defaults to `false`

Rules:

- Provide either `url` or `urls`, never both
- URLs must be valid `http` or `https`
- Duplicate batch URLs are removed automatically

Example, single fetch:

```json
{
  "url": "https://example.com/article",
  "include_metadata": true,
  "raw_html": true,
  "preserve_links": false,
  "max_content_chars": 4000
}
```

Example, batch fetch:

```json
{
  "urls": [
    "https://example.com/article-1",
    "https://example.com/article-2",
    "https://example.com/article-3"
  ],
  "timeout_seconds": 20,
  "fail_fast": true
}
```

## Response Shape

Each tool call returns:

- `content` with a JSON string for backwards compatibility
- `structuredContent` with the parsed result object
- `_meta` with tool metadata and schema version

Errors use a consistent shape inside `structuredContent`:

```json
{
  "success": false,
  "tool": "web_fetch",
  "error": {
    "code": "tool_error",
    "message": "..."
  },
  "generatedAt": "2026-04-18T12:00:00Z"
}
```

## Development

Run tests:

```bash
GOCACHE=/tmp/go-build go test ./...
```

Run locally with custom endpoints:

```bash
SEARXNG_URL=http://localhost:8888 \
BYPARR_URL=http://localhost:8191 \
LOG_LEVEL=DEBUG \
go run ./cmd/server
```

## Project Structure

```text
cmd/server/            binary entrypoint
internal/clients/      HTTP clients for SearXNG and Byparr
internal/config/       config loading and validation
internal/extractor/    content extraction and Markdown rendering
internal/mcp/          JSON-RPC / MCP server and tool registry
internal/tools/        tool orchestration and response models
```

## Troubleshooting

### Startup fails with configuration error

Check that `SEARXNG_URL` and `BYPARR_URL` are valid `http` or `https` base URLs with a host.

### `searxng_web_search` returns `success=false`

- Confirm SearXNG is reachable from the MCP process
- Verify the `/search?format=json` endpoint works manually
- Check logs with `LOG_LEVEL=DEBUG`

### `web_fetch` returns `Byparr error`

- Confirm Byparr is reachable at `BYPARR_URL`
- Verify Byparr can fetch the target page outside MCP
- Increase `timeout_seconds` for slower pages

### Extracted Markdown is incomplete

- Set `prefer_readable_text=false` to use the broader page HTML
- Set `raw_html=true` to inspect what was actually extracted
- Some highly dynamic pages may still degrade because the server only sees fetched HTML, not a browser DOM after client-side rendering

## License

MIT. See [LICENSE](LICENSE).
