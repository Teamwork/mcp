class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.0/tw-mcp_1.14.0_darwin_arm64.tar.gz"
      sha256 "836f7811572be40e9212b2d895489a31d86d60f49a8a66803c3199a8cb76cd25"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.0/tw-mcp_1.14.0_darwin_amd64.tar.gz"
      sha256 "99ca1ed56ff51695786d0ae1efca5ff82e56d6c6a773077551ac5f55a8bc0d00"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
