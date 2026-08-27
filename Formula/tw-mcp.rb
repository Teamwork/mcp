class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.35.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.35.2/tw-mcp_1.35.2_darwin_arm64.tar.gz"
      sha256 "72ab51c967ed377e193413a769514ddc96ffe70b6e5e3fa4bf220893b1b67a2d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.35.2/tw-mcp_1.35.2_darwin_amd64.tar.gz"
      sha256 "64a0618e60e81c8253f370c2b656725f9f080f4dc4136a44444d34921978c63b"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
