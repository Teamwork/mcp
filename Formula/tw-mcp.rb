class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.27.4"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.4/tw-mcp_1.27.4_darwin_arm64.tar.gz"
      sha256 "f66e994331e1d24e8e7224b4a7a6ffd8d53fed71d8949dfdab60721cb99e0daf"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.27.4/tw-mcp_1.27.4_darwin_amd64.tar.gz"
      sha256 "babee54e4b9e59b6dd1437d077ddba68e238a3dad457c082b3a2dc9f82e60b4a"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
