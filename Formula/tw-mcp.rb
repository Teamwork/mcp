class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.24.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.24.1/tw-mcp_1.24.1_darwin_arm64.tar.gz"
      sha256 "7605c3f832458fa816142a67280f934776247f4ed625ba8b9b68c37a24f347fa"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.24.1/tw-mcp_1.24.1_darwin_amd64.tar.gz"
      sha256 "3215cf2f04731662914015a6876d0c0a3a67ed0ce4b35e009c4fc0515bbdcdf8"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
