# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Context

This project creates a minimal production-ready MCP (Model Context Protocol) server in pure Crystal that provides two tools for local LLMs (LM Studio, Open WebUI):

1. **`searxng_web_search`** – Queries a local SearXNG instance for web search results
2. **`web_fetch`** – Fetches web pages through Byparr proxy and extracts clean Markdown using a ported version of go-trafilatura's extraction logic

The goal is minimal dependencies, high performance, and full Docker deployment.

---

## Project Setup

### shard.yml

```yaml
name: searxng-web-fetch-mcp
version: 0.1.0

authors:
  - enrell <enrellsa10@proton.me>

targets:
  searxng-web-fetch-mcp:
    main: src/searxng_web_fetch_mcp.cr

crystal: '>= 1.19.1'

license: MIT

dependencies:
  mcp:
    github: ralsina/mcp
    branch: master
  connect-proxy:
    github: spider-gazelle/connect-proxy
  lexbor:
    github: kostya/lexbor

development_dependencies:
  ameba:
    github: crystal-ameba/ameba
```

### Folder Structure

```
searxng-web-fetch-mcp/
├── src/
│   ├── searxng_web_fetch_mcp.cr      # Entry point & stdio handler
│   ├── tools/
│   │   ├── searxng_web_search.cr     # Tool 1: Search
│   │   └── web_fetch.cr              # Tool 2: Fetch + Extract
│   ├── extraction/
│   │   └── trafilatura_extractor.cr  # Ported extraction core (consolidated)
│   └── utils/
│       └── html_to_markdown.cr       # HTML→Markdown converter
├── docker/
│   ├── Dockerfile                     # Multi-stage production build
│   └── docker-compose.yml             # Full stack orchestration
├── spec/
├── .gitignore
├── shard.yml
└── README.md
```

---

## Architecture

### Tool Pattern (MCP::AbstractTool)

Tools inherit from `MCP::AbstractTool` and auto-register:

```crystal
class MyTool < MCP::AbstractTool
 @@tool_name = "my_tool"
 @@tool_description = "Description"
 @@tool_input_schema = {"type" => "object", "properties" => {...}}.to_json

 def invoke(params : Hash(String, JSON::Any), env : HTTP::Server::Context? = nil)
 # returns Hash(String, JSON::Any)
 end
end
```

### Byparr Proxy Injection

```crystal
BYPARR_URL = ENV.fetch("BYPARR_URL", "http://localhost:8191")
SEARXNG_URL = ENV.fetch("SEARXNG_URL", "http://localhost:8080")
```

HTTP requests through Byparr use the proxy environment variable:
```crystal
require "connect-proxy/ext/http-client"
# Set http_proxy/BYPARR_URL, then HTTP::Client respects it automatically
```

---

## Docker Deployment

### Dockerfile (Multi-stage)

```dockerfile
FROM crystallang/crystal:1.19.1-alpine AS builder
RUN apk add --no-cache git shards
WORKDIR /app
COPY shard.yml ./
RUN shards install --without development
COPY src/ ./src/
RUN crystal build src/searxng_web_fetch_mcp.cr -r static -o /searxng-web-fetch-mcp --release

FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates libgcc libstdc++
WORKDIR /app
COPY --from=builder /searxng-web-fetch-mcp .
RUN adduser -D -u 1000 mcpuser && chown mcpuser:mcpuser /searxng-web-fetch-mcp
USER mcpuser
ENV BYPARR_URL=http://byparr:8191
ENV SEARXNG_URL=http://searxng:8080
EXPOSE 8000
ENTRYPOINT ["/app/searxng-web-fetch-mcp"]
```

### docker-compose.yml (3 Services)

