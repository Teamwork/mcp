class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.26.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.3/tw-mcp_1.26.3_darwin_arm64.tar.gz"
      sha256 "7ae8157431f365b5d0c29594ef2bf264a6e0fa4c8442d96549285c2434a1b719"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.3/tw-mcp_1.26.3_darwin_amd64.tar.gz"
      sha256 "a7b38c5210c7b9dc2e9559be51c6a154c42d8feda4bff2b842c9a7d0fa1d1477"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
