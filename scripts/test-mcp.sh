#!/bin/bash
set -e

MCP_BINARY="${MCP_BINARY:-${HOME}/.local/bin/searxng-web-fetch-mcp}"
SEARXNG_URL="${SEARXNG_URL:-http://localhost:8888}"
BYPARR_URL="${BYPARR_URL:-http://localhost:8191}"

if [ ! -f "$MCP_BINARY" ]; then
    echo "ERROR: MCP binary not found at $MCP_BINARY"
    echo "Set MCP_BINARY environment variable to override"
    exit 1
fi

echo "=== MCP Server Test Suite ==="
echo "Binary: $MCP_BINARY"
echo "SearXNG: $SEARXNG_URL"
echo "Byparr: $BYPARR_URL"
echo ""

test_initialize() {
    echo "Test: Initialize"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)
    
    if echo "$RESPONSE" | grep -q '"result"'; then
        echo "  PASS: Initialize"
        return 0
    else
        echo "  FAIL: Initialize"
        echo "  Response: $RESPONSE"
        return 1
    fi
}

test_tools_list() {
    echo "Test: Tools List"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)
    
    if echo "$RESPONSE" | grep -q '"searxng_web_search"' && echo "$RESPONSE" | grep -q '"web_fetch"'; then
        echo "  PASS: Tools List"
        return 0
    else
        echo "  FAIL: Tools List"
        echo "  Response: $RESPONSE"
        return 1
    fi
}

test_search() {
    echo "Test: Search"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"searxng_web_search","arguments":{"query":"crystal programming","num_results":3}}}' | SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)
    
    if echo "$RESPONSE" | grep -q '\\"success\\":true'; then
        echo "  PASS: Search"
        return 0
    else
        echo "  FAIL: Search"
        echo "  Response: $RESPONSE"
        return 1
    fi
}

test_web_fetch() {
    echo "Test: Web Fetch"
    RESPONSE=$(echo '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"web_fetch","arguments":{"url":"https://example.com/"}}}' | SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)
    
    if echo "$RESPONSE" | grep -q '\\"success\\":true'; then
        echo "  PASS: Web Fetch"
        return 0
    else
        echo "  FAIL: Web Fetch"
        echo "  Response: $RESPONSE"
        return 1
    fi
}

test_concurrent_requests() {
    echo "Test: Concurrent Requests"
    
    (
        RESPONSE=$(echo '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"searxng_web_search","arguments":{"query":"test1","num_results":1}}}' | SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)
        echo "$RESPONSE" > /tmp/mcp_test_1.txt
    ) &
    PID1=$!
    
    (
        RESPONSE=$(echo '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"searxng_web_search","arguments":{"query":"test2","num_results":1}}}' | SEARXNG_URL="$SEARXNG_URL" BYPARR_URL="$BYPARR_URL" "$MCP_BINARY" 2>/dev/null)
        echo "$RESPONSE" > /tmp/mcp_test_2.txt
    ) &
    PID2=$!
    
    wait $PID1
    wait $PID2
    
    RESPONSE1=$(cat /tmp/mcp_test_1.txt 2>/dev/null || echo "")
    RESPONSE2=$(cat /tmp/mcp_test_2.txt 2>/dev/null || echo "")
    
    if echo "$RESPONSE1" | grep -q '\\"success\\"' && echo "$RESPONSE2" | grep -q '\\"success\\"'; then
        echo "  PASS: Concurrent Requests"
        rm -f /tmp/mcp_test_1.txt /tmp/mcp_test_2.txt
        return 0
    else
        echo "  FAIL: Concurrent Requests"
        echo "  Response1: $RESPONSE1"
        echo "  Response2: $RESPONSE2"
        rm -f /tmp/mcp_test_1.txt /tmp/mcp_test_2.txt
        return 1
    fi
}

FAILED=0

test_initialize || FAILED=1
test_tools_list || FAILED=1

if [ "$SKIP_LIVE_TESTS" != "1" ]; then
    test_search || FAILED=1
    test_web_fetch || FAILED=1
    test_concurrent_requests || FAILED=1
else
    echo "Skipping live tests (SKIP_LIVE_TESTS=1)"
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "=== All tests passed ==="
    exit 0
else
    echo "=== Some tests failed ==="
    exit 1
fi
