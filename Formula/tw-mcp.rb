class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.23.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.1/tw-mcp_1.23.1_darwin_arm64.tar.gz"
      sha256 "448c919ffdb76e6f355f4ae622c5fdf662b7dfba6a32a3c3bae2edcba524164a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.1/tw-mcp_1.23.1_darwin_amd64.tar.gz"
      sha256 "690c4683d214c8c6fd69951f1b1c259b00546b15c7184d35f142f6ee20bd7648"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
