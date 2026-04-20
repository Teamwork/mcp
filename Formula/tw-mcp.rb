class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.5/tw-mcp_1.14.5_darwin_arm64.tar.gz"
      sha256 "1ae9c40ece6bd38294f5e9c46d4a0773556dcecafe3da89bfca9cabe63560fd6"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.5/tw-mcp_1.14.5_darwin_amd64.tar.gz"
      sha256 "d29856a2acb547fb8a3f2d5625495fcf747c622a6c3a2522b6a8e1be01f0df75"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
