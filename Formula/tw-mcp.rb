class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.49.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.49.0/tw-mcp_1.49.0_darwin_arm64.tar.gz"
      sha256 "da262d9607ac0db7343fa081f9f5cb539d35804fe97ddd5a8bb63179fe80f5b3"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.49.0/tw-mcp_1.49.0_darwin_amd64.tar.gz"
      sha256 "a1dd64b3e6acf5aa23791838e5ef8865638b504988de249ed7082f96d9c96d02"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
