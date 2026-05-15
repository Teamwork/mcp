class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.20.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.0/tw-mcp_1.20.0_darwin_arm64.tar.gz"
      sha256 "2201ec8bf253e6d9cce9eb4064872e79919ac79b27619cfcef7096708d5bafe0"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.0/tw-mcp_1.20.0_darwin_amd64.tar.gz"
      sha256 "0ac779b70412990f66104117b833eb16dfe0ff7543d9ac94c832bbb1f8cb7e19"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
