class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.7"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.7/tw-mcp_1.11.7_darwin_arm64.tar.gz"
      sha256 "bfe3a3cdeee1a3ef39c425351f8a78688d42eecfe1c0622859de6563a7600736"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.7/tw-mcp_1.11.7_darwin_amd64.tar.gz"
      sha256 "af0aada552ab57e07e8128e77b2001450e546f08994d97af752a5f29fb306c20"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
