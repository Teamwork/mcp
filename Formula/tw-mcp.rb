class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.37.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.37.0/tw-mcp_1.37.0_darwin_arm64.tar.gz"
      sha256 "6df29486a6d8b1db0b6be22b7cb344a0f277b2dfdb498462d56546112c7c4b2e"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.37.0/tw-mcp_1.37.0_darwin_amd64.tar.gz"
      sha256 "ad2ee78306d04a23acdefe62fd373821eedd5565ca4b5a00d7c3dae9d42f023b"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
