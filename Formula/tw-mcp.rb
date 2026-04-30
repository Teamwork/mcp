class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.16.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.16.1/tw-mcp_1.16.1_darwin_arm64.tar.gz"
      sha256 "f6b4e0a430fdcec68f470316c780cfb2a4f4f271036aa128a1a31925786ce74e"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.16.1/tw-mcp_1.16.1_darwin_amd64.tar.gz"
      sha256 "004caa35020d96bdd89528ee56ca3ad711dde276c27f4cdc611b159e0093ecce"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
