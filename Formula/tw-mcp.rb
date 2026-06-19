class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.3"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.3/tw-mcp_1.21.3_darwin_arm64.tar.gz"
      sha256 "c7e0faa4b697f7b9cc721dd7f47e0350d7ff4e2adf64328ad0fb8cdb3f176247"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.3/tw-mcp_1.21.3_darwin_amd64.tar.gz"
      sha256 "b6ce173cff3b79464a1768023f5993f2e7cef994e0de7d09385ed53e8ae52364"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
