package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/enrell/better-search-mcp/internal/config"
)

const version = "0.3.0"

func Run() {
	if config.LogLevel == "DEBUG" {
		config.LogMsg("DEBUG", fmt.Sprintf("Starting MCP server v%s", version))
		config.LogMsg("DEBUG", fmt.Sprintf("SEARXNG_URL: %s", config.SearxngURL))
		config.LogMsg("DEBUG", fmt.Sprintf("BYPARR_URL: %s", config.ByparrURL))
		config.LogMsg("DEBUG", fmt.Sprintf("MAX_CONCURRENT_REQUESTS: %d", config.MaxConcurrentRequests))
		config.LogMsg("DEBUG", fmt.Sprintf("MCP_TIMEOUT: %ds", config.MCPTimeout))
	}

	scanner := bufio.NewScanner(os.Stdin)
	var wg sync.WaitGroup

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		wg.Add(1)
		go func(data []byte) {
			defer wg.Done()
			handleRequest(data)
		}(append([]byte(nil), line...))
	}

	wg.Wait()

	if err := scanner.Err(); err != nil {
		config.LogMsg("ERROR", fmt.Sprintf("Scanner error: %v", err))
	}
}

func handleRequest(data []byte) {
	var req jsonRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendResponse(nil, nil, &jsonRPCError{Code: -32700, Message: "Parse error"})
		return
	}

	switch req.Method {
	case "initialize":
		result := map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "better-search-mcp",
				"version": version,
			},
		}
		sendResponse(req.ID, result, nil)

	case "tools/list":
		tools := getToolsList()
		sendResponse(req.ID, toolsListResult{Tools: tools}, nil)

	case "tools/call":
		var params map[string]interface{}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendResponse(req.ID, nil, &jsonRPCError{Code: -32602, Message: "Invalid params"})
			return
		}
		result, isError := handleToolCall(params)
		sendResponse(req.ID, result, nil)
		if isError {
			config.LogMsg("ERROR", fmt.Sprintf("Tool call error: %v", result))
		}

	case "ping":
		sendResponse(req.ID, map[string]interface{}{}, nil)

	default:
		if req.ID != nil {
			sendResponse(req.ID, nil, &jsonRPCError{Code: -32601, Message: "Method not found"})
		}
	}
}

func sendResponse(id *json.RawMessage, result interface{}, rpcErr *jsonRPCError) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		config.LogMsg("ERROR", fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}

	fmt.Println(string(data))
}
