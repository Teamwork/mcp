class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.26.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.2/tw-mcp_1.26.2_darwin_arm64.tar.gz"
      sha256 "88c8031541b6730fae3ce79c039949afa41754a14a5c8658e833aba990d4c476"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.26.2/tw-mcp_1.26.2_darwin_amd64.tar.gz"
      sha256 "b211b2999b45dd7569368db5e260348362fec57634d9680d106642122dbc3fe6"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
