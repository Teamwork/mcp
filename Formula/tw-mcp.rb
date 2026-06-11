class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.2/tw-mcp_1.21.2_darwin_arm64.tar.gz"
      sha256 "cb5214e893d7f3d0d69109ae7f68ab7fd044ca27e0c39a4322d1b49d2e672bab"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.2/tw-mcp_1.21.2_darwin_amd64.tar.gz"
      sha256 "a542f415b11b253cd2999ca6559a0599d04dde119fae77c2b9c4f012bc43e1e5"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
