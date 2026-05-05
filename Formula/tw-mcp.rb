class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.17.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.3/tw-mcp_1.17.3_darwin_arm64.tar.gz"
      sha256 "a0e9bc24916e49e354d4243df83c71e965a5f440f71839cbb205b3a6ffec27e3"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.3/tw-mcp_1.17.3_darwin_amd64.tar.gz"
      sha256 "7c62cae5a4ddd59450f3149a8d7176966a5f4b336f2202b1220f01da0465154a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
