# mcp

Teamwork.com MCP server.

Model Context Protocol (MCP) is an open protocol that standardises how
applications provide context to LLMs. That means that we are able to build a
server that describes all the actions allowed in Teamwork (tools) in a way that
the LLM can understand and execute (via agents).

### 🤓 Debugging

Execute the MCP server optionally providing the base URL of the Teamwork
application:

```bash
DEVENV_INSTALLATION=example.dev.stg.teamworkops.com SERVER_ADDRESS=:8012 go run cmd/mcp/main.go
```

For debugging purposes, you can run the [MCP Inspector tool](https://github.com/modelcontextprotocol/inspector):

```bash
NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem npx @modelcontextprotocol/inspector node build/index.js
```

The `NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem` is required when relying
on OAuth2 authentication in the [CDE environment](https://github.com/Teamwork/dev-env-devspace).
You can download the certificate [here](https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem).