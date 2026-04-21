class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.7"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.7/tw-mcp_1.14.7_darwin_arm64.tar.gz"
      sha256 "289333afef858b67881aec8b11f566371583db76f1e4654ffc646fc0d023e60a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.7/tw-mcp_1.14.7_darwin_amd64.tar.gz"
      sha256 "924b499af3574257c59336469f510d7ed9dcf203c9d43f0af95b49bda7e68a7e"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
