class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.8"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.8/tw-mcp_1.11.8_darwin_arm64.tar.gz"
      sha256 "07a8667536a0ecbcdbb0bcc1244ea68df261be649233f66f95562ab0ecafe261"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.8/tw-mcp_1.11.8_darwin_amd64.tar.gz"
      sha256 "ac83b5c728928bcb1376e15656668cbd200c36720630f1c4bcc9c2d463e98ab2"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
