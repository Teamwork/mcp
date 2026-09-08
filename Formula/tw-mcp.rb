class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.39.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.1/tw-mcp_1.39.1_darwin_arm64.tar.gz"
      sha256 "f6197318d001344a37954b266d4e02a37c8983a2f13df8990520f60c8f5e9964"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.1/tw-mcp_1.39.1_darwin_amd64.tar.gz"
      sha256 "1c4b055f80e4649ba68834f2e433ac13c57fccb6f344c08ede1859c11bbc6eb6"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
