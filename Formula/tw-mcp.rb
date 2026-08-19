class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.31.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.31.0/tw-mcp_1.31.0_darwin_arm64.tar.gz"
      sha256 "b8f2520d6097d6247d4f277751ed05d3253cc7204b1f3cefc8e7ec34172cacd3"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.31.0/tw-mcp_1.31.0_darwin_amd64.tar.gz"
      sha256 "3c9b1532644b32c510b8fd1c1c11c78e5ad973e04de01d99b9ff74a9891bb83c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
