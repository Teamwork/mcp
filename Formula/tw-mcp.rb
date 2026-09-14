class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.43.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.43.1/tw-mcp_1.43.1_darwin_arm64.tar.gz"
      sha256 "856deb26edfc54bef87f96f2a1f63bb555405ca7546784eb0da1f6e5098771b1"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.43.1/tw-mcp_1.43.1_darwin_amd64.tar.gz"
      sha256 "05cb3b911ff93ba0583df71389b840638b700ef6242855a3e603385b0f3e1173"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
