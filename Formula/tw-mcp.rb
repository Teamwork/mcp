class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.27.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.2/tw-mcp_1.27.2_darwin_arm64.tar.gz"
      sha256 "b056469f46d6b9e369c0d2433bf6cc88576c2d7cf2f5dc847b02d7dddcaf6921"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.2/tw-mcp_1.27.2_darwin_amd64.tar.gz"
      sha256 "bb950e9f4afe27ae32b8ee1609d45181cbb911a07a993c9323be0939dc192ef3"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
