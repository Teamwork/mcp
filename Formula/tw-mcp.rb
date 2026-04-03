class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.10"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.10/tw-mcp_1.11.10_darwin_arm64.tar.gz"
      sha256 "46893fef99c181fad1ff24493195ec0d90487fe71e11c00177183281a049167d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.10/tw-mcp_1.11.10_darwin_amd64.tar.gz"
      sha256 "353419d32ff360743e825c686cce85cadc017f0e4f29563dfd15d7c9eb9242fd"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
