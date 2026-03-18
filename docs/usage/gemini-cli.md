# Gemini CLI — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

<img width="732" height="558" alt="Gemini CLI with Teamwork MCP" src="https://github.com/user-attachments/assets/b26d2fe0-2d88-4bcc-beb5-3dab5cb575b0" />

* Install Gemini CLI: https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#quickstart

## Prerequisites

- Gemini CLI installed
- A Bearer token — see [Get a Bearer Token](teamwork-cli.md#get-a-bearer-token)

## Setup

Edit `$HOME/.gemini/settings.json` and add the `mcpServers` block:

```json
{
  "selectedAuthType": "oauth-personal",
  "mcpServers": {
    "Teamwork.com": {
      "httpUrl": "https://mcp.ai.teamwork.com",
      "headers": { "Authorization": "Bearer <token>" },
      "trust": false,
      "timeout": 5000
    }
  }
}
```

Replace `<token>` with your Bearer token.

> [!TIP]
> **Get your token:** `npm i @teamwork/get-bearer-token@latest -g && teamwork-get-bearer-token`

## Notes

- `"trust": false` causes Gemini CLI to prompt you for confirmation before executing any action against Teamwork.com. This is recommended to prevent accidental modifications.
- Increase `timeout` (milliseconds) if you experience timeouts on slow networks.
