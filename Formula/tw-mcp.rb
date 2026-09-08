class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.39.2"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.2/tw-mcp_1.39.2_darwin_arm64.tar.gz"
      sha256 "cb9217c8cdbf17f0b9178961289afabdb36901e66e5062782404499d16cd7759"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.2/tw-mcp_1.39.2_darwin_amd64.tar.gz"
      sha256 "48799bbfcea5222bf8f06295b887616e1793d7fb96a00059ab570cf6334816a4"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
