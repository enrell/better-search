package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/enrell/better-search/internal/clients/byparr"
	"github.com/enrell/better-search/internal/clients/searxng"
	"github.com/enrell/better-search/internal/config"
)

type fakeSearxngClient struct {
	response searxng.SearchResponse
	err      error
}

func (f fakeSearxngClient) Search(context.Context, string, int, string) (searxng.SearchResponse, error) {
	return f.response, f.err
}

type fakeByparrClient struct {
	responses map[string]byparr.Response
	errors    map[string]error
}

func (f fakeByparrClient) Fetch(_ context.Context, rawURL string, _ int) (byparr.Response, error) {
	if err := f.errors[rawURL]; err != nil {
		return byparr.Response{}, err
	}
	if response, ok := f.responses[rawURL]; ok {
		return response, nil
	}
	return byparr.Response{}, errors.New("unexpected url")
}

func TestSearchUsesSearxngClient(t *testing.T) {
	previousFactory := searxngClientFactory
	searxngClientFactory = func(cfg config.Config) searxngSearcher {
		return fakeSearxngClient{
			response: searxng.SearchResponse{
				Results: []searxng.SearchResult{
					{
						Title:   "Go",
						URL:     "https://go.dev",
						Content: "The Go Programming Language",
						Engine:  "test",
					},
				},
			},
		}
	}
	t.Cleanup(func() { searxngClientFactory = previousFactory })

	cfg := config.Config{LogLevel: "ERROR", MCPTimeout: 5}
	resultRaw, err := Search(cfg, map[string]interface{}{
		"query":       "golang",
		"num_results": float64(1),
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	result := resultRaw.(SearchResponse)
	if !result.Success || len(result.Results) != 1 {
		t.Fatalf("unexpected result payload: %+v", result)
	}
	if result.Results[0].URL != "https://go.dev" {
		t.Fatalf("unexpected result url: %s", result.Results[0].URL)
	}
}

func TestFetchSupportsReadableTextAndRawHTML(t *testing.T) {
	previousFactory := byparrClientFactory
	byparrClientFactory = func(cfg config.Config) byparrFetcher {
		response := byparr.Response{Status: "ok"}
		response.Solution.Response = `<html><body><article><h1>Title</h1><p>Hello <a href="https://example.com">world</a>.</p></article></body></html>`
		return fakeByparrClient{
			responses: map[string]byparr.Response{
				"https://example.com/article": response,
			},
		}
	}
	t.Cleanup(func() { byparrClientFactory = previousFactory })

	cfg := config.Config{LogLevel: "ERROR", MCPTimeout: 5}
	resultRaw, err := Fetch(cfg, map[string]interface{}{
		"url":                  "https://example.com/article",
		"raw_html":             true,
		"preserve_links":       false,
		"max_content_chars":    float64(200),
		"include_metadata":     true,
		"timeout_seconds":      float64(3),
		"prefer_readable_text": true,
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	result := resultRaw.(FetchResult)
	if !result.Success {
		t.Fatalf("expected success result, got: %+v", result)
	}
	if strings.Contains(result.Text, "[world](") {
		t.Fatalf("expected links to be stripped, got: %s", result.Text)
	}
	if !strings.Contains(result.Text, "Hello world.") {
		t.Fatalf("expected readable text, got: %s", result.Text)
	}
	if !strings.Contains(result.RawHTML, "<article>") {
		t.Fatalf("expected raw html to include article markup, got: %s", result.RawHTML)
	}
	if result.Metadata == nil || result.Metadata.Title != "Title" {
		t.Fatalf("expected metadata title, got: %+v", result.Metadata)
	}
}

func TestFetchBatchFailFastStopsAfterFirstFailure(t *testing.T) {
	previousFactory := byparrClientFactory
	byparrClientFactory = func(cfg config.Config) byparrFetcher {
		okResponse := byparr.Response{Status: "ok"}
		okResponse.Solution.Response = `<html><body><article><p>ok</p></article></body></html>`
		return fakeByparrClient{
			responses: map[string]byparr.Response{
				"https://example.com/ok-1": okResponse,
				"https://example.com/ok-2": okResponse,
			},
			errors: map[string]error{
				"https://example.com/fail": errors.New("blocked"),
			},
		}
	}
	t.Cleanup(func() { byparrClientFactory = previousFactory })

	cfg := config.Config{LogLevel: "ERROR", MCPTimeout: 5, MaxConcurrentRequests: 5}
	resultRaw, err := Fetch(cfg, map[string]interface{}{
		"urls": []interface{}{
			"https://example.com/ok-1",
			"https://example.com/fail",
			"https://example.com/ok-2",
		},
		"fail_fast": true,
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	result := resultRaw.(BatchFetchResponse)
	if result.Count != 2 {
		t.Fatalf("expected fail-fast to stop after second item, got count=%d", result.Count)
	}
	if !result.Success {
		t.Fatalf("expected batch envelope success=true")
	}
	if result.SuccessCount != 1 || result.FailureCount != 1 {
		t.Fatalf("unexpected success/failure counts: %+v", result)
	}
	if result.Results[1].Success {
		t.Fatalf("expected second result to be failure, got %+v", result.Results[1])
	}
}

func TestFetchBatchPreservesDuplicateURLs(t *testing.T) {
	previousFactory := byparrClientFactory
	byparrClientFactory = func(cfg config.Config) byparrFetcher {
		okResponse := byparr.Response{Status: "ok"}
		okResponse.Solution.Response = `<html><body><article><p>ok</p></article></body></html>`
		return fakeByparrClient{
			responses: map[string]byparr.Response{
				"https://example.com/dup": okResponse,
			},
		}
	}
	t.Cleanup(func() { byparrClientFactory = previousFactory })

	cfg := config.Config{LogLevel: "ERROR", MCPTimeout: 5, MaxConcurrentRequests: 5}
	resultRaw, err := Fetch(cfg, map[string]interface{}{
		"urls": []interface{}{
			"https://example.com/dup",
			"https://example.com/dup",
		},
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	result := resultRaw.(BatchFetchResponse)
	if !result.Success {
		t.Fatalf("expected batch envelope success=true")
	}
	if result.Count != 2 || len(result.Results) != 2 {
		t.Fatalf("expected duplicate URLs to preserve cardinality, got %+v", result)
	}
	if result.SuccessCount != 2 || result.FailureCount != 0 {
		t.Fatalf("unexpected success/failure counts: %+v", result)
	}
	if result.Results[0].URL != "https://example.com/dup" || result.Results[1].URL != "https://example.com/dup" {
		t.Fatalf("expected duplicate results to preserve order, got %+v", result.Results)
	}
}
