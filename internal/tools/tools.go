package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/enrell/better-search-mcp/internal/clients/byparr"
	"github.com/enrell/better-search-mcp/internal/clients/searxng"
	"github.com/enrell/better-search-mcp/internal/config"
	"github.com/enrell/better-search-mcp/internal/extractor"
)

const (
	defaultNumResults = 10
	maxNumResults     = 50
	maxBatchURLs      = 25
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
	Success   bool      `json:"success"`
	URL       string    `json:"url"`
	Text      string    `json:"text,omitempty"`
	RawHTML   string    `json:"raw_html,omitempty"`
	Error     string    `json:"error,omitempty"`
	Metadata  *Metadata `json:"metadata,omitempty"`
	Truncated bool      `json:"truncated,omitempty"`
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

type FetchOptions struct {
	IncludeMetadata    bool
	TimeoutSeconds     int
	MaxContentChars    int
	PreserveLinks      bool
	RawHTML            bool
	PreferReadableText bool
	FailFast           bool
}

func Search(cfg config.Config, arguments map[string]interface{}) (interface{}, error) {
	query, ok := arguments["query"].(string)
	query = strings.TrimSpace(query)
	if !ok || query == "" {
		return nil, fmt.Errorf("'query' parameter is required")
	}

	numResults := defaultNumResults
	if nr, ok := arguments["num_results"].(float64); ok {
		numResults = int(nr)
	}
	if numResults < 1 || numResults > maxNumResults {
		return nil, fmt.Errorf("'num_results' must be between 1 and %d", maxNumResults)
	}

	language := "en"
	if lang, ok := arguments["language"].(string); ok {
		language = strings.TrimSpace(lang)
	}
	if language == "" {
		language = "en"
	}

	return searchSearXNG(cfg, query, numResults, language)
}

func Fetch(cfg config.Config, arguments map[string]interface{}) (interface{}, error) {
	options, err := parseFetchOptions(cfg, arguments)
	if err != nil {
		return nil, err
	}

	urlStr, hasURL := arguments["url"].(string)
	urlStr = strings.TrimSpace(urlStr)

	var urls []string
	if urlsRaw, ok := arguments["urls"].([]interface{}); ok {
		normalizedURLs, err := normalizeURLBatch(urlsRaw)
		if err != nil {
			return nil, err
		}
		urls = normalizedURLs
	}

	if hasURL && len(urls) > 0 {
		return nil, fmt.Errorf("provide either 'url' or 'urls', not both")
	}

	if len(urls) > 0 {
		return fetchBatch(cfg, urls, options), nil
	}

	if hasURL {
		if urlStr == "" {
			return nil, fmt.Errorf("'url' cannot be empty")
		}
		normalizedURL, err := normalizeHTTPURL(urlStr)
		if err != nil {
			return nil, fmt.Errorf("'url' is invalid: %w", err)
		}
		return fetchSingleResult(cfg, normalizedURL, options), nil
	}

	if _, ok := arguments["urls"]; ok {
		return nil, fmt.Errorf("'urls' must contain between 1 and %d valid URLs", maxBatchURLs)
	}

	return nil, fmt.Errorf("either 'url' or 'urls' parameter is required")
}

func parseFetchOptions(cfg config.Config, arguments map[string]interface{}) (FetchOptions, error) {
	options := FetchOptions{
		IncludeMetadata:    true,
		TimeoutSeconds:     cfg.MCPTimeout,
		PreserveLinks:      true,
		PreferReadableText: true,
	}

	if im, ok := arguments["include_metadata"].(bool); ok {
		options.IncludeMetadata = im
	}
	if v, ok := arguments["preserve_links"].(bool); ok {
		options.PreserveLinks = v
	}
	if v, ok := arguments["raw_html"].(bool); ok {
		options.RawHTML = v
	}
	if v, ok := arguments["prefer_readable_text"].(bool); ok {
		options.PreferReadableText = v
	}
	if v, ok := arguments["fail_fast"].(bool); ok {
		options.FailFast = v
	}
	if v, ok := arguments["timeout_seconds"].(float64); ok {
		options.TimeoutSeconds = int(v)
	}
	if v, ok := arguments["max_content_chars"].(float64); ok {
		options.MaxContentChars = int(v)
	}

	if options.TimeoutSeconds < 1 || options.TimeoutSeconds > 120 {
		return FetchOptions{}, fmt.Errorf("'timeout_seconds' must be between 1 and 120")
	}
	if options.MaxContentChars < 0 {
		return FetchOptions{}, fmt.Errorf("'max_content_chars' cannot be negative")
	}

	return options, nil
}

func normalizeURLBatch(urlsRaw []interface{}) ([]string, error) {
	if len(urlsRaw) == 0 {
		return nil, fmt.Errorf("'urls' cannot be empty")
	}
	if len(urlsRaw) > maxBatchURLs {
		return nil, fmt.Errorf("'urls' cannot contain more than %d items", maxBatchURLs)
	}

	urls := make([]string, 0, len(urlsRaw))
	for idx, raw := range urlsRaw {
		rawURL, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("'urls[%d]' must be a string", idx)
		}
		normalizedURL, err := normalizeHTTPURL(rawURL)
		if err != nil {
			return nil, fmt.Errorf("'urls[%d]' is invalid: %w", idx, err)
		}
		if slices.Contains(urls, normalizedURL) {
			continue
		}
		urls = append(urls, normalizedURL)
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("'urls' must contain at least one valid URL")
	}

	return urls, nil
}

func normalizeHTTPURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty value")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	return parsed.String(), nil
}

func searchSearXNG(cfg config.Config, query string, numResults int, language string) (SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MCPTimeout)*time.Second)
	defer cancel()
	startedAt := time.Now()

	client := searxng.NewClient(cfg.SearxngURL, newHTTPClient(cfg))
	resp, err := client.Search(ctx, query, numResults, language)
	if err != nil {
		cfg.LogAttrs("WARN", "searxng search failed", map[string]interface{}{
			"query":      query,
			"elapsed_ms": time.Since(startedAt).Milliseconds(),
		})
		return SearchResponse{Success: false, Error: err.Error(), Results: []SearchResult{}}, nil
	}

	results := make([]SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Engine:  r.Engine,
		})
	}

	cfg.LogAttrs("DEBUG", "searxng search succeeded", map[string]interface{}{
		"query":        query,
		"result_count": len(results),
		"elapsed_ms":   time.Since(startedAt).Milliseconds(),
	})

	return SearchResponse{
		Success: true,
		Query:   query,
		Results: results,
	}, nil
}

func fetchBatch(cfg config.Config, urls []string, options FetchOptions) BatchFetchResponse {
	maxConcurrent := cfg.MaxConcurrentRequests
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	if options.FailFast {
		results := make([]FetchResult, 0, len(urls))
		for _, rawURL := range urls {
			result := fetchSingleResult(cfg, rawURL, options)
			results = append(results, result)
			if !result.Success {
				break
			}
		}
		return BatchFetchResponse{
			Success: len(results) == len(urls),
			Count:   len(results),
			Results: results,
		}
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
			results[idx] = fetchSingleResult(cfg, rawURL, options)
		}(i, u)
	}

	wg.Wait()

	allSuccessful := true
	for _, result := range results {
		if !result.Success {
			allSuccessful = false
			break
		}
	}

	return BatchFetchResponse{
		Success: allSuccessful,
		Count:   len(results),
		Results: results,
	}
}

func fetchSingleResult(cfg config.Config, rawURL string, options FetchOptions) FetchResult {
	html, err := fetchViaByparr(withTimeout(cfg, options.TimeoutSeconds), rawURL)
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
	contentSource := extractionResult.ContentHTML
	if !options.PreferReadableText || strings.TrimSpace(contentSource) == "" {
		contentSource = html
	}

	markdown := extractor.HTMLToMarkdown(contentSource)
	if !options.PreserveLinks {
		markdown = stripMarkdownLinks(markdown)
	}

	rawHTML := ""
	if options.RawHTML {
		rawHTML = contentSource
	}

	text, textTruncated := truncateContent(markdown, options.MaxContentChars)
	rawHTML, htmlTruncated := truncateContent(rawHTML, options.MaxContentChars)

	result := FetchResult{
		Success:   true,
		URL:       rawURL,
		Text:      text,
		RawHTML:   rawHTML,
		Truncated: textTruncated || htmlTruncated,
	}

	if options.IncludeMetadata {
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

func fetchViaByparr(cfg config.Config, rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MCPTimeout)*time.Second)
	defer cancel()
	startedAt := time.Now()

	client := byparr.NewClient(cfg.ByparrURL, newHTTPClient(cfg))
	resp, err := client.Fetch(ctx, rawURL, cfg.MCPTimeout*1000)
	if err != nil {
		cfg.LogAttrs("WARN", "byparr fetch failed", map[string]interface{}{
			"url":        rawURL,
			"elapsed_ms": time.Since(startedAt).Milliseconds(),
		})
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
			return "", context.DeadlineExceeded
		}
		return "", err
	}
	cfg.LogAttrs("DEBUG", "byparr fetch succeeded", map[string]interface{}{
		"url":        rawURL,
		"elapsed_ms": time.Since(startedAt).Milliseconds(),
	})
	return resp.Solution.Response, nil
}

func newHTTPClient(cfg config.Config) *http.Client {
	return &http.Client{
		Timeout: time.Duration(cfg.MCPTimeout) * time.Second,
	}
}

func withTimeout(cfg config.Config, timeoutSeconds int) config.Config {
	cfg.MCPTimeout = timeoutSeconds
	return cfg
}

var markdownLinkPattern = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]+)\)`)

func stripMarkdownLinks(markdown string) string {
	return markdownLinkPattern.ReplaceAllStringFunc(markdown, func(match string) string {
		parts := markdownLinkPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}
		if strings.HasPrefix(match, "![") {
			return parts[1]
		}
		return parts[1]
	})
}

func truncateContent(content string, maxChars int) (string, bool) {
	if maxChars <= 0 || len(content) <= maxChars {
		return content, false
	}
	if maxChars <= 3 {
		return content[:maxChars], true
	}
	return content[:maxChars-3] + "...", true
}
