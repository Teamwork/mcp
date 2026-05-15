class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.20.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.1/tw-mcp_1.20.1_darwin_arm64.tar.gz"
      sha256 "68608f89e6da9fdc4abc8f25080b4fc0fb0e7774ecf611bf9aff3998b8e9131a"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.20.1/tw-mcp_1.20.1_darwin_amd64.tar.gz"
      sha256 "c8b6c0abcd09f566903250749697043865aa85ff22eb690ec70f9d41d0432c06"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
