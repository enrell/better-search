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

type server struct {
	cfg       config.Config
	writeMu   sync.Mutex
	requestWG sync.WaitGroup
}

func Run(cfg config.Config) {
	srv := &server{cfg: cfg}

	if cfg.LogLevel == "DEBUG" {
		cfg.LogMsg("DEBUG", fmt.Sprintf("Starting MCP server v%s", version))
		cfg.LogMsg("DEBUG", fmt.Sprintf("SEARXNG_URL: %s", cfg.SearxngURL))
		cfg.LogMsg("DEBUG", fmt.Sprintf("BYPARR_URL: %s", cfg.ByparrURL))
		cfg.LogMsg("DEBUG", fmt.Sprintf("MAX_CONCURRENT_REQUESTS: %d", cfg.MaxConcurrentRequests))
		cfg.LogMsg("DEBUG", fmt.Sprintf("MCP_TIMEOUT: %ds", cfg.MCPTimeout))
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if len(line) == 0 {
			continue
		}

		srv.requestWG.Add(1)
		go func(data []byte) {
			defer srv.requestWG.Done()
			srv.handleRequest(data)
		}(line)
	}

	srv.requestWG.Wait()

	if err := scanner.Err(); err != nil {
		cfg.LogMsg("ERROR", fmt.Sprintf("Scanner error: %v", err))
	}
}

func (s *server) handleRequest(data []byte) {
	var req jsonRPCRequest
	if err := json.Unmarshal(data, &req); err != nil {
		s.sendResponse(nil, nil, &jsonRPCError{Code: -32700, Message: "Parse error"})
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
		s.sendResponse(req.ID, result, nil)

	case "tools/list":
		tools := getToolsList()
		s.sendResponse(req.ID, toolsListResult{Tools: tools}, nil)

	case "tools/call":
		var params map[string]interface{}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendResponse(req.ID, nil, &jsonRPCError{Code: -32602, Message: "Invalid params"})
			return
		}

		result, isError := handleToolCall(s.cfg, params)
		s.sendResponse(req.ID, result, nil)
		if isError {
			s.cfg.LogMsg("ERROR", fmt.Sprintf("Tool call error: %v", result))
		}

	case "ping":
		s.sendResponse(req.ID, map[string]interface{}{}, nil)

	default:
		if req.ID != nil {
			s.sendResponse(req.ID, nil, &jsonRPCError{Code: -32601, Message: "Method not found"})
		}
	}
}

func (s *server) sendResponse(id *json.RawMessage, result interface{}, rpcErr *jsonRPCError) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   rpcErr,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		s.cfg.LogMsg("ERROR", fmt.Sprintf("Failed to marshal response: %v", err))
		return
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	_, _ = os.Stdout.Write(append(data, '\n'))
}
