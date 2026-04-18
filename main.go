package main

import (
	"fmt"
	"os"

	"github.com/enrell/better-search/internal/config"
	"github.com/enrell/better-search/internal/mcp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(1)
	}

	mcp.Run(cfg)
}
