class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.25.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.4/tw-mcp_1.25.4_darwin_arm64.tar.gz"
      sha256 "5b64a1380bf88c0a8769caecbe0e97eda6f32c638dbfefb8346e45ca83828baa"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.4/tw-mcp_1.25.4_darwin_amd64.tar.gz"
      sha256 "58a5d4ada591b21106ba0e8ea4f4c34c9de5ba8f838581e1303377b389ac2d95"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
