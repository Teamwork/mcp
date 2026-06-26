class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.23.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.0/tw-mcp_1.23.0_darwin_arm64.tar.gz"
      sha256 "805bf827f32f62487fc879a3f96c1bf717a54ef9023868bd7d9e13718bccd226"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.23.0/tw-mcp_1.23.0_darwin_amd64.tar.gz"
      sha256 "cfd8c8a6e859692d98964868c2c2beaf99d284470e76775af12647b40de71371"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