```yaml
services:
  mcp-server:
    build: .
    container_name: searxng-web-fetch-mcp
    ports: ["8000:8000"]
    environment:
      - BYPARR_URL=http://byparr:8191
      - SEARXNG_URL=http://searxng:8080
      - LOG_LEVEL=info
    depends_on:
      byparr: { condition: service_healthy }
      searxng: { condition: service_healthy }
    networks: [mcp-network]
    restart: unless-stopped

  byparr:
    image: ghcr.io/thephaseless/byparr:latest
    container_name: byparr
    ports: ["8191:8191"]
    environment: { HOST: "0.0.0.0", PORT: 8191 }
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8191"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks: [mcp-network]
    restart: unless-stopped

  searxng:
    image: searxng/searxng:latest
    container_name: searxng
    ports: ["8888:8080"]
    environment:
      - SEARXNG_BASE_URL=http://localhost:8888
      - SEARXNG_SECRET=dev-secret-change-in-prod
      - SEARXNG_general.safe_search=0
      - SEARXNG_search.max_results=20
    volumes:
      - searxng-data:/var/cache/searxng
      - searxng-config:/etc/searxng
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks: [mcp-network]
    restart: unless-stopped
    command: sh -c "sed -i 's/127.0.0.1/0.0.0.0/g' /etc/searxng/settings.yml && /docker-entrypoint.sh"

networks:
  mcp-network:
    driver: bridge

volumes:
  searxng-data:
  searxng-config:
```

---

## Build Commands

```bash
# Install dependencies
shards install

# Build release binary
crystal build src/searxng_web_fetch_mcp.cr -o bin/searxng-web-fetch-mcp --release

# Build for development
crystal build src/searxng_web_fetch_mcp.cr -o bin/searxng-web-fetch-mcp

# Run tests
crystal spec

# Run linter
./bin/ameba

# Verify MCP protocol works
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | crystal run src/searxng_web_fetch_mcp.cr
```

---

## Configuration

Environment variables:
- `SEARXNG_URL` – SearXNG instance URL (default: `http://localhost:8080`)
- `BYPARR_URL` – Byparr proxy URL (default: `http://localhost:8191`)
- `LOG_LEVEL` – Logging level: `debug`/`info`/`warn`/`error` (default: `INFO`)
- `MCP_TIMEOUT` – Request timeout in seconds (default: `30`)

---

## Extraction Logic Flow

Ported from go-trafilatura:

1. Parse HTML with lexbor
2. Extract metadata (title, date, author, language)
3. Find main content using heuristics:
   - Primary: semantic elements (`<article>`, `<main>`, `[role=main]`)
   - Secondary: text density + link density scoring
4. Clean DOM (remove script, style, nav, aside, footer, etc.)
5. Convert cleaned HTML to Markdown via `Utils::HtmlToMarkdown`

Scoring algorithm:
- High text-to-link ratio (low link density)
- Boost: class/id containing "content", "article"
- Penalty: class/id containing "comment", "sidebar", "footer"

---

## Implementation Roadmap

**Phase 1**: Basic MCP server skeleton (complete)
**Phase 2**: Implement `searxng_web_search` tool
**Phase 3**: Implement `web_fetch` with Byparr proxy
**Phase 4**: Port trafilatura extraction (heuristics + metadata)
**Phase 5**: HTML to Markdown converter
**Phase 6**: Error handling, logging, timeouts
**Phase 7**: Docker production build
**Phase 8**: Testing and verification

---

## Key Dependencies

- `ralsina/mcp` – MCP framework (MCP::AbstractTool, MCP::StdioHandler)
- `spider-gazelle/connect-proxy` – HTTP proxy support
- `kostya/lexbor` – HTML5 parser with CSS selectors

---

## Verification

```bash
# 1. Install dependencies
shards install

# 2. Build locally
crystal build src/searxng_web_fetch_mcp.cr -o bin/searxng-web-fetch-mcp

# 3. Run with Docker
docker compose -f docker/docker-compose.yml up -d

# 4. Test MCP protocol
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./bin/searxng-web-fetch-mcp

# Expected: Initialize response; tools/list shows the two registered tools; tool calls return results
```
