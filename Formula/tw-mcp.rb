class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.3/tw-mcp_1.11.3_darwin_arm64.tar.gz"
      sha256 "3a8f8cb84960247ccca467853874c4b95472db86d4f32d50da95e3fcc9762199"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.3/tw-mcp_1.11.3_darwin_amd64.tar.gz"
      sha256 "4f1cd8fa9c97d73fbca642abdbf3c859b242e266ecef61967fdc81cd41454201"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
