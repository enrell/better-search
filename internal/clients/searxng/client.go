package searxng

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) Search(ctx context.Context, query string, numResults int, language string) (SearchResponse, error) {
	params := url.Values{}
	params.Add("q", query)
	params.Add("format", "json")
	params.Add("lang", language)
	params.Add("engines", "general")
	params.Add("categories", "general")
	params.Add("safesearch", "0")
	params.Add("num_results", fmt.Sprintf("%d", numResults))

	searchURL := c.baseURL + "/search?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchResponse{}, fmt.Errorf("searxng returned status %d: %s", resp.StatusCode, string(body))
	}

	var parsed SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return SearchResponse{}, fmt.Errorf("parse response: %w", err)
	}

	return parsed, nil
}
