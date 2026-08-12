class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.28.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.0/tw-mcp_1.28.0_darwin_arm64.tar.gz"
      sha256 "306bddf1c8ba8a32caac1bb9ee7455542ed6b999045f2a5fb940e85b20f32a35"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.28.0/tw-mcp_1.28.0_darwin_amd64.tar.gz"
      sha256 "c61a2a13ba6860f4ada835041cc06853a54e05a61eba408770bac7bd0e28aa52"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
