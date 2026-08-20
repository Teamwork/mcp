class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.32.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.32.0/tw-mcp_1.32.0_darwin_arm64.tar.gz"
      sha256 "f047daf0e2df84b62ced82488bd46c261fea49842228b0b9e74a13d7bb29a6b0"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.32.0/tw-mcp_1.32.0_darwin_amd64.tar.gz"
      sha256 "b45041baedd2f53562815d31caaa4b3ce55eeeadb1425013adad202c3c41e925"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
