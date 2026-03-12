class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.2/tw-mcp_1.11.2_darwin_arm64.tar.gz"
      sha256 "74b78876461f1899ce6820cb2d721b0ec893ac5f9d0b32cf7a244dad54f85fea"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.2/tw-mcp_1.11.2_darwin_amd64.tar.gz"
      sha256 "eeee566449668dc26b5fdb8ee831c943de2860d375dcc10f024901fb27a97333"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
