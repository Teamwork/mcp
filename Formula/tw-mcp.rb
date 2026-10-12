class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.51.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.2/tw-mcp_1.51.2_darwin_arm64.tar.gz"
      sha256 "077481f85116cea19407591f98b641362bcf90eb87d9d84ed517422e87959d56"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.51.2/tw-mcp_1.51.2_darwin_amd64.tar.gz"
      sha256 "c205f0edb8cba05c43225481c3ab6618eba045acdf395c241fdd1bb498f08b92"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
