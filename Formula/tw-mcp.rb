class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.36.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.36.0/tw-mcp_1.36.0_darwin_arm64.tar.gz"
      sha256 "c521a4f5d273e8d9b7c25f7c2648c0ab8519cf30ff405b9850a837a248255f4b"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.36.0/tw-mcp_1.36.0_darwin_amd64.tar.gz"
      sha256 "404e1f33fe873a7806153e543cebff7f7982332537215ff4d74d1753b7b97d90"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
