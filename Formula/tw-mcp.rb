class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.27.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.0/tw-mcp_1.27.0_darwin_arm64.tar.gz"
      sha256 "861b3b758481d379440df36120754703049b14bd9a8f4339f5540fe4957e81dd"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.0/tw-mcp_1.27.0_darwin_amd64.tar.gz"
      sha256 "c2042e88425653a437fd38292d54a988b51c08f66b1d9b19a88826cdca0275c7"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
