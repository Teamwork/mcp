class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.18.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.1/tw-mcp_1.18.1_darwin_arm64.tar.gz"
      sha256 "72b2ae26ebc3b3ecc482caab802041b97395c54824deba680f3950de13d19b9b"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.1/tw-mcp_1.18.1_darwin_amd64.tar.gz"
      sha256 "f216dc395f87f07e8e1048f3011e14a620382b933d827aa5a7ae0e006740e676"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
