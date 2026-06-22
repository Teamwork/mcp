class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.5/tw-mcp_1.21.5_darwin_arm64.tar.gz"
      sha256 "15dd779e117390d81bb5a08b56ed77386d0b1378f656ecc0c13bf71b25df1e8b"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.5/tw-mcp_1.21.5_darwin_amd64.tar.gz"
      sha256 "c0e7d98eb7ca01d2579d6dc5a5ea2e981ada8f50b37313aed903e9bef9c5721a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
