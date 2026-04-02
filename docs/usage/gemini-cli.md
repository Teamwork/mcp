# Gemini CLI — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

<img width="732" height="558" alt="Gemini CLI with Teamwork MCP" src="https://github.com/user-attachments/assets/b26d2fe0-2d88-4bcc-beb5-3dab5cb575b0" />

* Install Gemini CLI: https://github.com/google-gemini/gemini-cli?tab=readme-ov-file#quickstart

## Prerequisites

- Gemini CLI installed

## Setup

Edit `$HOME/.gemini/settings.json` and add the `mcpServers` block:

```json
{
  "mcpServers": {
    "Teamwork.com": {
      "url": "https://mcp.ai.teamwork.com",
      "type": "http",
      "oauth": {
        "scopes": ["projects","desk"]
      },
      "trust": false,
      "timeout": 5000,
      "description": "Teamwork.com MCP server"
    }
  }
}
```

> [!NOTE]
> This configuration is for Gemini CLI v0.36.0. If you're using a different
> version, please refer to the corresponding documentation for any changes in
> configuration options.

More configuration options can be found [here](https://github.com/google-gemini/gemini-cli/blob/v0.36.0/packages/core/src/code_assist/types.ts#L369-L389).

## Notes

- `"trust": false` causes Gemini CLI to prompt you for confirmation before
  executing any action against Teamwork.com. This is recommended to prevent
  accidental modifications.

- Increase `timeout` (milliseconds) if you experience timeouts on slow networks.
