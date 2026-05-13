class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.19.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.19.0/tw-mcp_1.19.0_darwin_arm64.tar.gz"
      sha256 "41c1296394530bdc0dfa3f7af821c7e90b8879e97ae7bd4abbdd2a783c2369dd"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.19.0/tw-mcp_1.19.0_darwin_amd64.tar.gz"
      sha256 "73d5a77d0c9a0787734707dad75d076a12ad78247787c67d4bdfeb4c8ff548fa"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
