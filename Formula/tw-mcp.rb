class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.41.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.41.1/tw-mcp_1.41.1_darwin_arm64.tar.gz"
      sha256 "07cfbe66d04646e45b69ca24efe1405535f1eadf623f68fb213a54346c6d9605"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.41.1/tw-mcp_1.41.1_darwin_amd64.tar.gz"
      sha256 "22167846422dd1df98bb624d4e91222714d7604936f03bbc72c8dc75c3b39221"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
