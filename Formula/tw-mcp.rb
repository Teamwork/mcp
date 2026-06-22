class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.6"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.6/tw-mcp_1.21.6_darwin_arm64.tar.gz"
      sha256 "343f5737ff84107de51d185ba1dcc0abb221b8d33674046ea9547e5767c937bd"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.6/tw-mcp_1.21.6_darwin_amd64.tar.gz"
      sha256 "3eee7036e877e0f9018caadd387be6f287d5fa0cc01c5a3519c879f4207e61c1"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
