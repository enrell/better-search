require "mcp"
require "json"
require "http/client"
require "connect-proxy"
require "../extraction/trafilatura_extractor"
require "../utils/html_to_markdown"
require "../utils/concurrent_http"

class WebFetch < MCP::AbstractTool
  @@tool_name = "web_fetch"
  @@tool_description = "Fetch one or more web pages and extract their main content as clean Markdown. Supports parallel batch fetching for multiple URLs."
  @@tool_input_schema = {
    "type"       => "object",
    "properties" => {
      "url" => {
        "type"        => "string",
        "description" => "The URL to fetch (use 'urls' for batch fetching)",
      },
      "urls" => {
        "type"        => "array",
        "items"       => {"type" => "string"},
        "description" => "Array of URLs to fetch in parallel (faster than sequential)",
      },
      "include_metadata" => {
        "type"        => "boolean",
        "description" => "Include metadata like title, author, date (default: true)",
        "default"     => true,
      },
    },
    "oneOf" => [
      {"required" => ["url"]},
      {"required" => ["urls"]},
    ],
  }.to_json

  def invoke(params : Hash(String, JSON::Any), env : HTTP::Server::Context? = nil)
    include_metadata = params["include_metadata"]?.try(&.as_bool) || true

    if urls = params["urls"]?.try(&.as_a)
      fetch_batch(urls.map(&.as_s), include_metadata)
    elsif url = params["url"]?
      fetch_and_extract(url.as_s, include_metadata)
    else
      error_hash = Hash(String, JSON::Any).new
      error_hash["success"] = JSON::Any.new(false)
      error_hash["error"] = JSON::Any.new("Either 'url' or 'urls' parameter is required")
      format_mcp_response(error_hash, true)
    end
  end

  private def format_mcp_response(data : Hash(String, JSON::Any), is_error : Bool)
    Hash(String, JSON::Any).new.tap do |_hash|
      _hash["content"] = JSON::Any.new([
        JSON::Any.new({
          "type" => JSON::Any.new("text"),
          "text" => JSON::Any.new(data.to_json),
        } of String => JSON::Any),
      ])
      _hash["isError"] = JSON::Any.new(is_error)
    end
  end

  private def fetch_batch(urls : Array(String), include_metadata : Bool)
    max_concurrent = SearxngWebFetchMcp::MAX_CONCURRENT_REQUESTS

    results = Utils::ConcurrentHTTP.run_parallel(max_concurrent, urls.map do |url|
      -> { fetch_single_result(url, include_metadata) }
    end)

    response = Hash(String, JSON::Any).new
    response["success"] = JSON::Any.new(true)
    response["count"] = JSON::Any.new(results.size)
    response["results"] = JSON::Any.new(results)

    format_mcp_response(response, false)
  rescue ex : Exception
    error_hash = Hash(String, JSON::Any).new
    error_hash["success"] = JSON::Any.new(false)
    error_hash["error"] = JSON::Any.new(ex.message || "Batch fetch failed")
    format_mcp_response(error_hash, true)
  end

  private def fetch_single_result(url : String, include_metadata : Bool) : JSON::Any
    result = fetch_and_extract(url, include_metadata)
    if result["isError"]?.try(&.as_bool)
      JSON::Any.new(Hash(String, JSON::Any).new.tap do |_hash|
        _hash["success"] = JSON::Any.new(false)
        _hash["url"] = JSON::Any.new(url)
        _hash["error"] = JSON::Any.new(result.dig("content", 0, "text").as_s)
      end)
    else
      JSON::Any.new(Hash(String, JSON::Any).new.tap do |_hash|
        _hash["success"] = JSON::Any.new(true)
        _hash["url"] = JSON::Any.new(url)
        _hash["text"] = JSON::Any.new(result.dig("content", 0, "text").as_s)
      end)
    end
  end

  private def fetch_and_extract(url : String, include_metadata : Bool)
    html = fetch_html(url)
    if html.empty?
      res_err = {
        "success" => false,
        "error"   => "Failed to fetch URL",
        "url"     => url,
      }
      return Hash(String, JSON::Any).new.tap do |_hash|
        _hash["content"] = JSON::Any.new([
          JSON::Any.new({
            "type" => JSON::Any.new("text"),
            "text" => JSON::Any.new(res_err.to_json),
          } of String => JSON::Any),
        ])
        _hash["isError"] = JSON::Any.new(true)
      end
    end

    extractor = Extraction::TrafilaturaExtractor.new
    result = extractor.extract(html)

    markdown = Utils::HtmlToMarkdown.convert(result.text)

    res = Hash(String, JSON::Any).new
    res["success"] = JSON::Any.new(true)
    res["url"] = JSON::Any.new(url)
    res["text"] = JSON::Any.new(markdown)

    if include_metadata
      metadata = Hash(String, JSON::Any).new
      metadata["title"] = JSON::Any.new(result.title)
      metadata["author"] = JSON::Any.new(result.author)
      metadata["date"] = JSON::Any.new(result.date)
      metadata["language"] = JSON::Any.new(result.language)
      metadata["url"] = JSON::Any.new(result.url)
      res["metadata"] = JSON::Any.new(metadata)
    end

    response = Hash(String, JSON::Any).new
    response["content"] = JSON::Any.new([
      JSON::Any.new({
        "type" => JSON::Any.new("text"),
        "text" => JSON::Any.new(res.to_json),
      } of String => JSON::Any),
    ])
    response["isError"] = JSON::Any.new(false)
    response
  rescue ex : Exception
    res = {
      "success" => false,
      "error"   => ex.message || "Unknown error",
      "url"     => url,
    }

    Hash(String, JSON::Any).new.tap do |_hash|
      _hash["content"] = JSON::Any.new([
        JSON::Any.new({
          "type" => JSON::Any.new("text"),
          "text" => JSON::Any.new(res.to_json),
        } of String => JSON::Any),
      ])
      _hash["isError"] = JSON::Any.new(true)
    end
  end

  private def fetch_html(url : String) : String
    target_uri = URI.parse(url)

    begin
      # Create client - use TLS for HTTPS, disable verification for some environments
      client = if target_uri.scheme == "https"
                 tls = OpenSSL::SSL::Context::Client.new
                 tls.verify_mode = OpenSSL::SSL::VerifyMode::NONE
                 HTTP::Client.new(target_uri, tls: tls)
               else
                 HTTP::Client.new(target_uri)
               end

      client.connect_timeout = SearxngWebFetchMcp::MCP_TIMEOUT.seconds
      client.read_timeout = SearxngWebFetchMcp::MCP_TIMEOUT.seconds

      request = HTTP::Request.new("GET", url)
      request.headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
      request.headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8"
      request.headers["Accept-Language"] = "en-US,en;q=0.9"

      response = client.exec(request)

      if response.status_code >= 300 && response.status_code < 400
        location = response.headers["Location"]?
        if location
          # Prevent infinite redirects
          if location == url
            raise "Redirect loop detected"
          end
          # Build absolute URL if relative
          redirect_url = location.starts_with?("http") ? location : URI.parse(url).resolve(location).to_s
          return fetch_html(redirect_url)
        end
      end

      if response.status_code != 200
        raise "HTTP error: #{response.status_code}"
      end

      response.body
    rescue ex : IO::TimeoutError
      SearxngWebFetchMcp.log("ERROR", "Timeout fetching #{url}: #{ex.message}")
      raise Exception.new("Request timeout after #{SearxngWebFetchMcp::MCP_TIMEOUT} seconds")
    rescue ex : Exception
      SearxngWebFetchMcp.log("ERROR", "Failed to fetch #{url}: #{ex.message}")
      raise ex
    end
  end
end
