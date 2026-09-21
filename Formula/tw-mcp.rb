class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.44.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.44.0/tw-mcp_1.44.0_darwin_arm64.tar.gz"
      sha256 "c5a16158eb9f6c49f68a77b13e4adcf637e1453098630d3d1a0aa679a899caa6"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.44.0/tw-mcp_1.44.0_darwin_amd64.tar.gz"
      sha256 "0d7aa6bd26332a77aa3fdcf89aa1027fc077d97989de065bf80909077e4eee29"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
