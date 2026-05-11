class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.18.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.2/tw-mcp_1.18.2_darwin_arm64.tar.gz"
      sha256 "717ac1b2e5a3990b99b585b0be783c9f816fd780c5904f0b6ae321a77b42d30d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.18.2/tw-mcp_1.18.2_darwin_amd64.tar.gz"
      sha256 "aed1ec08868244555256068d01d93df5ac2e86392139dec42e03bd4698734bb4"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
