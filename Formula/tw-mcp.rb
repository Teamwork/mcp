class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.39.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.0/tw-mcp_1.39.0_darwin_arm64.tar.gz"
      sha256 "52eca80d3947e9da00f9efc6ffd5a863a00b24cf0f2eaa897bd2cade95c260dc"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.39.0/tw-mcp_1.39.0_darwin_amd64.tar.gz"
      sha256 "068fc49bfbbe22b7ac4a0df787d1424d9a311391c692ec0a2a846378c736775c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
