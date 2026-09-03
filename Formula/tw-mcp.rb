class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.38.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.38.1/tw-mcp_1.38.1_darwin_arm64.tar.gz"
      sha256 "37c6466f6bffe260ecf9891dcea78300f88e9021913189a48284694c3bd3216c"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.38.1/tw-mcp_1.38.1_darwin_amd64.tar.gz"
      sha256 "6980054ec8b05e3f4aae2461a6050b7b22dc37e35930a85d68b86298f761cb4c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
