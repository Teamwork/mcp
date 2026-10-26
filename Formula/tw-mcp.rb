class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.60.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.0/tw-mcp_1.60.0_darwin_arm64.tar.gz"
      sha256 "dc985467019ed2b02aec57832d40639c40c949c2a3802b820cf7cb35bdf1972f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.0/tw-mcp_1.60.0_darwin_amd64.tar.gz"
      sha256 "1decb7a29c1358c8b8a77e8e4568d9c3a8b50b6748c221e82ac7ca2c74a5b61d"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
