class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.43.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.43.0/tw-mcp_1.43.0_darwin_arm64.tar.gz"
      sha256 "a36c77328e02404bc264bc3bbdf4c6dbee3069a00c608e1b9dc1e9809e05285f"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.43.0/tw-mcp_1.43.0_darwin_amd64.tar.gz"
      sha256 "a0fd2e5bf588fca176563588bc775adc7fcd72c864bc56b83cc3d3ec49f399d3"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
