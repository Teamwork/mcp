class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.17.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.0/tw-mcp_1.17.0_darwin_arm64.tar.gz"
      sha256 "9c5b7385d257a479d1c9852ad6ff165c32064c6c920b9889b62f595946436ea9"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.0/tw-mcp_1.17.0_darwin_amd64.tar.gz"
      sha256 "a9cbbde942b1fbeedff99001b63839aeb267a2dfc76af577c4a7abb020f24ce8"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
