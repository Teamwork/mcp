class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.47.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.47.0/tw-mcp_1.47.0_darwin_arm64.tar.gz"
      sha256 "57cc240cf44da4f349df27022b0cdb84c456b08e7954279bb5e6bc07a8200611"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.47.0/tw-mcp_1.47.0_darwin_amd64.tar.gz"
      sha256 "19dced7f73b305cd83400bc8099244688a67c863e469c915601a408a3a3ae32a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
