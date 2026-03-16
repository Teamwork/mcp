# Teamwork.com MCP — Usage Guide

This guide helps you connect AI tools to your Teamwork.com site via MCP.

**Public hosted endpoint (HTTP):** `https://mcp.ai.teamwork.com`

**Self-hosted / local binary (STDIO):** see the [Teamwork CLI setup guide](usage/teamwork-cli.md)

---

## Prerequisites

- A Teamwork.com account with permission to create an API key
- *(Optional)* Admin access to enable the AI / MCP feature on your site

## Get a Bearer Token

Use the interactive helper:

```sh
# Install (or update) the helper
npm i @teamwork/get-bearer-token@latest -g

# Run it and follow the prompts
teamwork-get-bearer-token
```

Copy the token it outputs — you will use it as `<token>` (or `TW_MCP_BEARER_TOKEN`) in your client config.

Alternatively, follow the manual steps at:
https://apidocs.teamwork.com/guides/teamwork/app-login-flow

### Enable MCP for Your Site

Ask an account administrator to enable MCP under **Settings → AI**.

<img width="2876" height="1296" alt="Teamwork Settings – AI/MCP toggle" src="https://github.com/user-attachments/assets/f76deec2-27fb-494d-9b0a-b0a8d302db3d" />

---

## Client Setup Guides

| Client | Transport | Guide |
|--------|-----------|-------|
| **Teamwork CLI** | STDIO | [usage/teamwork-cli.md](usage/teamwork-cli.md) |
| **Claude Desktop** | STDIO | [usage/claude-desktop.md](usage/claude-desktop.md) |
| **Claude Code (CLI)** | HTTP | [usage/claude-code.md](usage/claude-code.md) |
| **VSCode — GitHub Copilot Chat** | HTTP or STDIO | [usage/vscode-copilot.md](usage/vscode-copilot.md) |
| **Gemini CLI** | HTTP | [usage/gemini-cli.md](usage/gemini-cli.md) |
| **n8n, Appmixer, custom** | HTTP | [usage/other-platforms.md](usage/other-platforms.md) |
