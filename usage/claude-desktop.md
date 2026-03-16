# Claude Desktop — Teamwork.com MCP Setup

← [Back to Usage Guide](../usage.md)

* Video walkthrough: https://www.youtube.com/watch?v=BHPSuAYEVYU

<img width="764" height="428" alt="Claude Desktop with Teamwork MCP" src="https://github.com/user-attachments/assets/de6bb3c2-dfc5-4f6c-b497-6ea22ea01636" />

## Prerequisites

- `tw-mcp` binary installed and in your PATH — see [Getting a Bearer Token](../usage.md#2--get-a-bearer-token) and [Choose a Connection Mode](../usage.md#4--choose-a-connection-mode)
- Claude Desktop installed: https://claude.ai/download

## Setup

1. Download the latest `tw-mcp` release: https://github.com/Teamwork/mcp/releases/latest
2. Rename/move the binary into your PATH as `tw-mcp` (e.g. `/usr/local/bin/tw-mcp`)
3. **(macOS)** Approve it in **System Settings → Privacy & Security** if macOS blocks it.
4. Open or create the Claude Desktop config file:
   - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

For more details on the config file location, see the MCP quickstart:
https://modelcontextprotocol.io/quickstart/user

## Configuration

### Option A — Local binary (STDIO, recommended)

```json
{
  "mcpServers": {
    "Teamwork.com": {
      "command": "tw-mcp",
      "args": [],
      "env": {
        "TW_MCP_BEARER_TOKEN": "<token>"
      }
    }
  }
}
```

Replace `<token>` with your Bearer token.

> [!TIP]
> **Get your token:** `npm i @teamwork/get-bearer-token@latest -g && teamwork-get-bearer-token`

### Option B — Docker (STDIO)

```json
{
  "mcpServers": {
    "Teamwork.com": {
      "command": "docker",
      "args": [
        "run",
        "-i",
        "--rm",
        "-e",
        "TW_MCP_BEARER_TOKEN",
        "ghcr.io/teamwork/mcp:latest"
      ],
      "env": {
        "TW_MCP_BEARER_TOKEN": "<token>"
      }
    }
  }
}
```

## Verify

Restart Claude Desktop. You should see the Teamwork.com MCP tools listed in the tool selector (hammer icon) within a chat.
