class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.40.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.40.1/tw-mcp_1.40.1_darwin_arm64.tar.gz"
      sha256 "e8a4ea18e34eeee20973c07afc57e6ca72b6ab764f775a0d6362cf7cb5e7b2b6"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.40.1/tw-mcp_1.40.1_darwin_amd64.tar.gz"
      sha256 "c33b53972598ec345c2ddf5fd4867f54c24a706d9e1d1b96a10bc53586cc8faf"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
