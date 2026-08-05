class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.26.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.5/tw-mcp_1.26.5_darwin_arm64.tar.gz"
      sha256 "e223927568684392e5ebdb787796c428cc921c26c37cb45c1572ef90b28ca107"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.5/tw-mcp_1.26.5_darwin_amd64.tar.gz"
      sha256 "57735f6cdf2ab8f036bb5b4d4ca760c168ec2362a342e824301ef7750b5a07a6"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
