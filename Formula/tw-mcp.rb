class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.51.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.0/tw-mcp_1.51.0_darwin_arm64.tar.gz"
      sha256 "73ab3fa0ebafd0f24d1e97754f2694e2d669eed096124a76f427a979397fed23"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.0/tw-mcp_1.51.0_darwin_amd64.tar.gz"
      sha256 "b059ec337a6be0efbada415b3137cf71e09f7263756028176bdd88ae83e0e4d3"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
