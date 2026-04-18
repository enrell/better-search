package byparr

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Response struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		URL      string                 `json:"url"`
		Status   int                    `json:"status"`
		Response string                 `json:"response"`
		Cookies  []Cookie               `json:"cookies"`
		Headers  map[string]interface{} `json:"headers"`
	} `json:"solution"`
}

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Error struct {
	Status int
	Body   string
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) Fetch(ctx context.Context, rawURL string, timeoutMillis int) (Response, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"cmd":        "request.get",
		"url":        rawURL,
		"maxTimeout": timeoutMillis,
	})
	if err != nil {
		return Response{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1", bytes.NewReader(requestBody))
	if err != nil {
		return Response{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Response{}, &Error{Status: resp.StatusCode, Body: string(body)}
	}

	var parsed Response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return Response{}, err
	}
	if parsed.Status != "ok" {
		return Response{}, &Error{Body: parsed.Message}
	}

	return parsed, nil
}

func (e *Error) Error() string {
	if e.Status > 0 {
		return "Byparr error: HTTP " + strings.TrimSpace(e.Body)
	}
	return "Byparr error: " + strings.TrimSpace(e.Body)
}
