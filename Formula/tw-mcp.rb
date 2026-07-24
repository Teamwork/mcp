class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.25.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.3/tw-mcp_1.25.3_darwin_arm64.tar.gz"
      sha256 "7755b6e9a2fe3eb0282d96f3de70a359baf759a2a71fba58d96a142cfac8f89e"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.3/tw-mcp_1.25.3_darwin_amd64.tar.gz"
      sha256 "0aae9e780fa3936ad9365da1f65c9249757ff696bae9e01ade3a7946f3cdbdb5"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
