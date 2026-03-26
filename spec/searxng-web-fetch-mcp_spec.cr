require "./spec_helper"

describe SearxngWebFetchMcp do
  it "has a version" do
    SearxngWebFetchMcp::VERSION.should eq("0.2.1")
  end

  it "should log correctly" do
    # Should not raise
    SearxngWebFetchMcp.log("INFO", "test message")
  end
end
