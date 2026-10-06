class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.51.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.1/tw-mcp_1.51.1_darwin_arm64.tar.gz"
      sha256 "290ffe63917a7a4667b835584d5daed4487eb85b39919d82d0f3e0455ba51223"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.1/tw-mcp_1.51.1_darwin_amd64.tar.gz"
      sha256 "8aa7e732c8a5bf0068a19d91e298cbe0158dbbdf6d4d2fe153e126615b812d2e"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
