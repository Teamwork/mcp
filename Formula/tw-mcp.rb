class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.18.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.0/tw-mcp_1.18.0_darwin_arm64.tar.gz"
      sha256 "f56c78394389910e7c0be24ec11e5b161579d317c4e83d82b13c470aa9c3483f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.0/tw-mcp_1.18.0_darwin_amd64.tar.gz"
      sha256 "fbca3c0a06716bf8efce21ba894869e22aadf1b60cab525748a62c5825f406bd"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
