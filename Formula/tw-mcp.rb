class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.17.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.1/tw-mcp_1.17.1_darwin_arm64.tar.gz"
      sha256 "fece8ce6e2095fde6f55dafd6f8712bce6e780f9b52222442835450c9468149d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.17.1/tw-mcp_1.17.1_darwin_amd64.tar.gz"
      sha256 "8b7482e7a56a84847edbfb65f844eae0695e02253a6b7ec75d80776c61a29cd1"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
