class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.12.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.12.0/tw-mcp_1.12.0_darwin_arm64.tar.gz"
      sha256 "3d94c9cb361d9aa39dd099e4d11b79b842e88ba027d83d56b080edd7001f9cb0"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.12.0/tw-mcp_1.12.0_darwin_amd64.tar.gz"
      sha256 "77c1908d3760b39e88c85c9ce3c534d291b1eed73c5a671f377cd6807c5d6917"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
