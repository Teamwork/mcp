class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.38.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.38.0/tw-mcp_1.38.0_darwin_arm64.tar.gz"
      sha256 "134093205211f3ccd78c799f03e05968e8e9d565a0d1bcf45402d1776ab06d19"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.38.0/tw-mcp_1.38.0_darwin_amd64.tar.gz"
      sha256 "80467130d4ab6feb282000ea80ce7dda70996e7c2acf929858375d75d5de821f"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
