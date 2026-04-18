package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/enrell/better-search-mcp/internal/config"
	"github.com/enrell/better-search-mcp/internal/extractor"
)

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Engine  string `json:"engine"`
}

type SearchResponse struct {
	Success bool           `json:"success"`
	Query   string         `json:"query,omitempty"`
	Error   string         `json:"error,omitempty"`
	Results []SearchResult `json:"results"`
}

type FetchResult struct {
	Success  bool      `json:"success"`
	URL      string    `json:"url"`
	Text     string    `json:"text,omitempty"`
	Error    string    `json:"error,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

type Metadata struct {
	Title    string `json:"title"`
	Author   string `json:"author"`
	Date     string `json:"date"`
	Language string `json:"language"`
	URL      string `json:"url"`
}

type BatchFetchResponse struct {
	Success bool          `json:"success"`
	Count   int           `json:"count"`
	Results []FetchResult `json:"results"`
}

func Search(arguments map[string]interface{}) (interface{}, error) {
	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("'query' parameter is required")
	}

	numResults := 10
	if nr, ok := arguments["num_results"].(float64); ok {
		numResults = int(nr)
	}

	language := "en"
	if lang, ok := arguments["language"].(string); ok {
		language = lang
	}

	return searchSearXNG(query, numResults, language)
}

func Fetch(arguments map[string]interface{}) (interface{}, error) {
	includeMetadata := true
	if im, ok := arguments["include_metadata"].(bool); ok {
		includeMetadata = im
	}

	if urlsRaw, ok := arguments["urls"].([]interface{}); ok {
		urls := make([]string, 0, len(urlsRaw))
		for _, u := range urlsRaw {
			if s, ok := u.(string); ok {
				urls = append(urls, s)
			}
		}
		if len(urls) > 0 {
			return fetchBatch(urls, includeMetadata), nil
		}
	}

	if urlStr, ok := arguments["url"].(string); ok && urlStr != "" {
		return fetchSingleResult(urlStr, includeMetadata), nil
	}

	return nil, fmt.Errorf("Either 'url' or 'urls' parameter is required")
}

func searchSearXNG(query string, numResults int, language string) (SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.MCPTimeout)*time.Second)
	defer cancel()

	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")
	params.Add("lang", language)
	params.Add("engines", "general")
	params.Add("categories", "general")
	params.Add("safesearch", "0")
	params.Add("num_results", fmt.Sprintf("%d", numResults))

	searchURL := config.SearxngURL
	if searchURL[len(searchURL)-1] == '/' {
		searchURL += "search?" + params.Encode()
	} else {
		searchURL += "/search?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return SearchResponse{Success: false, Error: fmt.Sprintf("Failed to create request: %v", err)}, nil
	}

	client := &http.Client{
		Timeout: time.Duration(config.MCPTimeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return SearchResponse{Success: false, Error: fmt.Sprintf("Request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchResponse{
			Success: false,
			Error:   fmt.Sprintf("SearXNG returned status %d: %s", resp.StatusCode, string(body)),
			Results: []SearchResult{},
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResponse{Success: false, Error: fmt.Sprintf("Failed to read response: %v", err)}, nil
	}

	var searxngResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
			Engine  string `json:"engine"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searxngResp); err != nil {
		return SearchResponse{Success: false, Error: fmt.Sprintf("Failed to parse response: %v", err)}, nil
	}

	results := make([]SearchResult, 0, len(searxngResp.Results))
	for _, r := range searxngResp.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Engine:  r.Engine,
		})
	}

	return SearchResponse{
		Success: true,
		Query:   query,
		Results: results,
	}, nil
}

func fetchBatch(urls []string, includeMetadata bool) BatchFetchResponse {
	maxConcurrent := config.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	semaphore := make(chan struct{}, maxConcurrent)
	results := make([]FetchResult, len(urls))
	var wg sync.WaitGroup

	for i, u := range urls {
		wg.Add(1)
		go func(idx int, rawURL string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			results[idx] = fetchSingleResult(rawURL, includeMetadata)
		}(i, u)
	}

	wg.Wait()

	return BatchFetchResponse{
		Success: true,
		Count:   len(results),
		Results: results,
	}
}

func fetchSingleResult(rawURL string, includeMetadata bool) FetchResult {
	html, err := fetchViaByparr(rawURL)
	if err != nil {
		return FetchResult{
			Success: false,
			URL:     rawURL,
			Error:   err.Error(),
		}
	}

	if html == "" {
		return FetchResult{
			Success: false,
			URL:     rawURL,
			Error:   "Failed to fetch URL: empty response",
		}
	}

	extractionResult := extractor.Extract(html)
	markdown := extractor.HTMLToMarkdown(extractionResult.Text)

	result := FetchResult{
		Success: true,
		URL:     rawURL,
		Text:    markdown,
	}

	if includeMetadata {
		result.Metadata = &Metadata{
			Title:    extractionResult.Title,
			Author:   extractionResult.Author,
			Date:     extractionResult.Date,
			Language: extractionResult.Language,
			URL:      extractionResult.URL,
		}
	}

	return result
}

func fetchViaByparr(rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.MCPTimeout)*time.Second)
	defer cancel()

	byparrReq, err := json.Marshal(map[string]interface{}{
		"cmd":        "request.get",
		"url":        rawURL,
		"maxTimeout": config.MCPTimeout * 1000,
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.ByparrURL+"/v1", bytes.NewReader(byparrReq))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := &http.Client{
		Timeout: time.Duration(config.MCPTimeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
			return "", context.DeadlineExceeded
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", &byparrError{status: resp.StatusCode, body: string(body)}
	}

	var byparrResp struct {
		Status   string `json:"status"`
		Message  string `json:"message"`
		Solution struct {
			URL      string `json:"url"`
			Status   int    `json:"status"`
			Response string `json:"response"`
			Cookies  []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"cookies"`
			Headers map[string]interface{} `json:"headers"`
		} `json:"solution"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&byparrResp); err != nil {
		return "", err
	}

	if byparrResp.Status != "ok" {
		return "", &byparrError{status: 0, body: byparrResp.Message}
	}

	return byparrResp.Solution.Response, nil
}

type byparrError struct {
	status int
	body   string
}

func (e *byparrError) Error() string {
	if e.status > 0 {
		return "Byparr error: HTTP " + strings.TrimSpace(e.body)
	}
	return "Byparr error: " + strings.TrimSpace(e.body)
}
