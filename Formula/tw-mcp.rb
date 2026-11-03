class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.60.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.2/tw-mcp_1.60.2_darwin_arm64.tar.gz"
      sha256 "b49b7c53e6ed1e3a382422e0bb33e00a7effa62b790adca8056e35cd84121e8a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.60.2/tw-mcp_1.60.2_darwin_amd64.tar.gz"
      sha256 "7fc7d76780a33b0e0d4cf74e9db68b31aed773a4622be80cd89e059b92fb2f2c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
