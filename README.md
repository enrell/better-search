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

- [SearXNG](https://github.com/searxng/searxng) running locally or remotely
- [Byparr](https://github.com/ThePhaseless/byparr) available as an HTTP service
- Go 1.23+ if you want to build from source

## Install

### Option 1: Install with Go

```bash
go install github.com/enrell/better-search@latest
```

The binary is installed as `better-search` in `$GOPATH/bin` or `$HOME/go/bin`.

### Option 2: Install with a Code Agent

Use the prompt below with your preferred code agent if you want the agent to set up SearXNG, Byparr, validate both services, install this MCP, and configure it in your coding agents.

The prompt tells the agent to consult the official SearXNG container installation docs and the Byparr repository before making changes:

- SearXNG container install docs: `https://docs.searxng.org/admin/installation-docker.html#installation-container`
- Byparr repository: `https://github.com/ThePhaseless/Byparr`

Prompt:

<details>
<summary>Code agent install prompt</summary>

```text
Set up Better Search MCP end to end on this machine.

Requirements:
1. Before doing anything, read these sources and use them as the installation reference:
   - https://docs.searxng.org/admin/installation-docker.html#installation-container
   - https://github.com/ThePhaseless/Byparr
2. First inspect the machine and determine whether SearXNG and Byparr are already running and healthy.
3. If they are already running, do not reinstall them. Reuse the existing services.
4. If they are not running, install and start them.
5. Before creating any new container directories, inspect whether I already use a dedicated containers folder pattern, such as:
   - ~/Containers
   - ~/containers
   - ~/docker
   - ~/compose
   - any similar directory containing per-service folders and compose files
6. If I already have an established pattern for container projects, follow my existing convention and place new service folders there.
7. If I do not have an established pattern, ask me to choose between:
   - Docker Compose / docker compose
   - Docker CLI
   - Podman
   Default to Docker Compose for maintainability, but ask first if no existing convention is detected.
8. If Podman is installed and I prefer Podman, keep the setup aligned for Podman instead of Docker.
9. After starting SearXNG and Byparr, verify both with curl before continuing:
   - Verify SearXNG responds successfully on its configured local URL
   - Verify Byparr responds successfully on its configured local URL
10. Only after both services are confirmed healthy, install this MCP:
   - go install github.com/enrell/better-search@latest
11. Then configure the MCP for my installed code agents. Detect which coding agents I use and update their MCP config files if possible. At minimum, check for:
   - Claude Code
   - OpenCode
   - other local coding-agent MCP configs you can detect safely
12. Use these environment variables in the MCP config, matching the actual working local endpoints:
   - SEARXNG_URL
   - BYPARR_URL
   - LOG_LEVEL=INFO
13. Show me:
   - where you placed the SearXNG files
   - where you placed the Byparr files
   - the exact URLs used for both services
   - the MCP config changes you made
   - the curl commands and their outputs used to validate the services

Implementation details:
- Prefer reusing existing service directories and compose files if they already exist and are valid.
- If existing files are broken or incomplete, repair them instead of creating a parallel setup unless there is a strong reason not to.
- If SearXNG and Byparr are already healthy, skip directly to MCP installation and configuration.
- For SearXNG, prefer the documented compose-based setup when creating a new deployment, unless I explicitly prefer another method.
- For Byparr, use the repository’s documented container approach. If the repo already provides a compose file or compose-based workflow, prefer that when using Compose.
- Do not assume ports blindly; inspect actual running services and local configs first.
- Keep the setup local-only unless I explicitly ask to expose it publicly.
- Stop and ask before making destructive changes to an existing container stack.
```

</details>

## MCP Client Configuration

### OpenCode

```json
{
  "mcp": {
    "better-search": {
      "type": "local",
      "command": ["$HOME/go/bin/better-search"],
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
      "command": "$HOME/go/bin/better-search",
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
git clone https://github.com/enrell/better-search.git
cd better-search
go build -o better-search ./cmd/server
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
- Duplicate batch URLs are preserved in order, so batch results keep the same cardinality as the input list

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
