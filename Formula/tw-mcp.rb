class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.19.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.19.1/tw-mcp_1.19.1_darwin_arm64.tar.gz"
      sha256 "176e05aba9b9c511f9223a76635e5dd85c4b799e8f882e6981678c86dd0d421f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.19.1/tw-mcp_1.19.1_darwin_amd64.tar.gz"
      sha256 "ae209ff3e48a99e041941c62208f5c4324fec872025a140430f828af33f54901"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
