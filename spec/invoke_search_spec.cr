require "./spec_helper"
require "json"

describe SearxngWebSearch do
  it "fetches responses from SearXNG" do
    tool = SearxngWebSearch.new
    
    # We pass the arguments exactly as the client would
    args = {
      "query" => JSON::Any.new("test query")
    } of String => JSON::Any

    result = tool.invoke(args, nil)
    
    # Let's print the result to see the shape and content
    puts "\n--- Tool Result ---"
    puts result.to_json
    puts "-------------------\n"
  end
end
