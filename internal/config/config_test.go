package config

import "testing"

func TestLoadValidatesURLs(t *testing.T) {
	t.Setenv("SEARXNG_URL", "://bad-url")
	t.Setenv("BYPARR_URL", "http://localhost:8191")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid searxng url error")
	}
}

func TestLoadUsesEnvironmentValues(t *testing.T) {
	t.Setenv("SEARXNG_URL", "http://localhost:9000")
	t.Setenv("BYPARR_URL", "https://byparr.local")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("MCP_TIMEOUT", "45")
	t.Setenv("MAX_CONCURRENT_REQUESTS", "12")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.SearxngURL != "http://localhost:9000" {
		t.Fatalf("unexpected searxng url: %s", cfg.SearxngURL)
	}
	if cfg.ByparrURL != "https://byparr.local" {
		t.Fatalf("unexpected byparr url: %s", cfg.ByparrURL)
	}
	if cfg.LogLevel != "DEBUG" {
		t.Fatalf("unexpected log level: %s", cfg.LogLevel)
	}
	if cfg.MCPTimeout != 45 {
		t.Fatalf("unexpected timeout: %d", cfg.MCPTimeout)
	}
	if cfg.MaxConcurrentRequests != 12 {
		t.Fatalf("unexpected max concurrent requests: %d", cfg.MaxConcurrentRequests)
	}
}
