class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.28.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.2/tw-mcp_1.28.2_darwin_arm64.tar.gz"
      sha256 "534f915e651c131e44b001084665b3d473d11283055cf3b616b2e52f3456510a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.2/tw-mcp_1.28.2_darwin_amd64.tar.gz"
      sha256 "cd7fe3702ec188a9b1b75a03b56b79e990bd67af1d2aecdd0d6df64d892da261"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
