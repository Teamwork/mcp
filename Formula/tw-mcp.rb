class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.28.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.4/tw-mcp_1.28.4_darwin_arm64.tar.gz"
      sha256 "ebfefa4f591962786332606cafc4e500321414b3d343b13b467bb5eac8544c1d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.4/tw-mcp_1.28.4_darwin_amd64.tar.gz"
      sha256 "f6b1b1b474e785fc13d6c661b9445479a313aef54037c23aaa208e3ae682e10c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
