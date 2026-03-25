require "mcp"
require "json"
require "http/client"
require "connect-proxy"
require "../extraction/trafilatura_extractor"
require "../utils/html_to_markdown"

class WebFetch < MCP::AbstractTool
  @@tool_name = "web_fetch"
  @@tool_description = "Fetch a web page and extract its main content as clean Markdown. Uses Byparr proxy for fetching and trafilatura-style extraction."
  @@tool_input_schema = {
    "type"       => "object",
    "properties" => {
      "url" => {
        "type"        => "string",
        "description" => "The URL to fetch",
      },
      "include_metadata" => {
        "type"        => "boolean",
        "description" => "Include metadata like title, author, date (default: true)",
        "default"     => true,
      },
    },
    "required" => ["url"],
  }.to_json

  def invoke(params : Hash(String, JSON::Any), env : HTTP::Server::Context? = nil)
    url = params["url"].as_s
    include_metadata = params["include_metadata"]?.try(&.as_bool) || true

    fetch_and_extract(url, include_metadata)
  end

  private def fetch_and_extract(url : String, include_metadata : Bool)
    begin
      html = fetch_html(url)
      return {
        "success" => false,
        "error"   => "Failed to fetch URL",
        "url"     => url,
      } if html.empty?

      extractor = Extraction::TrafilaturaExtractor.new
      result = extractor.extract(html)

      markdown = Utils::HtmlToMarkdown.convert(result.text)

      response = Hash(String, JSON::Any).new
      response["success"] = JSON::Any.new(true)
      response["url"] = JSON::Any.new(url)
      response["content"] = JSON::Any.new(markdown)

      if include_metadata
        metadata = Hash(String, JSON::Any).new
        metadata["title"] = JSON::Any.new(result.title)
        metadata["author"] = JSON::Any.new(result.author)
        metadata["date"] = JSON::Any.new(result.date)
        metadata["language"] = JSON::Any.new(result.language)
        metadata["url"] = JSON::Any.new(result.url)
        response["metadata"] = JSON::Any.new(metadata)
      end

      response
    rescue ex : Exception
      {
        "success" => false,
        "error"   => ex.message || "Unknown error",
        "url"     => url,
      }
    end
  end

  private def fetch_html(url : String) : String
    target_uri = URI.parse(url)
    client = ConnectProxy::HTTPClient.new(target_uri)
    client.connect_timeout = 30.seconds
    client.read_timeout = 30.seconds

    request = HTTP::Request.new("GET", url)
    request.headers["User-Agent"] = "Mozilla/5.0 (compatible; MCP-Bot/1.0)"
    request.headers["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

    response = client.exec(request)

    if response.status_code >= 300 && response.status_code < 400
      location = response.headers["Location"]?
      if location
        return fetch_html(location)
      end
    end

    if response.status_code != 200
      raise "HTTP error: #{response.status_code}"
    end

    response.body
  end
end
