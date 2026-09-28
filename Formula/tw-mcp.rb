class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.48.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.48.2/tw-mcp_1.48.2_darwin_arm64.tar.gz"
      sha256 "93c50868a54f2d93cd924089b999d4fa64199826a76f8793b400fad70f0f7347"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.48.2/tw-mcp_1.48.2_darwin_amd64.tar.gz"
      sha256 "8b0fe447e29a41024c3b0c7ee6f5bf26cc610a2a427390ef9147abfad2598711"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
