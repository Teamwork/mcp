class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.22.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.22.0/tw-mcp_1.22.0_darwin_arm64.tar.gz"
      sha256 "427e99578c48709596318043bb44c09c9bb8522e5dd7317e243441d8c8dbf620"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.22.0/tw-mcp_1.22.0_darwin_amd64.tar.gz"
      sha256 "3460e57059f4b00a28883dba63ba34cec1da03d46346fe76d28b895c9a621712"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
