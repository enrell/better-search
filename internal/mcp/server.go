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

func Run(cfg config.Config) {
	if cfg.LogLevel == "DEBUG" {
		cfg.LogMsg("DEBUG", fmt.Sprintf("Starting MCP server v%s", version))
		cfg.LogMsg("DEBUG", fmt.Sprintf("SEARXNG_URL: %s", cfg.SearxngURL))
		cfg.LogMsg("DEBUG", fmt.Sprintf("BYPARR_URL: %s", cfg.ByparrURL))
		cfg.LogMsg("DEBUG", fmt.Sprintf("MAX_CONCURRENT_REQUESTS: %d", cfg.MaxConcurrentRequests))
		cfg.LogMsg("DEBUG", fmt.Sprintf("MCP_TIMEOUT: %ds", cfg.MCPTimeout))
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
			handleRequest(cfg, data)
		}(append([]byte(nil), line...))
	}

	wg.Wait()

	if err := scanner.Err(); err != nil {
		cfg.LogMsg("ERROR", fmt.Sprintf("Scanner error: %v", err))
	}
}

func handleRequest(cfg config.Config, data []byte) {
	var req jsonRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		sendResponse(cfg, nil, nil, &jsonRPCError{Code: -32700, Message: "Parse error"})
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
		sendResponse(cfg, req.ID, result, nil)

	case "tools/list":
		tools := getToolsList()
		sendResponse(cfg, req.ID, toolsListResult{Tools: tools}, nil)

	case "tools/call":
		var params map[string]interface{}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendResponse(cfg, req.ID, nil, &jsonRPCError{Code: -32602, Message: "Invalid params"})
			return
		}
		result, isError := handleToolCall(cfg, params)
		sendResponse(cfg, req.ID, result, nil)
		if isError {
			cfg.LogMsg("ERROR", fmt.Sprintf("Tool call error: %v", result))
		}

	case "ping":
		sendResponse(cfg, req.ID, map[string]interface{}{}, nil)

	default:
		if req.ID != nil {
			sendResponse(cfg, req.ID, nil, &jsonRPCError{Code: -32601, Message: "Method not found"})
		}
	}
}

func sendResponse(cfg config.Config, id *json.RawMessage, result interface{}, rpcErr *jsonRPCError) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		cfg.LogMsg("ERROR", fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}

	fmt.Println(string(data))
}
