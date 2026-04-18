package mcp

import (
	"encoding/json"
	"fmt"

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

func handleToolCall(params map[string]interface{}) (callToolResult, bool) {
	toolName, ok := params["name"].(string)
	if !ok {
		return makeErrorResult("Missing 'name' parameter"), true
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		arguments = map[string]interface{}{}
	}

	var result interface{}
	var err error

	switch toolName {
	case "searxng_web_search":
		result, err = tools.Search(arguments)
	case "web_fetch":
		result, err = tools.Fetch(arguments)
	default:
		return makeErrorResult(fmt.Sprintf("Unknown tool: %s", toolName)), true
	}

	if err != nil {
		return makeErrorResult(err.Error()), true
	}

	return makeSuccessResult(result), false
}

func makeErrorResult(message string) callToolResult {
	return callToolResult{
		Content: []contentItem{
			{Type: "text", Text: message},
		},
		IsError: true,
	}
}

func makeSuccessResult(data interface{}) callToolResult {
	jsonBytes, _ := json.Marshal(data)
	return callToolResult{
		Content: []contentItem{
			{Type: "text", Text: string(jsonBytes)},
		},
		IsError: false,
	}
}
