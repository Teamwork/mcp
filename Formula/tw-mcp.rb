class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.18.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.3/tw-mcp_1.18.3_darwin_arm64.tar.gz"
      sha256 "58ed8762c92d9eec6ae83247ca416373f624c8a2e5901e5aa7796f872631bc9e"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.3/tw-mcp_1.18.3_darwin_amd64.tar.gz"
      sha256 "baf3eac5139e2c7cd28de77a283b5e37d470baedd015d7c0a414060f8470c828"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
