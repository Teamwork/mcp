class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.4/tw-mcp_1.21.4_darwin_arm64.tar.gz"
      sha256 "7c70cfcffea83aded927d42251259a99096ecce4224dd08903cfb2aeeb7c688a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.4/tw-mcp_1.21.4_darwin_amd64.tar.gz"
      sha256 "514a0e60cccf663ceb42cc4455bd1af2e0bc6a123d7ef4e5bc95f471cb101e53"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
