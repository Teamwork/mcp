class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.26.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.0/tw-mcp_1.26.0_darwin_arm64.tar.gz"
      sha256 "af48671af5f6a0ce1efdf604a86b348b8b981caeca313acf091125ea3b1bf685"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.0/tw-mcp_1.26.0_darwin_amd64.tar.gz"
      sha256 "912466ef685267b5a53058c6ab3c14ddaa4ec4d0518d273de05babc0e11b686b"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
