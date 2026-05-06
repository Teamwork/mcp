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

### Option C — Profile Endpoint (HTTP)

Use a profile URL to load only the tools relevant to your role. This keeps the
tool list smaller and reduces noise for the model.

```bash
# Project managers — projects, tasks, people, content
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/project-manager \
  --header "Authorization: Bearer <token>"

# Support agents — help desk tickets and customers
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/support \
  --header "Authorization: Bearer <token>"

# Analysts — projects, tasks, time tracking, tickets, and more
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/analyst \
  --header "Authorization: Bearer <token>"

# Knowledge managers — spaces, pages, and content
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/knowledge-manager \
  --header "Authorization: Bearer <token>"

# Ops — all available tools
claude mcp add --transport http teamwork https://mcp.ai.teamwork.com/ops \
  --header "Authorization: Bearer <token>"
```

## Verify

```bash
# List registered MCP servers
claude mcp list

# Inspect the Teamwork server configuration
claude mcp get teamwork
```

You should see `teamwork` listed with transport `http` and the correct URL.
