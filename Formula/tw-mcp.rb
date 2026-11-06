class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.62.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.62.2/tw-mcp_1.62.2_darwin_arm64.tar.gz"
      sha256 "1f9262c5c2f90e21e599d03ccc8b3768296f6f59f2b72457a38dfd0c93e27cc6"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.62.2/tw-mcp_1.62.2_darwin_amd64.tar.gz"
      sha256 "fb6442db5839f4869b97d380e5062472101aad8ca63447b65593e33832adc535"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
