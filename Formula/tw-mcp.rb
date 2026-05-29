class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.21.1"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.1/tw-mcp_1.21.1_darwin_arm64.tar.gz"
      sha256 "89dd2ec948e320bf4e6eda7bb45dd6f6c11ab2b33225fa1dd6fc66ad4b929c20"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.21.1/tw-mcp_1.21.1_darwin_amd64.tar.gz"
      sha256 "6cfac33038ecd95f7f5eb005e59f0cc0609527fd382259d270ddcfb0a30b0d4d"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
