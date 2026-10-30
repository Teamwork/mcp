class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.60.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.1/tw-mcp_1.60.1_darwin_arm64.tar.gz"
      sha256 "235bef992cbe6897d18dbabe8260aa34cd59430d75fe28a2a4b3eba8a01e0d75"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.1/tw-mcp_1.60.1_darwin_amd64.tar.gz"
      sha256 "5c0c4dbc07600c8bcb78cb990b4db24441846d65da35bc04ca71ae9e2df32c5a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
