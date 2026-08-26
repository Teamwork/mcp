class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.34.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.34.0/tw-mcp_1.34.0_darwin_arm64.tar.gz"
      sha256 "d9f243bcf92db3d16115bae865686c628760399bb75f5e3c4b3f1f706c465442"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.34.0/tw-mcp_1.34.0_darwin_amd64.tar.gz"
      sha256 "ecbcc03013f4b3956f373bf4fb904075b3929f38c20a70f243b8bf7f4f986eee"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
