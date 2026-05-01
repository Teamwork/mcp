class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.17.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.2/tw-mcp_1.17.2_darwin_arm64.tar.gz"
      sha256 "e44aed63f7cf132280bec420365464019a4e58be0846698de5d8aa63e08450b9"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.2/tw-mcp_1.17.2_darwin_amd64.tar.gz"
      sha256 "dd066d083d0c1be0a8209ae29fb3fc8ababb310f4cb2fc2d874e70e0b2c2d08a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
