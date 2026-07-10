class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.23.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.2/tw-mcp_1.23.2_darwin_arm64.tar.gz"
      sha256 "355fe4358442018fbd1692323e5ef99597651e9165adbf529a6854b5c57827bb"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.2/tw-mcp_1.23.2_darwin_amd64.tar.gz"
      sha256 "d5f2520b77e990f63121c06e70579908f5c5e317f2a906973728df073e0b0eac"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
