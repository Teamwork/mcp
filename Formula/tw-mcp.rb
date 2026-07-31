class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.26.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.4/tw-mcp_1.26.4_darwin_arm64.tar.gz"
      sha256 "0157df9d62bde0a5948057bb9951f8b90fa12f6aeacffb13dcedf826fb90c7c1"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.4/tw-mcp_1.26.4_darwin_amd64.tar.gz"
      sha256 "6aa24d431b93de1255f6627f3ea91afcbaf1c32b9704c484be88a7a911336f1a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
