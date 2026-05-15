# Teamwork.com MCP — Usage Guide

This guide helps you connect AI tools to your Teamwork.com site via MCP.

**Public hosted endpoint (HTTP):** `https://mcp.ai.teamwork.com`

**Self-hosted / local binary (STDIO):** see the [Teamwork CLI setup
guide](/docs/usage/teamwork-cli.md)

---

## Profile Endpoints

The main endpoint loads all available tools. If you only need a subset, use one
of the profile endpoints to keep the tool list focused:

| Profile | Endpoint | Tools included |
|---|---|---|
| **Project Manager** | `https://mcp.ai.teamwork.com/project-manager` | Projects, tasks, people, content |
| **Support** | `https://mcp.ai.teamwork.com/support` | Help desk tickets and customers |
| **Desk** | `https://mcp.ai.teamwork.com/desk` | Alias for **Support** — help desk tickets and customers |
| **Analyst** | `https://mcp.ai.teamwork.com/analyst` | Projects, tasks, people, time tracking, content, tickets, customers, and desk admin |
| **Knowledge Manager** | `https://mcp.ai.teamwork.com/knowledge-manager` | Spaces, pages, and content |
| **Ops** | `https://mcp.ai.teamwork.com/ops` | All available tools |

Profile endpoints accept the same authentication methods as the main endpoint.

---

## Prerequisites

- A Teamwork.com account with permission to create an API key
- *(Optional)* Admin access to enable the AI / MCP feature on your site

### Enable MCP for Your Site

Ask an account administrator to enable MCP under **Settings → AI**.

<img width="2876" height="1296" alt="Teamwork Settings – AI/MCP toggle" src="https://github.com/user-attachments/assets/f76deec2-27fb-494d-9b0a-b0a8d302db3d" />

---

## Client Setup Guides

| Client                           | Transport     | Guide                                                |
|----------------------------------|---------------|------------------------------------------------------|
| **Teamwork CLI**                 | STDIO         | [teamwork-cli.md](/docs/usage/teamwork-cli.md)       |
| **ChatGPT**                      | HTTP          | [chat-gpt.md](/docs/usage/chat-gpt.md)             |
| **Claude Desktop**               | STDIO         | [claude-desktop.md](/docs/usage/claude-desktop.md)   |
| **Claude Code (CLI)**            | HTTP          | [claude-code.md](/docs/usage/claude-code.md)         |
| **VSCode — GitHub Copilot Chat** | HTTP or STDIO | [vscode-copilot.md](/docs/usage/vscode-copilot.md)   |
| **Gemini CLI**                   | HTTP          | [gemini-cli.md](/docs/usage/gemini-cli.md)           |
| **n8n, Appmixer, custom**        | HTTP          | [other-platforms.md](/docs/usage/other-platforms.md) |

---

## Tool Parameters

### `verbose` flag (list tools)

Most `list_*` tools accept an optional `verbose` boolean (default `true`):

- **`verbose=true`** — full entity details, including any sideloaded related
  data (custom fields, project categories, etc.).
- **`verbose=false`** — only a minimal subset of fields (typically `id` and a
  name/title) is returned to reduce response size. Useful when scanning many
  results to pick an id before fetching details on a specific one via the
  corresponding `get_*` tool.

Structured content (`structuredContent`) is returned in both modes. The
`outputSchema` published for `list_*` tools marks every property as
optional, so sparse responses returned when `verbose=false` still validate
against the schema. Single-entity `get_*` tools keep a strict schema with
required fields intact.

> [!NOTE]
>
> Because `list_*` output schemas mark all fields as optional (to support
> `verbose=false`), they are **incompatible with OpenAI's strict structured-
> output mode**, which requires every property to be `required` and forbids
> `additionalProperties`. Clients targeting `list_*` tools must run with
> strict mode disabled. `get_*` tools remain strict-compatible.
