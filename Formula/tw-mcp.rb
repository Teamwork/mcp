class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.39.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.3/tw-mcp_1.39.3_darwin_arm64.tar.gz"
      sha256 "6e8643aa83e8fca2aeb9c177da8e9895848416b57abb06374e4ec6f05a3d33db"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.3/tw-mcp_1.39.3_darwin_amd64.tar.gz"
      sha256 "84a8d3067864b08f83dffac488d6a9295e6cba52e21254298748d15380a8a3f3"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
