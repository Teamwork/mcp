class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/teamwork/mcp"
  version "1.11.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/teamwork/mcp/releases/download/v1.11.0/tw-mcp_1.11.0_darwin_arm64.tar.gz"
      sha256 "48ee2f7ec63b3f11043569c92e61cce65c6948ab6a4a9bb564c5e518d76965bc"
    else
      url "https://github.com/teamwork/mcp/releases/download/v1.11.0/tw-mcp_1.11.0_darwin_amd64.tar.gz"
      sha256 "6fb74eb4712929c2ba13f71247968b43f09fa1afa4d108d1e9aafc27ffc8ba11"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end