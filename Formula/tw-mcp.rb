class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.33.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.33.1/tw-mcp_1.33.1_darwin_arm64.tar.gz"
      sha256 "b7155ea5bdaafb1ca67658d3ca29055445b0d4b67feede1ac94daaf7261011fa"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.33.1/tw-mcp_1.33.1_darwin_amd64.tar.gz"
      sha256 "28f802275b3b8957ed1ecdc3e92f47bfa995c1b2c4f0bcecfe527dcd381182de"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
