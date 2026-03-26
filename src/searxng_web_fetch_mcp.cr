require "mcp"
require "log"
require "./tools/*"
require "./extraction/*"
require "./utils/*"

module SearxngWebFetchMcp
  VERSION = "0.1.2"

  LOG_LEVEL = ENV.fetch("LOG_LEVEL", "INFO").upcase

  def self.log(level, message)
    STDERR.puts "[#{level}] #{message}" if should_log?(level)
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
# Start stdio server (unless we are running specs)
unless PROGRAM_NAME.includes?("spec")
  SearxngWebFetchMcp.log("INFO", "Starting MCP server v#{SearxngWebFetchMcp::VERSION}")
  SearxngWebFetchMcp.log("INFO", "SEARXNG_URL: #{SEARXNG_URL}")
  SearxngWebFetchMcp.log("INFO", "BYPARR_URL: #{BYPARR_URL}")
  MCP::StdioHandler.start_server("searxng-web-fetch-mcp")
end
