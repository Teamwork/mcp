class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.28.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.5/tw-mcp_1.28.5_darwin_arm64.tar.gz"
      sha256 "ae8e6d7a7ee26039e90aacb9110ef95214658797cee495dd36893ce2c81f0f01"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.5/tw-mcp_1.28.5_darwin_amd64.tar.gz"
      sha256 "39e7ef05183b6e0e25004170b59d9ff5c9f878515df54ac1f025cde36b5ec5d2"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
