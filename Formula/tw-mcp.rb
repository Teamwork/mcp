class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.23.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.4/tw-mcp_1.23.4_darwin_arm64.tar.gz"
      sha256 "55b22969f7914af20154db8a957f2e281618b8d206224b14f1acadef741e48b6"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.4/tw-mcp_1.23.4_darwin_amd64.tar.gz"
      sha256 "3c0643c290db0f84cff291789c02c054c7d55d3ea464191731cceb51de17ddcd"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
