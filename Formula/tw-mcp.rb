class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.27.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.1/tw-mcp_1.27.1_darwin_arm64.tar.gz"
      sha256 "00e116a699b016e0cc81b0f50162f637f6cc936130fe10fa58bd0e5e05e1f5ec"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.1/tw-mcp_1.27.1_darwin_amd64.tar.gz"
      sha256 "31fdfdced881e4b74890fe03e7363f6715ecbe796035fb89a061bf12e4d03225"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
