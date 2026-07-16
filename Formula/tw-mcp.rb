class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.25.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.0/tw-mcp_1.25.0_darwin_arm64.tar.gz"
      sha256 "d737fd96c742ebb6045b8459a8ac61abe5693c0f8668904d0c3538a127c5a3c4"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.0/tw-mcp_1.25.0_darwin_amd64.tar.gz"
      sha256 "0a665c9b8dd8994597d084b8e1383398bc0dba1e2fecb4b54371e360dc2ee6a3"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
