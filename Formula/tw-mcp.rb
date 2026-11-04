class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.61.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.61.0/tw-mcp_1.61.0_darwin_arm64.tar.gz"
      sha256 "88d62c36a1a1d9d6e1936d176d460a990a50a476a55ea4b5ff98eca81b02c189"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.61.0/tw-mcp_1.61.0_darwin_amd64.tar.gz"
      sha256 "b9cac18ec5e9d69c30ecec57ca9645f74eb376503ccb742da15c237bcca3f103"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
