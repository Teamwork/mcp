class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.14.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.1/tw-mcp_1.14.1_darwin_arm64.tar.gz"
      sha256 "a4801a40b02dfac4a4f0d5b738e4d5c251d38e52995853fea3969e0ed80137d6"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.14.1/tw-mcp_1.14.1_darwin_amd64.tar.gz"
      sha256 "94da89c5b30e9de8ce2974e6cf536ec78185bf6789efee3ce6ec65dd786559ad"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
