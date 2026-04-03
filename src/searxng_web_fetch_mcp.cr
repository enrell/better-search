require "mcp"
require "log"
require "./tools/*"
require "./extraction/*"
require "./utils/*"

module SearxngWebFetchMcp
  VERSION = "0.2.6"
  DEFAULT_SEARXNG_URL    = "http://localhost:8080"
  DEFAULT_BYPARR_URL     = "http://localhost:8191"

  LOG_LEVEL               = ENV.fetch("LOG_LEVEL", "INFO").upcase
  MCP_TIMEOUT             = ENV.fetch("MCP_TIMEOUT", "30").to_i
  MAX_CONCURRENT_REQUESTS = ENV.fetch("MAX_CONCURRENT_REQUESTS", "30").to_i

  def self.searxng_url : String
    ENV.fetch("SEARXNG_URL", DEFAULT_SEARXNG_URL)
  end

  def self.byparr_url : String
    ENV.fetch("BYPARR_URL", DEFAULT_BYPARR_URL)
  end

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

unless PROGRAM_NAME.includes?("spec")
  SearxngWebFetchMcp.log("INFO", "Starting MCP server v#{SearxngWebFetchMcp::VERSION}")
  SearxngWebFetchMcp.log("INFO", "SEARXNG_URL: #{SearxngWebFetchMcp.searxng_url}")
  SearxngWebFetchMcp.log("INFO", "BYPARR_URL: #{SearxngWebFetchMcp.byparr_url}")
  SearxngWebFetchMcp.log("INFO", "MAX_CONCURRENT_REQUESTS: #{SearxngWebFetchMcp::MAX_CONCURRENT_REQUESTS}")
  SearxngWebFetchMcp.log("INFO", "MCP_TIMEOUT: #{SearxngWebFetchMcp::MCP_TIMEOUT}s")

  MCP::StdioHandler.start_server("searxng-web-fetch-mcp")
end
