class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.25.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.2/tw-mcp_1.25.2_darwin_arm64.tar.gz"
      sha256 "8e1873ce4023407887dad10722dc9b4f7158bb34ce1756a6ba9f6c2838972a8f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.25.2/tw-mcp_1.25.2_darwin_amd64.tar.gz"
      sha256 "2fdc9dbb8e64568dd61d598f45e298b5293315af8671a8390b1726d59f9a4061"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
