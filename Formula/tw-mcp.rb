class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.62.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.62.1/tw-mcp_1.62.1_darwin_arm64.tar.gz"
      sha256 "990662ecfc25011fd39081020382af7b5488eb2029eef8fc2ff81553e93eeaac"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.62.1/tw-mcp_1.62.1_darwin_amd64.tar.gz"
      sha256 "4ad4f81111ed5a0261ffcfabc9ae2cd1e6b749322337fe0141169ac6821124a3"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
