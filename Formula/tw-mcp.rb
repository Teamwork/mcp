class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.0/tw-mcp_1.21.0_darwin_arm64.tar.gz"
      sha256 "827e25b2823119d89a21db4b8318e551e4a585894b0e4fff992c52fd68be738d"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.0/tw-mcp_1.21.0_darwin_amd64.tar.gz"
      sha256 "dd3483567414544b978bfba96d46165fa9db8e5568d19a8b610caeb403ed8a8e"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
