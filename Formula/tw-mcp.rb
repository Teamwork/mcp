class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.4/tw-mcp_1.14.4_darwin_arm64.tar.gz"
      sha256 "2492c4339d8af0dbfc7139002fece10e7adb66fb2291ec3e0ff2200f0044898c"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.4/tw-mcp_1.14.4_darwin_amd64.tar.gz"
      sha256 "35355c9b9bb26efc9c6b900c23d220c97114a405e43decd8c01dd70777265344"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
