class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.46.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.46.0/tw-mcp_1.46.0_darwin_arm64.tar.gz"
      sha256 "a2cd93207cc366a46b1ae3d9665e74a7c75448b0f4953058278c9e681dc06b94"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.46.0/tw-mcp_1.46.0_darwin_amd64.tar.gz"
      sha256 "bf84e58c550f74e240709293aa48df934827ba276b0eecc9597bd8b703469daf"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
