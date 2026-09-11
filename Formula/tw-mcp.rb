class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.41.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.41.0/tw-mcp_1.41.0_darwin_arm64.tar.gz"
      sha256 "38aeb457a266d0e2ece3db4869251e49b3c6d7b2bfffc887b0162db4226a2974"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.41.0/tw-mcp_1.41.0_darwin_amd64.tar.gz"
      sha256 "a055774688e70c159e01f03c966e4f9a17109052029747d11b7d5db28c79e558"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
