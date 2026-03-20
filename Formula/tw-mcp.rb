class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.5"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.5/tw-mcp_1.11.5_darwin_arm64.tar.gz"
      sha256 "2c6b99e9f6c5e836dbf40a3cbf149e1ae6e0efeb1055a9a0e20cc9abfad4b792"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.5/tw-mcp_1.11.5_darwin_amd64.tar.gz"
      sha256 "4943d7223d637fef67f093d611b16c07301c804fa97a00d3dac0ddb1150a259a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
