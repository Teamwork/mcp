class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.25.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.1/tw-mcp_1.25.1_darwin_arm64.tar.gz"
      sha256 "488619ef470c3ab255fe4d6eaa215cda17fdd2efb93679e1214f529d7c5f23a2"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.1/tw-mcp_1.25.1_darwin_amd64.tar.gz"
      sha256 "51407a4bd1689ace45243bbc89576951ea4bcc6e86349648a06d517e80eb7050"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
