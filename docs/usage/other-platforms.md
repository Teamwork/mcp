# Other Platforms — Teamwork.com MCP Setup

← [Back to Usage Guide](README.md)

Use the public hosted HTTP endpoint for any platform that supports MCP over HTTP or generic JSON-RPC.

**Endpoint:** `https://mcp.ai.teamwork.com`
**Auth header:** `Authorization: Bearer <token>`

> [!TIP]
> See [Get a Bearer Token](teamwork-cli.md#get-a-bearer-token)

## n8n

1. Add an **HTTP Request** node (or an MCP-aware node if available in your version).
2. Set the URL to `https://mcp.ai.teamwork.com`.
3. Add the header `Authorization: Bearer <token>`.
4. Use the MCP JSON-RPC payload format to call tools.

## Appmixer

1. Create a new integration and select the **HTTP** connector.
2. Set the base URL to `https://mcp.ai.teamwork.com`.
3. Add the `Authorization: Bearer <token>` header to the connector authentication settings.

### LibreChat

1. Use the SSE endpoint: `https://mcp.ai.teamwork.com/sse`.
2. Set the `SSE` transport method.
3. Select the OAuth authentication
4. Change your Teamwork Developer App to use the correct redirect URL (`http://localhost:3080/api/mcp/teamwork/oauth/callback` by default).
5. Fill the client ID and secret from your Teamwork Developer App.
6. Populate the Authorization server fields:
  - Authorization URL: `https://www.teamwork.com/launchpad/login`
  - Token URL: `https://www.teamwork.com/launchpad/v1/token.json`
7. Set the same scopes from the Teamwork Developer App (`projects`, `desk` or `projects,desk`).

## Custom / Programmatic

Any HTTP client can call the MCP server directly:

```bash
curl -s https://mcp.ai.teamwork.com \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Refer to the [MCP specification](https://modelcontextprotocol.io/specification) for the full JSON-RPC API.
