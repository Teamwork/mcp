class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.16.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.16.0/tw-mcp_1.16.0_darwin_arm64.tar.gz"
      sha256 "fa4a0b911fe64c627b1b1c99d50bc7c46c0a61a3a23b0b7fd520756a23f8e9bd"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.16.0/tw-mcp_1.16.0_darwin_amd64.tar.gz"
      sha256 "7925fcf8407db20512c88bf9a268f90107a54bd79e89a2e4b16344e1d37a6650"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
