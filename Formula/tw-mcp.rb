class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.43.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.43.2/tw-mcp_1.43.2_darwin_arm64.tar.gz"
      sha256 "6278ad93432a77ac0562c5a428438ce2dc71c50dcb27dc6ebd01a7f0f06fc329"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.43.2/tw-mcp_1.43.2_darwin_amd64.tar.gz"
      sha256 "c793d4956074f39de84c2b3393d4a42b2bcb6f69886373d56e50e5f9da6a516e"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
