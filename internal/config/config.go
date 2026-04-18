package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSearxngURL            = "http://localhost:8080"
	defaultByparrURL             = "http://localhost:8191"
	defaultLogLevel              = "INFO"
	defaultMCPTimeout            = 30
	defaultMaxConcurrentRequests = 30
)

var logLevels = map[string]int{
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
}

type Config struct {
	SearxngURL            string
	ByparrURL             string
	LogLevel              string
	MCPTimeout            int
	MaxConcurrentRequests int
}

func Load() (Config, error) {
	cfg := Config{
		SearxngURL:            envOrDefault("SEARXNG_URL", defaultSearxngURL),
		ByparrURL:             envOrDefault("BYPARR_URL", defaultByparrURL),
		LogLevel:              strings.ToUpper(envOrDefault("LOG_LEVEL", defaultLogLevel)),
		MCPTimeout:            envIntOrDefault("MCP_TIMEOUT", defaultMCPTimeout),
		MaxConcurrentRequests: envIntOrDefault("MAX_CONCURRENT_REQUESTS", defaultMaxConcurrentRequests),
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if err := validateBaseURL("SEARXNG_URL", c.SearxngURL); err != nil {
		return err
	}
	if err := validateBaseURL("BYPARR_URL", c.ByparrURL); err != nil {
		return err
	}
	if _, ok := logLevels[c.LogLevel]; !ok {
		return fmt.Errorf("LOG_LEVEL must be one of DEBUG, INFO, WARN, ERROR")
	}
	if c.MCPTimeout <= 0 {
		return fmt.Errorf("MCP_TIMEOUT must be greater than zero")
	}
	if c.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("MAX_CONCURRENT_REQUESTS must be greater than zero")
	}
	return nil
}

func (c Config) ShouldLog(level string) bool {
	current, ok1 := logLevels[c.LogLevel]
	msg, ok2 := logLevels[strings.ToUpper(level)]
	return ok1 && ok2 && current <= msg
}

func (c Config) LogMsg(level, message string) {
	c.LogAttrs(level, message, nil)
}

func (c Config) LogAttrs(level, message string, attrs map[string]interface{}) {
	if !c.ShouldLog(level) {
		return
	}

	var builder strings.Builder
	builder.WriteString("ts=")
	builder.WriteString(time.Now().UTC().Format(time.RFC3339))
	builder.WriteString(" level=")
	builder.WriteString(strings.ToUpper(level))
	builder.WriteString(" msg=")
	builder.WriteString(strconv.Quote(message))

	for key, value := range attrs {
		builder.WriteByte(' ')
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(fmt.Sprint(value))
	}

	builder.WriteByte('\n')
	_, _ = os.Stderr.WriteString(builder.String())
}

func envOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			return v
		}
	}
	return defaultVal
}

func validateBaseURL(key, raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s must be a valid URL: %w", key, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", key)
	}
	if parsed.Host == "" {
		return fmt.Errorf("%s must include a host", key)
	}
	return nil
}
