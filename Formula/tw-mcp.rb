class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.48.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.48.0/tw-mcp_1.48.0_darwin_arm64.tar.gz"
      sha256 "93f9aaee9bce7de678083cc8eaab46f1534f7156b5b5e9ae2c9caa793a0a5b0f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.48.0/tw-mcp_1.48.0_darwin_amd64.tar.gz"
      sha256 "961f1093e36a2b53b286fa7e6056abc1387043cd60d29e9e9d532b0fb752fc88"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
