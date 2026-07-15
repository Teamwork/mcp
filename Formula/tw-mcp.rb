class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.24.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.24.0/tw-mcp_1.24.0_darwin_arm64.tar.gz"
      sha256 "6fbde5d3850263e60ef4c44d3fd513ceec317f7f0282d184f1a5b5fac44b9fe2"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.24.0/tw-mcp_1.24.0_darwin_amd64.tar.gz"
      sha256 "74a28aa460ca4ddd780b0cb76ef2e5e7ac57a82ee85e64d9aebe9edabd783029"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
