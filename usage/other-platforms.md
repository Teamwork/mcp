# Other Platforms — Teamwork.com MCP Setup

← [Back to Usage Guide](../usage.md)

Use the public hosted HTTP endpoint for any platform that supports MCP over HTTP or generic JSON-RPC.

**Endpoint:** `https://mcp.ai.teamwork.com`
**Auth header:** `Authorization: Bearer <token>`

> [!TIP]
> **Get your token:** `npm i @teamwork/get-bearer-token@latest -g && teamwork-get-bearer-token`

## n8n

1. Add an **HTTP Request** node (or an MCP-aware node if available in your version).
2. Set the URL to `https://mcp.ai.teamwork.com`.
3. Add the header `Authorization: Bearer <token>`.
4. Use the MCP JSON-RPC payload format to call tools.

## Appmixer

1. Create a new integration and select the **HTTP** connector.
2. Set the base URL to `https://mcp.ai.teamwork.com`.
3. Add the `Authorization: Bearer <token>` header to the connector authentication settings.

## Custom / Programmatic

Any HTTP client can call the MCP server directly:

```bash
curl -s https://mcp.ai.teamwork.com \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Refer to the [MCP specification](https://modelcontextprotocol.io/specification) for the full JSON-RPC API.
