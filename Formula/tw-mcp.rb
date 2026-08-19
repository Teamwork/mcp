class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.29.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.29.0/tw-mcp_1.29.0_darwin_arm64.tar.gz"
      sha256 "401bba550fe2e53270f3574c06e77f5cadcc30424d44c330ff24a1d97384e6a4"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.29.0/tw-mcp_1.29.0_darwin_amd64.tar.gz"
      sha256 "4db7a3a61678924e545865323f96f37d463e82d6e7c9572942ee66ddcd9587f7"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
