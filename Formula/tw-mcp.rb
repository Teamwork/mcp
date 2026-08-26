class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.35.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.35.0/tw-mcp_1.35.0_darwin_arm64.tar.gz"
      sha256 "7a0462e83bd3a839b57edfc812785f412d5a026abf117ca6cc6c3a5686d17124"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.35.0/tw-mcp_1.35.0_darwin_amd64.tar.gz"
      sha256 "3698be958c15216feca48333b345a0660ad351dddb796be41189772edf9663d6"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
