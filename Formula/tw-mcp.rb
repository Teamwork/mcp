class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.6"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.6/tw-mcp_1.14.6_darwin_arm64.tar.gz"
      sha256 "3702bbb2d99b011f4ad9bfdcb556904e7e990e11dde56534ad5a4b5c66c026c0"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.6/tw-mcp_1.14.6_darwin_amd64.tar.gz"
      sha256 "cf27b5b482928ab29ad0a243fa226fe03e1d545ead5fc5be8d124078eb61aa46"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
