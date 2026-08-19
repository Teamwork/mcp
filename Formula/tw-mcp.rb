class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.30.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.30.0/tw-mcp_1.30.0_darwin_arm64.tar.gz"
      sha256 "c93325418eed5b4ffc4d285b096d58cd802fb39de744d4c4325f314e1ed0b0a2"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.30.0/tw-mcp_1.30.0_darwin_amd64.tar.gz"
      sha256 "13dbb03cb6cb44ef228bc03b48ad10e5c2dcab5c9041ccda5afee6651bc15d2c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
