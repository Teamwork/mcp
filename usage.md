## 🧭 Teamwork.com MCP Usage Guide

Public hosted MCP endpoint (HTTP):
https://mcp.ai.teamwork.com

Self‑hosted / local binary (STDIO): use the [`tw-mcp`](https://github.com/Teamwork/mcp/releases/latest)
release binary.

This guide helps a Teamwork.com user connect AI tools (Claude, VSCode Copilot
Chat, Gemini, etc.) to Teamwork.com via MCP.

---
## 1. 📋 Prerequisites
- A Teamwork.com account (role with permission to create an API key)
- An API key (Bearer token)
- (Optional) Admin access to enable the AI / MCP feature

## 2. 🔐 Get a Bearer Token

Follow the official steps:
https://apidocs.teamwork.com/guides/teamwork/app-login-flow

Helper (interactive) tool:
```
npx @teamwork/get-bearer-token
```

It outputs a token you will paste into your MCP client configuration as
`<token>`.

## 3. ⚙️ Enable MCP for Your Site

Ask an account administrator to enable MCP under Settings → AI.

<img width="2876" height="1296" alt="image" src="https://github.com/user-attachments/assets/f76deec2-27fb-494d-9b0a-b0a8d302db3d" />

## 4. 🔌 Choose a Connection Mode

Two main ways to connect:

1. Hosted HTTP (simplest) – point the client to https://mcp.ai.teamwork.com and
   send the `Authorization: Bearer <token>` header.

2. Local STDIO – download and run the `tw-mcp` binary; the client spawns it and
   passes environment variables (no exposed network port needed).

Use STDIO for desktop apps (Claude Desktop, local Copilot) and HTTP for cloud
workflows or clients that only speak HTTP.

## 5. 🖥️ Configure Common Clients

### 💬 Claude Desktop (STDIO)

<img width="764" height="428" alt="image" src="https://github.com/user-attachments/assets/de6bb3c2-dfc5-4f6c-b497-6ea22ea01636" />

1. Download the latest release: https://github.com/Teamwork/mcp/releases/latest
2. Rename/move the binary into your PATH as `tw-mcp` (e.g. `/usr/local/bin/tw-mcp`)
3. (macOS) Approve it in Security & Privacy if blocked.
4. Add to Claude Desktop config (see MCP quickstart): https://modelcontextprotocol.io/quickstart/user

Example snippet:
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

Replace `<installation>` with your site domain (e.g. `mycompany.teamwork.com`).
Alternatively, you can use a Docker image:

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

### 🧩 VSCode (GitHub Copilot Chat MCP)

<img width="753" height="839" alt="image" src="https://github.com/user-attachments/assets/61204ca7-c904-4cf6-aa3a-059b8c96fa48" />

* Reference config file: https://github.com/Teamwork/mcp/blob/main/.vscode/mcp.json
* Docs: https://code.visualstudio.com/docs/copilot/chat/mcp-servers#_add-an-mcp-server

Use either `command` (STDIO) or `httpUrl` (hosted) style.

> [!TIP]
>
> When using HTTP, in case you get a "Teamwork account not found" error during
> the OAuth2 authentication process, ensure VSCode didn't cache old credentials.
> You can clear cached credentials by going to Cmd+Shift+P (or Ctrl+Shift+P on
> Windows/Linux) and selecting "Authentication: Remove Dynamic Authentication
> Providers".

### 🌐 Gemini CLI (HTTP)

<img width="732" height="558" alt="image" src="https://github.com/user-attachments/assets/b26d2fe0-2d88-4bcc-beb5-3dab5cb575b0" />

* Install: https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#quickstart
* Edit `$HOME/.gemini/settings.json`:

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

The `"trust": false` setting will make Gemini CLI ask before executing any
action against the Teamwork.com MCP server. This is recommended to avoid
accidental modifications.

### 🛠️ Other Platforms (n8n, Appmixer, custom)

Use the hosted HTTP endpoint. Provide a generic MCP client (if supported) or
wrap calls to the MCP JSON-RPC over HTTP endpoint with the `Authorization`
header.
