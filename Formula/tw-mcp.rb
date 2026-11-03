class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.60.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.3/tw-mcp_1.60.3_darwin_arm64.tar.gz"
      sha256 "2dda9b74b4165a642d0e47de9105aa2d15bafaf0f8a30c132b68200bf9ae6761"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.3/tw-mcp_1.60.3_darwin_amd64.tar.gz"
      sha256 "e478d6889ceec6a81f98e46a0ed0c5829fe089d06b4e13d89cc20675e9b9f7d2"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
