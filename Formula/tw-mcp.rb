class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.27.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.5/tw-mcp_1.27.5_darwin_arm64.tar.gz"
      sha256 "8476903d178aecbecef79d97c73cf17aabedebd3cb82a62eeccb349f7b046266"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.5/tw-mcp_1.27.5_darwin_amd64.tar.gz"
      sha256 "debd34616d2ede5b6463de5863c62e9b0670bc9e4437df84ca039ede6eb305de"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
