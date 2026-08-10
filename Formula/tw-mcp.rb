class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.27.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.3/tw-mcp_1.27.3_darwin_arm64.tar.gz"
      sha256 "994a8a5571f9bdfee3c4dbfa0b998625899e8b23a31810bb6b2047ea6b914e4b"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.3/tw-mcp_1.27.3_darwin_amd64.tar.gz"
      sha256 "1b898551a3cd2444db15e29bb33eedcab6d39d95db24718c8e193fe0036d1053"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
