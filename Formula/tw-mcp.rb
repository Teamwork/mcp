class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.42.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.42.0/tw-mcp_1.42.0_darwin_arm64.tar.gz"
      sha256 "ef4f89c964459de3fd2f961a30d30e58a4ecb665f1217f4cc80bbbcc8dbe2d87"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.42.0/tw-mcp_1.42.0_darwin_amd64.tar.gz"
      sha256 "7bf9261f43e1bc150b80ab6372b135b488c957dd691ab9e82b42ab4c124b2624"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
