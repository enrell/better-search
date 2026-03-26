require "mcp"
require "json"
require "http/client"
require "connect-proxy"

class SearxngWebSearch < MCP::AbstractTool
  @@tool_name = "searxng_web_search"
  @@tool_description = "Search the web using a local SearXNG instance. Returns search results with title, url, and snippet."
  @@tool_input_schema = {
    "type"       => "object",
    "properties" => {
      "query" => {
        "type"        => "string",
        "description" => "The search query",
      },
      "num_results" => {
        "type"        => "number",
        "description" => "Number of results to return (default: 10)",
        "default"     => 10,
      },
      "language" => {
        "type"        => "string",
        "description" => "Search language (default: en)",
        "default"     => "en",
      },
    },
    "required" => ["query"],
  }.to_json

  def invoke(params : Hash(String, JSON::Any), env : HTTP::Server::Context? = nil)
    query = params["query"].as_s
    num_results = params["num_results"]?.try(&.as_i) || 10
    language = params["language"]?.try(&.as_s) || "en"

    res = search_results(query, num_results, language)

    Hash(String, JSON::Any).new.tap do |_hash|
      _hash["content"] = JSON::Any.new([
        JSON::Any.new({
          "type" => JSON::Any.new("text"),
          "text" => JSON::Any.new(res.to_json),
        } of String => JSON::Any),
      ])
      _hash["isError"] = JSON::Any.new(res["success"] == false)
    end
  end

  private def search_results(query : String, num_results : Int, language : String)
    uri = URI.parse("#{SEARXNG_URL}/search")
    client = create_proxy_client

    query_params = HTTP::Params.build do |params|
      params.add("q", query)
      params.add("format", "json")
      params.add("lang", language)
      params.add("engines", "general")
      params.add("categories", "general")
      params.add("safesearch", "0")
      params.add("num_results", num_results.to_s)
    end

    request = HTTP::Request.new("GET", "#{uri.path}?#{query_params}")
    response = client.exec(request)

    if response.status_code != 200
      return {
        "success" => false,
        "error"   => "SearXNG returned status #{response.status_code}",
        "results" => [] of Hash(String, String),
      }
    end

    results = parse_search_results(response.body)
    {
      "success" => true,
      "query"   => query,
      "results" => results,
    }
  rescue ex : Exception
    {
      "success" => false,
      "error"   => ex.message || "Unknown error",
      "results" => [] of Hash(String, String),
    }
  end

  private def create_proxy_client
    # Connect directly to our SearXNG instance without sending the traffic through Byparr proxy
    uri = URI.parse(SEARXNG_URL)
    HTTP::Client.new(uri)
  end

  private def parse_search_results(body : String)
    json = JSON.parse(body)
    results = [] of Hash(String, String)

    if results_obj = json["results"]?
      results_obj.as_a.each do |result|
        result_hash = {} of String => String
        result_hash["title"] = result["title"]?.try(&.as_s) || ""
        result_hash["url"] = result["url"]?.try(&.as_s) || ""
        result_hash["snippet"] = result["content"]?.try(&.as_s) || ""
        result_hash["engine"] = result["engine"]?.try(&.as_s) || ""
        results << result_hash
      end
    end

    results
  end
end
