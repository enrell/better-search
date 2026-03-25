FROM --platform=linux/amd64 crystallang/crystal:1.20.0-dev-alpine-build AS builder
RUN apk add --no-cache git shards || true
WORKDIR /app
COPY shard.yml ./
COPY src/ ./src/
COPY spec/ ./spec/
RUN shards install --without development
RUN crystal build src/searxng_web_fetch_mcp.cr -o searxng-web-fetch-mcp --release
RUN ls -la /app/

FROM --platform=linux/amd64 alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates libgcc libstdc++
WORKDIR /app
COPY --from=builder /app/searxng-web-fetch-mcp /app/searxng-web-fetch-mcp
RUN ls -la /app/
RUN adduser -D -u 1000 mcpuser && chown mcpuser:mcpuser /app/searxng-web-fetch-mcp
USER mcpuser
ENV BYPARR_URL=http://byparr:8191
ENV SEARXNG_URL=http://searxng:8080
EXPOSE 8000
ENTRYPOINT ["/app/searxng-web-fetch-mcp"]
