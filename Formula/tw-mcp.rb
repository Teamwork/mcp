class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.28.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.3/tw-mcp_1.28.3_darwin_arm64.tar.gz"
      sha256 "947f2b9d0651cf51bb504d680bc6d567134027fbbcee1b9d9bb9547357a9edb5"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.3/tw-mcp_1.28.3_darwin_amd64.tar.gz"
      sha256 "d1087acc1f0cde2db54eec16c0da8b018a415cb48e8752d388378a9b07c715ca"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
