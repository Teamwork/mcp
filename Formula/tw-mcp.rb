class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.15.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.15.0/tw-mcp_1.15.0_darwin_arm64.tar.gz"
      sha256 "97c164dfa14fe6be71b09a43a8a4aa88b07d9386792853600314ce5b165a65b9"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.15.0/tw-mcp_1.15.0_darwin_amd64.tar.gz"
      sha256 "cb07093ced8c5ae12f7e31e9e1ba5a51172ad0adc1a9e2e350aca8e4bf6f534a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
