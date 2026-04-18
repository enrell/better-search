package mcp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/enrell/better-search-mcp/internal/config"
	"github.com/enrell/better-search-mcp/internal/tools"
)

func getToolsList() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "searxng_web_search",
			Description: "Search the web using a local SearXNG instance. Returns search results with title, url, and snippet.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
					"num_results": map[string]interface{}{
						"type":        "number",
						"description": "Number of results to return (default: 10)",
						"default":     10,
					},
					"language": map[string]interface{}{
						"type":        "string",
						"description": "Search language (default: en)",
						"default":     "en",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "web_fetch",
			Description: "Fetch one or more web pages and extract their main content as clean Markdown. Supports parallel batch fetching for multiple URLs.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "The URL to fetch (use 'urls' for batch fetching)",
					},
					"urls": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Array of URLs to fetch in parallel (faster than sequential)",
					},
					"include_metadata": map[string]interface{}{
						"type":        "boolean",
						"description": "Include metadata like title, author, date (default: true)",
						"default":     true,
					},
				},
			},
		},
	}
}

func handleToolCall(cfg config.Config, params map[string]interface{}) (callToolResult, bool) {
	toolName, ok := params["name"].(string)
	if !ok {
		return makeErrorResult("unknown", "invalid_request", "Missing 'name' parameter", nil), true
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = map[string]interface{}{}
	}

	var result interface{}
	var err error

	switch toolName {
	case "searxng_web_search":
		result, err = tools.Search(cfg, arguments)
	case "web_fetch":
		result, err = tools.Fetch(cfg, arguments)
	default:
		return makeErrorResult(toolName, "unknown_tool", fmt.Sprintf("Unknown tool: %s", toolName), nil), true
	}

	if err != nil {
		return makeErrorResult(toolName, "tool_error", err.Error(), nil), true
	}

	return makeSuccessResult(toolName, result), false
}

func makeErrorResult(toolName, code, message string, details interface{}) callToolResult {
	payload := map[string]interface{}{
		"success":     false,
		"tool":        toolName,
		"error":       map[string]interface{}{"code": code, "message": message},
		"generatedAt": time.Now().UTC().Format(time.RFC3339),
	}
	if details != nil {
		payload["error"].(map[string]interface{})["details"] = details
	}

	return callToolResult{
		Content: []contentItem{
			{Type: "text", Text: message},
		},
		StructuredContent: payload,
		Meta: map[string]interface{}{
			"tool":          toolName,
			"schemaVersion": "2026-04-18",
		},
		IsError: true,
	}
}

func makeSuccessResult(toolName string, data interface{}) callToolResult {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return makeErrorResult(toolName, "marshal_error", "Failed to serialize tool result", err.Error())
	}

	return callToolResult{
		Content: []contentItem{
			{Type: "text", Text: string(jsonBytes)},
		},
		StructuredContent: data,
		Meta: map[string]interface{}{
			"tool":          toolName,
			"schemaVersion": "2026-04-18",
			"generatedAt":   time.Now().UTC().Format(time.RFC3339),
		},
		IsError: false,
	}
}
