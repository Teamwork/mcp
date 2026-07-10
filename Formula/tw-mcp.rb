class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.23.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.3/tw-mcp_1.23.3_darwin_arm64.tar.gz"
      sha256 "92aeed3fd4d6a4c64afbd38eab9d6b84c82e187a9b8f0bf95087b7ca2cee4015"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.3/tw-mcp_1.23.3_darwin_amd64.tar.gz"
      sha256 "a55067a4d3884dcc750de7205cbcd3694386bca2f43ada40e2c3cfea0a4dde0d"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
