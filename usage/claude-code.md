# Claude Code (CLI) — Teamwork.com MCP Setup

← [Back to Usage Guide](../usage.md)

* Docs: https://docs.anthropic.com/en/docs/claude-code/mcp

## Prerequisites

- Claude Code installed: `npm i -g @anthropic-ai/claude-code`
- A Bearer token — see [Getting a Bearer Token](../usage.md#2--get-a-bearer-token)

## Setup

### Option A — Bearer Token (HTTP)

```bash
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com \
  --header "Authorization: Bearer <token>"
```

Replace `<token>` with your Bearer token.

> [!TIP]
> **Get your token:** `npm i @teamwork/get-bearer-token@latest -g && teamwork-get-bearer-token`

### Option B — Browser Authentication (HTTP + OAuth2)

```bash
# Register the MCP server
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/

# Start Claude Code, then authenticate interactively
claude
# Inside Claude Code run:
/mcp   # Select "Authenticate in Teamwork MCP"
```

## Verify

```bash
# List registered MCP servers
claude mcp list

# Inspect the Teamwork server configuration
claude mcp get teamwork
```

You should see `teamwork` listed with transport `http` and the correct URL.
