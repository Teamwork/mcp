class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.35.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.35.1/tw-mcp_1.35.1_darwin_arm64.tar.gz"
      sha256 "b5ed5e0928104c90c0a0c571b9adc5596cb330aeb410053bddf1fe3549877e0e"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.35.1/tw-mcp_1.35.1_darwin_amd64.tar.gz"
      sha256 "8ad7d7e40cbbee401bedf5c25f86b646578c88f2d29e306a35d4bc91449f8007"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
