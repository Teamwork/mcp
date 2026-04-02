class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.9"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.9/tw-mcp_1.11.9_darwin_arm64.tar.gz"
      sha256 "9c88866bb783eeced2cca0ead3a86bc65d4267f6c5bf60d2d1c0523d185af17a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.9/tw-mcp_1.11.9_darwin_amd64.tar.gz"
      sha256 "8a435515772cd1ade158bc9b18efec72513afeeb3c2e6fa29d7cdb580d46e8df"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
