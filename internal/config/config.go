package config

import (
	"os"
	"strconv"
)

var (
	SearxngURL            = envOrDefault("SEARXNG_URL", "http://localhost:8080")
	ByparrURL             = envOrDefault("BYPARR_URL", "http://localhost:8191")
	LogLevel              = envOrDefault("LOG_LEVEL", "INFO")
	MCPTimeout            = envIntOrDefault("MCP_TIMEOUT", 30)
	MaxConcurrentRequests = envIntOrDefault("MAX_CONCURRENT_REQUESTS", 30)
)

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

func ShouldLog(level string) bool {
	levels := map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}
	current, ok1 := levels[LogLevel]
	msg, ok2 := levels[level]
	return ok1 && ok2 && current <= msg
}

func LogMsg(level, message string) {
	if ShouldLog(level) {
		os.Stderr.WriteString("[" + level + "] " + message + "\n")
	}
}
