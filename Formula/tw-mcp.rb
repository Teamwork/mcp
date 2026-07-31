class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.26.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.1/tw-mcp_1.26.1_darwin_arm64.tar.gz"
      sha256 "a39a87a1ff8ba3677cef6cce7788dea3e75f592d63e36aeffb1dd53168914e72"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.1/tw-mcp_1.26.1_darwin_amd64.tar.gz"
      sha256 "5e838e2d13fd5c3700a16a38334b0613166cfe67fcc5f8bfbc6db7979b7bfcb0"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
