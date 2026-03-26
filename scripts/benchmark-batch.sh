#!/bin/bash

MCP_BINARY="${MCP_BINARY:-${HOME}/.local/bin/searxng-web-fetch-mcp}"
SEARXNG_URL="${SEARXNG_URL:-http://localhost:8888}"
BYPARR_URL="${BYPARR_URL:-http://localhost:8191}"
NUM_URLS="${NUM_URLS:-10}"

if [ ! -f "$MCP_BINARY" ]; then
    echo "ERROR: MCP binary not found at $MCP_BINARY"
    exit 1
fi

echo "=== MCP Server Batch Fetch Benchmark ==="
echo "Binary: $MCP_BINARY"
echo "URLs per batch: $NUM_URLS"
echo ""

URLS_JSON="["
for i in $(seq 1 $NUM_URLS); do
    if [ $i -gt 1 ]; then
        URLS_JSON+=","
    fi
    URLS_JSON+="\"https://example.com/?page=$i\""
done
URLS_JSON+="]"

echo "Testing batch fetch with $NUM_URLS URLs..."
start_time=$(date +%s%N)

RESPONSE=$(echo "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"web_fetch\",\"arguments\":{\"urls\":$URLS_JSON}}}" | \
    SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)

end_time=$(date +%s%N)
wall_time=$(( (end_time - start_time) / 1000000 ))

SUCCESS=$(echo "$RESPONSE" | grep -o '"success":true' | wc -l)

if echo "$RESPONSE" | grep -q '"result"'; then
    RPS=$(( NUM_URLS * 1000 / wall_time ))
    echo ""
    echo "=== Results ==="
    echo "URLs fetched:    $NUM_URLS"
    echo "Total time:       ${wall_time}ms"
    echo "URLs/sec:        $RPS"
    echo "Avg per URL:     $((wall_time / NUM_URLS))ms"
    echo ""
    
    if [ $RPS -gt 50 ]; then
        echo "Rating: Excellent (>$RPS URLs/s)"
    elif [ $RPS -gt 20 ]; then
        echo "Rating: Good ($RPS URLs/s)"
    elif [ $RPS -gt 10 ]; then
        echo "Rating: Moderate ($RPS URLs/s)"
    else
        echo "Rating: Slow ($RPS URLs/s)"
    fi
else
    echo "FAILED: $RESPONSE"
fi
