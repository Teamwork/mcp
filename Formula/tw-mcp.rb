class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.28.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.1/tw-mcp_1.28.1_darwin_arm64.tar.gz"
      sha256 "cfa0ec9779f14c433d561a7485f001f8556d66faab1a055f55e57025e52930aa"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.1/tw-mcp_1.28.1_darwin_amd64.tar.gz"
      sha256 "4631b6d522573471e2304a55844b9ec2644fa36fb36b26ddb763e5b2c52fe01a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
