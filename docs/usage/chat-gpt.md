# ChatGPT — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

* Video walkthrough: https://www.youtube.com/watch?v=OiWVXRrcHYs

## Prerequisites

- A ChatGPT account with access to the **ChatGPT Workspace** (Team or Enterprise plan)
- Developer mode enabled in your workspace settings

## Setup

### Step 1 — Create the MCP connection

1. Open **ChatGPT Workspace Settings** → **Apps**.
2. Click **Create** and enable **developer mode** when prompted.
3. Enter the MCP Server URL:
   ```
   https://mcp.ai.teamwork.com
   ```
4. Select **OAuth** as the authentication method.
5. Acknowledge the warning after reading the [MCP risks](https://developers.openai.com/api/docs/mcp#risks-and-safety).

### Step 2 — Authenticate with Teamwork.com

You will be redirected to Teamwork.com to log in and grant ChatGPT access to
your account. After approving, you will be sent back to ChatGPT automatically.

### Step 3 — Start chatting

Back in the chat, select the new **Teamwork.com** MCP connection from the
dropdown and start chatting!

## Verify

Once connected, you can ask ChatGPT questions like:

- *"What tasks are assigned to me this week?"*
- *"Show me all open tickets in my Teamwork Desk inbox."*
- *"Create a new task in project X."*

If the Teamwork.com tools are available, ChatGPT will use them to answer.

## Troubleshooting

> [!TIP]
>
> If the connection fails during OAuth, try opening the **Apps** settings page
> again and removing the existing Teamwork.com entry before re-creating it.
> ChatGPT may have cached a stale token.
