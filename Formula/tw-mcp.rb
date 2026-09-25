class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.48.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.48.1/tw-mcp_1.48.1_darwin_arm64.tar.gz"
      sha256 "0fa64bc3dd7e8e7aecf3f89cd5cfeb422e85904ff7b217e6064f78cfad5273ee"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.48.1/tw-mcp_1.48.1_darwin_amd64.tar.gz"
      sha256 "745a4e90848f3cb757a6ae06f1787f111e2013bafea6402fcb9d339395e225b0"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
