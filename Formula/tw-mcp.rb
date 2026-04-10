class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.13.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.13.0/tw-mcp_1.13.0_darwin_arm64.tar.gz"
      sha256 "ba2dcc2a7423d240d56a5209475ef56619f666093dd6366b85acca3378a70307"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.13.0/tw-mcp_1.13.0_darwin_amd64.tar.gz"
      sha256 "66d8c24bb8deb7c4a3a0fb8e6f9d930ca831656971858355557ae6f3d8b490db"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
