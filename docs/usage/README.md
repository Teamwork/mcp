# Teamwork.com MCP — Usage Guide

This guide helps you connect AI tools to your Teamwork.com site via MCP.

**Public hosted endpoint (HTTP):** `https://mcp.ai.teamwork.com`

**Self-hosted / local binary (STDIO):** see the [Teamwork CLI setup guide](usage/teamwork-cli.md)

---

## Prerequisites

- A Teamwork.com account with permission to create an API key
- *(Optional)* Admin access to enable the AI / MCP feature on your site

### Enable MCP for Your Site

Ask an account administrator to enable MCP under **Settings → AI**.

<img width="2876" height="1296" alt="Teamwork Settings – AI/MCP toggle" src="https://github.com/user-attachments/assets/f76deec2-27fb-494d-9b0a-b0a8d302db3d" />

---

## Client Setup Guides

| Client                           | Transport     | Guide                                    |
|----------------------------------|---------------|------------------------------------------|
| **Teamwork CLI**                 | STDIO         | [teamwork-cli.md](teamwork-cli.md)       |
| **Claude Desktop**               | STDIO         | [claude-desktop.md](claude-desktop.md)   |
| **Claude Code (CLI)**            | HTTP          | [claude-code.md](claude-code.md)         |
| **VSCode — GitHub Copilot Chat** | HTTP or STDIO | [vscode-copilot.md](vscode-copilot.md)   |
| **Gemini CLI**                   | HTTP          | [gemini-cli.md](gemini-cli.md)           |
| **n8n, Appmixer, custom**        | HTTP          | [other-platforms.md](other-platforms.md) |
