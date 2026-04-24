class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.15.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.15.2/tw-mcp_1.15.2_darwin_arm64.tar.gz"
      sha256 "0b7f2feca37f63814a65469e6d3f5053d2eeb862957275ffec0c6ffdcab14fc3"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.15.2/tw-mcp_1.15.2_darwin_amd64.tar.gz"
      sha256 "3490ddb3fad54209e3f0d4eb234299ea51a5ef816594682b2709d84519b8e4c5"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
