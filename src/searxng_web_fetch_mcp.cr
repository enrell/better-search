require "mcp"
require "log"
require "./tools/*"
require "./extraction/*"
require "./utils/*"

module SearxngWebFetchMcp
  VERSION = "0.2.1"

  LOG_LEVEL               = ENV.fetch("LOG_LEVEL", "INFO").upcase
  MCP_TIMEOUT             = ENV.fetch("MCP_TIMEOUT", "30").to_i
  MAX_CONCURRENT_REQUESTS = ENV.fetch("MAX_CONCURRENT_REQUESTS", "30").to_i

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

SEARXNG_URL = ENV.fetch("SEARXNG_URL", "http://localhost:8080")
BYPARR_URL  = ENV.fetch("BYPARR_URL", "http://localhost:8191")

unless PROGRAM_NAME.includes?("spec")
  SearxngWebFetchMcp.log("INFO", "Starting MCP server v#{SearxngWebFetchMcp::VERSION}")
  SearxngWebFetchMcp.log("INFO", "SEARXNG_URL: #{SEARXNG_URL}")
  SearxngWebFetchMcp.log("INFO", "BYPARR_URL: #{BYPARR_URL}")
  SearxngWebFetchMcp.log("INFO", "MAX_CONCURRENT_REQUESTS: #{SearxngWebFetchMcp::MAX_CONCURRENT_REQUESTS}")
  SearxngWebFetchMcp.log("INFO", "MCP_TIMEOUT: #{SearxngWebFetchMcp::MCP_TIMEOUT}s")

  MCP::StdioHandler.start_server("searxng-web-fetch-mcp")
end
