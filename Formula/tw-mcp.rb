class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.40.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.40.0/tw-mcp_1.40.0_darwin_arm64.tar.gz"
      sha256 "8d18b67290f47225ba28a7a3b7d0e5e302e6d07ffeb9740503cc87fbf65309d1"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.40.0/tw-mcp_1.40.0_darwin_amd64.tar.gz"
      sha256 "9518cc1ba15dbf55bae67e16724cde7bdaffecc4691745bc7724646da12cbb5a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
