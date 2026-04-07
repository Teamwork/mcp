class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.11"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.11/tw-mcp_1.11.11_darwin_arm64.tar.gz"
      sha256 "d80df89bd9320d4c0b0faec88f92f0b54e13984b8d385a4e65a88faf2a874afa"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.11/tw-mcp_1.11.11_darwin_amd64.tar.gz"
      sha256 "794d72fcc47dc5212fd0d83d846a0c1374aa81d5d049e668408a550acc7a128d"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
