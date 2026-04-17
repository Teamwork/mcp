class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.2/tw-mcp_1.14.2_darwin_arm64.tar.gz"
      sha256 "7ca50982eff8afd85ed73d4cf7577a5f1d9fdb655ab4f4a56fa5f67d1e51b51f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.2/tw-mcp_1.14.2_darwin_amd64.tar.gz"
      sha256 "4166b6f641a2f060b9aaf53d2d51fa347df273a41b374b214932cfe12fba1021"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
