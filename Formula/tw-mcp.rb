class TwMcp < Formula
  desc "Teamwork.com MCP server"
  homepage "https://github.com/Teamwork/mcp"
  version "1.50.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Teamwork/mcp/releases/download/v1.50.0/tw-mcp_1.50.0_darwin_arm64.tar.gz"
      sha256 "394e91859b25b652aadbda0708d4d570ccd6de1847174167c11d2ea8cdc27c22"
    else
      url "https://github.com/Teamwork/mcp/releases/download/v1.50.0/tw-mcp_1.50.0_darwin_amd64.tar.gz"
      sha256 "912ee4b23d2f64df31f76688ec2b91d903774700b345afe1a8a36ab823293932"
    end
  end

  def install
    bin.install "tw-mcp"
  end

  test do
    assert_match "Usage", shell_output("#{bin}/tw-mcp -h", 2)
  end
end
