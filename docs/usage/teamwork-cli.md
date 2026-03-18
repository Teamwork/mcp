# Teamwork CLI — Setup

← [Back to Usage Guide](README.md)

The Teamwork CLI (`tw-mcp`) is the self-hosted STDIO binary for desktop MCP
clients (Claude Desktop, VSCode Copilot, etc.). It runs locally — no exposed
network port needed.

## Get a Bearer Token

```sh
# Install (or update) the helper
npm i @teamwork/get-bearer-token@latest -g

# Run it and follow the prompts
teamwork-get-bearer-token
```

Copy the token it outputs — you will use it as `<token>` (or
`TW_MCP_BEARER_TOKEN`) in your client config.

Alternatively follow the manual steps at:
https://apidocs.teamwork.com/guides/teamwork/app-login-flow

## Install

### macOS — Homebrew (recommended)

```sh
brew tap teamwork/mcp https://github.com/teamwork/mcp
brew install teamwork/mcp/tw-mcp

# Verify
tw-mcp -h

# Upgrade later
brew update && brew upgrade teamwork/mcp/tw-mcp
```

### All platforms — download binary

Download the latest release for your OS/arch from:
https://github.com/Teamwork/mcp/releases/latest

Then move it into your PATH:

```sh
# Example for macOS/Linux
mv tw-mcp /usr/local/bin/tw-mcp
chmod +x /usr/local/bin/tw-mcp

# Verify
tw-mcp -h
```

> [!IMPORTANT]
> **macOS security note:** If macOS blocks the binary, open **System Settings →
> Privacy & Security** and click **Allow Anyway**.

## Usage

`tw-mcp` reads its configuration from environment variables:

| Variable | Description |
|----------|-------------|
| `TW_MCP_BEARER_TOKEN` | Bearer token for authentication (required) |

Your MCP client (Claude Desktop, VSCode, etc.) is responsible for spawning
`tw-mcp` and passing the environment variables — see the [client
guides](README.md#client-setup-guides) for exact config snippets.
