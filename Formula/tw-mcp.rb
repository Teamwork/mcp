class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.20.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.3/tw-mcp_1.20.3_darwin_arm64.tar.gz"
      sha256 "8362efb9ba2f599a854c4d44c68b5fcf0f8fcf11689f15b9c09b6cac79819130"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.3/tw-mcp_1.20.3_darwin_amd64.tar.gz"
      sha256 "9e86f9e77487334619da555bab8b0520a8818ec2ab11acade81d0eef132750be"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
