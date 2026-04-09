class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.12.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.12.1/tw-mcp_1.12.1_darwin_arm64.tar.gz"
      sha256 "7d9f10c8a553e7fc7ce95aa3986b20af543b19414d97d565641c357ba7708e31"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.12.1/tw-mcp_1.12.1_darwin_amd64.tar.gz"
      sha256 "92127a72fc5e82a379776b2c9512fbba51fa5d7c86ac84d685eb77ab23cd3004"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
