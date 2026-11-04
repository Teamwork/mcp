class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.62.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.62.0/tw-mcp_1.62.0_darwin_arm64.tar.gz"
      sha256 "b576542864ea50f2b91c2de86bb4b445009d9023748c4ea49619f80a838d1cb5"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.62.0/tw-mcp_1.62.0_darwin_amd64.tar.gz"
      sha256 "83b4c069e39ce8112582d2a0b6f2ebe58e68e40bb021e39b119a3962634c3f44"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
