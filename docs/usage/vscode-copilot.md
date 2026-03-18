# VSCode (GitHub Copilot Chat) — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

<img width="753" height="839" alt="VSCode Copilot Chat with Teamwork MCP" src="https://github.com/user-attachments/assets/61204ca7-c904-4cf6-aa3a-059b8c96fa48" />

* Reference config file: https://github.com/Teamwork/mcp/blob/main/.vscode/mcp.json
* Docs: https://code.visualstudio.com/docs/copilot/chat/mcp-servers#_add-an-mcp-server

## Prerequisites

- VSCode with the **GitHub Copilot Chat** extension installed
- *(STDIO only)* Teamwork CLI in your PATH — see the [Teamwork CLI setup guide](teamwork-cli.md)

## Setup

Add an entry to your `.vscode/mcp.json` (workspace) or the user-level MCP
settings. Use either STDIO or HTTP depending on your preference.

### Option A — Hosted HTTP

```json
{
  "servers": {
    "Teamwork.com": {
      "type": "http",
      "url": "https://mcp.ai.teamwork.com"
    }
  }
}
```

### Option B — Local binary (STDIO)

```json
{
  "servers": {
    "Teamwork.com": {
      "type": "stdio",
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
> See [Get a Bearer Token](teamwork-cli.md#get-a-bearer-token)

## Troubleshooting

> [!TIP]
>
> When using HTTP, if you get a **"Teamwork account not found"** error during the
> OAuth2 authentication process, VSCode may have cached old credentials.
> Clear them by opening the Command Palette (`Cmd+Shift+P` / `Ctrl+Shift+P`)
> and running **"Authentication: Remove Dynamic Authentication Providers"**.
