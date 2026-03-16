# Teamwork.com MCP — Usage Guide

This guide helps you connect AI tools to your Teamwork.com site via MCP.

**Public hosted endpoint (HTTP):** `https://mcp.ai.teamwork.com`

**Self-hosted / local binary (STDIO):** see the [`tw-mcp` setup guide](usage/tw-mcp.md)

---

## Prerequisites

- A Teamwork.com account with permission to create an API key
- *(Optional)* Admin access to enable the AI / MCP feature on your site

### Enable MCP for Your Site

Ask an account administrator to enable MCP under **Settings → AI**.

<img width="2876" height="1296" alt="Teamwork Settings – AI/MCP toggle" src="https://github.com/user-attachments/assets/f76deec2-27fb-494d-9b0a-b0a8d302db3d" />

---

## Client Setup Guides

| Client | Transport | Guide |
|--------|-----------|-------|
| **Teamwork CLI** | STDIO | [usage/tw-mcp.md](usage/tw-mcp.md) |
| **Claude Desktop** | STDIO | [usage/claude-desktop.md](usage/claude-desktop.md) |
| **Claude Code (CLI)** | HTTP | [usage/claude-code.md](usage/claude-code.md) |
| **VSCode — GitHub Copilot Chat** | HTTP or STDIO | [usage/vscode-copilot.md](usage/vscode-copilot.md) |
| **Gemini CLI** | HTTP | [usage/gemini-cli.md](usage/gemini-cli.md) |
| **n8n, Appmixer, custom** | HTTP | [usage/other-platforms.md](usage/other-platforms.md) |
