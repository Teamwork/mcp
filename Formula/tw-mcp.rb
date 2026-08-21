class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.33.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.33.0/tw-mcp_1.33.0_darwin_arm64.tar.gz"
      sha256 "7207ab0b82c64907c948a7845d7a93838fcafa63708a777dbcd6d83564f24dca"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.33.0/tw-mcp_1.33.0_darwin_amd64.tar.gz"
      sha256 "022907d02eac5307239a6dde0c98cd87bb3aed63ad934df9b807af162b327e97"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
