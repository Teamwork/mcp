class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.3/tw-mcp_1.14.3_darwin_arm64.tar.gz"
      sha256 "42f38d2d02caa8f308e254ed8540b98b6f2c9b44ae42e251c2de9aad3a9a425f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.3/tw-mcp_1.14.3_darwin_amd64.tar.gz"
      sha256 "b70b4e68b24fbf80a9af31c76d829b1423b1caa036b98a356d5ce38bc0eadc0c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
