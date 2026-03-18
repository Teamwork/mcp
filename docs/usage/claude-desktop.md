# Claude Desktop — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

* Video walkthrough: https://www.youtube.com/watch?v=BHPSuAYEVYU

<img width="764" height="428" alt="Claude Desktop with Teamwork MCP" src="https://github.com/user-attachments/assets/de6bb3c2-dfc5-4f6c-b497-6ea22ea01636" />

## Prerequisites

- Teamwork CLI installed and in your PATH — see the [Teamwork CLI setup guide](teamwork-cli.md)
- Claude Desktop installed: https://claude.ai/download

## Setup

Open or create the Claude Desktop config file:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

For more details on the config file location, see the MCP quickstart:
https://modelcontextprotocol.io/quickstart/user

## Configuration

> [!TIP]
> See [Get a Bearer Token](teamwork-cli.md#get-a-bearer-token)

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

Restart Claude Desktop. You should see the Teamwork.com MCP tools listed in the
tool selector (hammer icon) within a chat.
