require "mcp"
require "log"
require "./tools/*"
require "./extraction/*"
require "./utils/*"

module SearxngWebFetchMcp
  VERSION = "0.1.0"

  LOG_LEVEL = ENV.fetch("LOG_LEVEL", "INFO").upcase

  def self.log(level, message)
    puts "[#{level}] #{message}" if should_log?(level)
  end

  private def self.should_log?(level)
    levels = {"DEBUG" => 0, "INFO" => 1, "WARN" => 2, "ERROR" => 3}
    current = levels[LOG_LEVEL]?
    msg = levels[level]?
    current && msg && current <= msg
  end
end

# Default configuration from environment
SEARXNG_URL = ENV.fetch("SEARXNG_URL", "http://localhost:8080")
BYPARR_URL  = ENV.fetch("BYPARR_URL", "http://localhost:8191")

# Tools auto-register via MCP::AbstractTool's inherited macro
# Start stdio server
MCP::StdioHandler.start_server("searxng-web-fetch-mcp")
