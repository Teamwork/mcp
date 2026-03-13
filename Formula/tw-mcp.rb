class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.4/tw-mcp_1.11.4_darwin_arm64.tar.gz"
      sha256 "fa424311cffbdb589d023eedf64e2b1dac12770d3766fba9a760ee542c4b738f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.4/tw-mcp_1.11.4_darwin_amd64.tar.gz"
      sha256 "f8dfeaae399b3300942bfe8c97e2b9e0509f94e62bdb03133a0a7e845d28b910"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
