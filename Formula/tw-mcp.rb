class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.20.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.2/tw-mcp_1.20.2_darwin_arm64.tar.gz"
      sha256 "61e4fab820e490a1a4f5f6c0c93ea40a129022f480c9e2029cbf64cb317ff032"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.2/tw-mcp_1.20.2_darwin_amd64.tar.gz"
      sha256 "0282a2a5b7101c5d7a7e581e331a503f71e26d6d16bfcb9e7f9be0e2e9563285"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
