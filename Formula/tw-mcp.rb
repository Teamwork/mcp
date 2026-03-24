class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.11.6"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.6/tw-mcp_1.11.6_darwin_arm64.tar.gz"
      sha256 "6f418b023ef3d9d680d78b69825475a8c5646e7f919074ea9b6d4c83803ac52d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.11.6/tw-mcp_1.11.6_darwin_amd64.tar.gz"
      sha256 "28cc483c77d0bdd8092460caec0ed98ebd2b700a999001d01c54e33fceb2348c"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
