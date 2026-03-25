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
# Monkey patch MCP::StdioHandler.start_server to prevent STDOUT corruption
# Standard MCP requires STDOUT to ONLY contain valid JSON-RPC responses.
class MCP::StdioHandler
  def self.start_server(user_id : String = "stdio_user")
    STDERR.puts "MCP stdio server started. Available tools:"
    STDERR.puts MCP.registered_tools.keys.join(", ")
    STDERR.puts "---"
    STDOUT.flush

    while !STDIN.closed?
      begin
        line = STDIN.gets
        break unless line

        line = line.strip
        next if line.empty?

        # Handle the request
        response = handle_request(line, user_id)

        # Send response
        puts response
        STDOUT.flush
      rescue ex
        error_response = {
          "jsonrpc" => "2.0",
          "error"   => {
            "code"    => -32603,
            "message" => "Internal error: #{ex.message}",
          },
          "id" => nil,
        }
        puts error_response.to_json
        STDOUT.flush
      end
    end
  end
end

# Start stdio server (unless we are running specs)
unless PROGRAM_NAME.includes?("spec")
  MCP::StdioHandler.start_server("searxng-web-fetch-mcp")
end
