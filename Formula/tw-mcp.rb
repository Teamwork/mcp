class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.15.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.15.1/tw-mcp_1.15.1_darwin_arm64.tar.gz"
      sha256 "6ab27ddc109a173cc6914b36ec637fcef0d56e1b861a90a38f3a9d4814fe843e"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.15.1/tw-mcp_1.15.1_darwin_amd64.tar.gz"
      sha256 "2793d728af0ff2535c4c634d80f8e3f9d89baffb31edd14dae8938bf24cdf375"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
