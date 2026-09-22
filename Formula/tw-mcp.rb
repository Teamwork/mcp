class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.45.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.45.0/tw-mcp_1.45.0_darwin_arm64.tar.gz"
      sha256 "c2c03d9792e77f69df9f09f1f596d3d516455bba5756aaa7f22469242a828c38"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.45.0/tw-mcp_1.45.0_darwin_amd64.tar.gz"
      sha256 "56b66fe43b153927e4e12b8fa5e047770be6b92c8e0a192ca3df4af6a81b529d"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
