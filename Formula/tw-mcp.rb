class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.8"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.8/tw-mcp_1.14.8_darwin_arm64.tar.gz"
      sha256 "9a89f9950aaece122c8249f000b164836b8e7a364e954f3f238aa53f69f5473b"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.8/tw-mcp_1.14.8_darwin_amd64.tar.gz"
      sha256 "ed75ccfcf09f69f764c385309161cc99e15570da1cd453525c2eafca3e6778c1"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
