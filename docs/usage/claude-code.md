# Claude Code (CLI) — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

* Docs: https://docs.anthropic.com/en/docs/claude-code/mcp

## Prerequisites

- Claude Code installed: `npm i -g @anthropic-ai/claude-code`

## Setup

### Option A — Browser Authentication (HTTP + OAuth2)

```bash
# Register the MCP server
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/

# Start Claude Code, then authenticate interactively
claude
# Inside Claude Code run:
/mcp   # Select "Authenticate in Teamwork MCP"
```

### Option B — Bearer Token (HTTP)

```bash
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com \
  --header "Authorization: Bearer <token>"
```

Replace `<token>` with your Bearer token.

> [!TIP]
> See [Get a Bearer Token](teamwork-cli.md#get-a-bearer-token)

## Verify

```bash
# List registered MCP servers
claude mcp list

# Inspect the Teamwork server configuration
claude mcp get teamwork
```

You should see `teamwork` listed with transport `http` and the correct URL.
