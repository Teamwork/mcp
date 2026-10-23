class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.51.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.3/tw-mcp_1.51.3_darwin_arm64.tar.gz"
      sha256 "391c75190183c8ee84abc9d43f389d86297bbb2433c5670d44e9a796978a163a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.3/tw-mcp_1.51.3_darwin_amd64.tar.gz"
      sha256 "51bfd4c9c7291f9730ff83bf77623e5651b43487e3f0fabfa50d805999898400"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
