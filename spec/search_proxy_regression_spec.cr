require "./spec_helper"
require "http/server"
require "socket"
require "json"

private def with_env(name : String, value : String?)
  previous = ENV[name]?

  if value
    ENV[name] = value
  else
    ENV.delete(name)
  end

  yield
ensure
  if previous
    ENV[name] = previous
  else
    ENV.delete(name)
  end
end

private def with_test_env(searxng_url : String, byparr_url : String, &)
  with_env("SEARXNG_URL", searxng_url) do
    with_env("BYPARR_URL", byparr_url) do
      with_env("http_proxy", nil) do
        with_env("https_proxy", nil) do
          with_env("HTTP_PROXY", nil) do
            with_env("HTTPS_PROXY", nil) do
              yield
            end
          end
        end
      end
    end
  end
end

private class TestHttpServer
  getter url : String

  def initialize(&handler : HTTP::Server::Context ->)
    @server = HTTP::Server.new do |context|
      handler.call(context)
    end
    address = @server.bind_unused_port("127.0.0.1")
    @url = "http://127.0.0.1:#{address.port}"

    spawn do
      @server.listen
    rescue IO::Error
    end
  end

  def close
    @server.close
  rescue IO::Error
  end
end

private class TestProxyServer
  getter url : String

  def initialize(@blocked_ports : Array(Int32))
    @connections = Atomic(Int32).new(0)
    @server = TCPServer.new("127.0.0.1", 0)
    address = @server.local_address.as(Socket::IPAddress)
    @url = "http://127.0.0.1:#{address.port}"

    spawn do
      accept_loop
    end
  end

  def connection_count : Int32
    @connections.get
  end

  def close
    @server.close
  rescue IO::Error
  end

  private def accept_loop
    loop do
      socket = @server.accept?
      break unless socket

      spawn do
        handle(socket)
      end
    end
  rescue IO::Error
  end

  private def handle(socket : TCPSocket)
    request = socket.gets("\r\n\r\n")
    return socket.close unless request

    details = request.split(" ", 3)[1].split(":")
    host = details[0]
    port = details[1].to_i

    if @blocked_ports.includes?(port)
      socket << "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n"
      socket.close
      return
    end

    client = TCPSocket.new(host, port)
    socket << "HTTP/1.1 200 OK\r\n\r\n"
    @connections.add(1)

    spawn do
      begin
        raw_data = Bytes.new(2048)
        while !client.closed?
          bytes_read = client.read(raw_data)
          break if bytes_read.zero?
          socket.write raw_data[0, bytes_read].dup
        end
      rescue IO::Error
      ensure
        socket.close
      end
    end

    begin
      out_data = Bytes.new(2048)
      while !socket.closed?
        bytes_read = socket.read(out_data)
        break if bytes_read.zero?
        client.write out_data[0, bytes_read].dup
      end
    rescue IO::Error
    ensure
      client.close
      socket.close
    end
  rescue IO::Error
    socket.close
  end
end

describe "proxy regression" do
  it "keeps SearXNG search off the fetch proxy path" do
    search_server = TestHttpServer.new do |context|
      if context.request.path == "/search"
        context.response.content_type = "application/json"
        context.response.print({
          "results" => [
            {
              "title" => "Proxy-safe result",
              "url" => "https://example.com/article",
              "content" => "search snippet",
              "engine" => "stub",
            },
          ],
        }.to_json)
      else
        context.response.status_code = 404
      end
    end

    fetch_server = TestHttpServer.new do |context|
      context.response.content_type = "text/html"
      context.response.print("<html><body><article><h1>Proxy-safe article</h1><p>Body text for extraction.</p></article></body></html>")
    end

    search_port = URI.parse(search_server.url).port.not_nil!
    proxy_server = TestProxyServer.new([search_port])

    with_test_env(search_server.url, proxy_server.url) do
      fetch_tool = WebFetch.new
      fetch_result = fetch_tool.invoke({"url" => JSON::Any.new("#{fetch_server.url}/article")} of String => JSON::Any, nil)
      fetch_payload = JSON.parse(fetch_result["content"].as_a[0]["text"].as_s)

      fetch_payload["success"].as_bool.should be_true
      proxy_server.connection_count.should eq(1)

      search_tool = SearxngWebSearch.new
      search_result = search_tool.invoke({"query" => JSON::Any.new("proxy regression")} of String => JSON::Any, nil)
      search_payload = JSON.parse(search_result["content"].as_a[0]["text"].as_s)

      search_payload["success"].as_bool.should be_true
      search_payload["results"].as_a.size.should eq(1)
      proxy_server.connection_count.should eq(1)
    end
  ensure
    proxy_server.try(&.close)
    fetch_server.try(&.close)
    search_server.try(&.close)
  end
end