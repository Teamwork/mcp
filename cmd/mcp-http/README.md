# Teamwork MCP HTTP Server

> HTTP transport server for the Teamwork Model Context Protocol implementation

[![Go](https://img.shields.io/badge/Go-1.26.0-blue.svg)](https://golang.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

## 📖 Overview

The Teamwork MCP HTTP Server provides an HTTP interface for the Model Context
Protocol, enabling secure and scalable communication between Large Language
Models and Teamwork.com. This server implements the MCP specification over HTTP
transport, making it suitable for production deployments and cloud environments.

### 🤖 What is the HTTP Server for?

This HTTP server is designed for:
- **Production deployments**: Scalable, stateless HTTP-based MCP communication
- **Cloud environments**: Easy deployment with load balancing and health checks
- **Multi-client support**: Handle multiple concurrent LLM connections
- **Monitoring and observability**: Built-in metrics, logging, and tracing

## ✨ Features

- **HTTP Transport**: POST-based API implementing the MCP specification
- **SSE Transport**: GET-based Server-Sent Events for streaming MCP communication
- **Health Checks**: Built-in health endpoint
- **Observability**: Comprehensive logging, metrics, and OpenTelemetry tracing
- **Production Ready**: Designed for cloud deployment with proper error handling
- **Stateless**: No server-side session management for horizontal scaling

## 🚀 Quick Start

### 📋 Prerequisites

- Go 1.26 or later
- Valid Teamwork API credentials
- OAuth2 client configuration

### 🏃 Running the Server

```bash
# Basic HTTP server
TW_MCP_SERVER_ADDRESS=:8080 \
  go run cmd/mcp-http/main.go

# With custom configuration
TW_MCP_URL=https://my-mcp.example.com \
  TW_MCP_SERVER_ADDRESS=:8080 \
  TW_MCP_LOG_LEVEL=debug \
  go run cmd/mcp-http/main.go
```

### 🚀 Transport Options

The server supports two transport mechanisms:

#### POST Transport (JSON-RPC)
Traditional HTTP POST requests to `/` with JSON-RPC payloads:
```bash
curl -X POST http://localhost:8080/ \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"..."}}'
```

#### GET Transport (Server-Sent Events)
Streaming SSE connections to `/sse` with GET requests:
```bash
curl -X GET http://localhost:8080/sse \
  -H "Authorization: Bearer <token>" \
  -H "Accept: text/event-stream"
```
This establishes a persistent connection for bidirectional streaming communication via Server-Sent Events.

### 🔗 Extended API Endpoints

Besides the MCP endpoints, the HTTP server provides the following extended API endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | POST | MCP HTTP transport (JSON-RPC over HTTP) |
| `/sse` | GET | MCP SSE transport (Server-Sent Events for streaming) |
| `/api/health` | GET | Health check endpoint |
| `/.well-known/oauth-protected-resource` | GET | OAuth 2.0 protected resource metadata ([RFC 9728](https://datatracker.ietf.org/doc/html/rfc9728)) |
| `/.well-known/openai-apps-challenge` | GET | Origin verification token for the ChatGPT app listing (plain text) |

## ⚙️ Configuration

#### Command-Line Flags

| Flag         | Description                                                     | Default | Example                                              |
| ------------ | --------------------------------------------------------------- | ------- | ---------------------------------------------------- |
| `-toolsets`  | Comma-separated list of sub-toolsets or profile names to enable | `all`   | `project-manager`, `twprojects-tasks,twdesk-tickets` |

### Server Configuration

The server can be configured using the following environment variables:

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TW_MCP_VERSION` | Version of the MCP server | `dev` | `v1.0.0` |
| `TW_MCP_SERVER_ADDRESS` | Server bind address | `:8080` | `:80`, `0.0.0.0:80` |
| `TW_MCP_ENV` | Environment the app is running in | `dev` | `staging`, `production` |
| `TW_MCP_AWS_REGION` | AWS region where the app is running | `us-east-1` | `eu-west-1` |
| `TW_MCP_HAPROXY_URL` | HAProxy instance URL | _(empty)_ | `https://haproxy.example.com` |
| `TW_MCP_URL` | The base URL for the MCP server | `https://mcp.ai.teamwork.com` |
| `TW_MCP_API_URL` | The Teamwork API base URL | `https://teamwork.com` |

### Logging Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TW_MCP_LOG_FORMAT` | Log output format | `text` | `json`, `text` |
| `TW_MCP_LOG_LEVEL` | Logging level | `info` | `debug`, `warn`, `error`, `fatal` |
| `TW_MCP_SENTRY_DSN` | Sentry DSN for error reporting | _(empty)_ | `https://xxx@sentry.io/xxx` |

### OpenTelemetry Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `OTEL_TRACING_ENABLED` | Enable OpenTelemetry tracing | `false` | `true` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP endpoint traces are sent to | `http://localhost:4318` | `http://otel-collector:4318` |
| `OTEL_SERVICE_NAME` | Service name reported on every span | `mcp-server` | `teamwork-mcp` |
| `OTEL_ENV` | Deployment environment | _(uses TW_MCP_ENV)_ | `staging`, `production` |
| `OTEL_VERSION` | Service version | _(uses TW_MCP_VERSION)_ | `v1.0.0` |

## 🔄 Protocol Compatibility

The server negotiates the highest protocol revision the client also supports, so
clients on older spec revisions keep working unchanged. Both handshakes are
served: `server/discover` (SEP-2575) for stateless clients and `initialize` for
everything older. Both bypass authentication, so pre-auth connector setup works
either way.

Two wire-level behaviours changed with the `2026-07-28` revision, and the server
runs stateless, so they apply here:

| Behaviour | Before | Now | Spec basis |
|-----------|--------|-----|------------|
| `Mcp-Session-Id` on responses | A session ID was generated and echoed back | Not sent; an incoming one is ignored | Session IDs are optional; a client that never receives one never sends one |
| `DELETE /` (session termination) | `204 No Content` | `405 Method Not Allowed` | Servers MAY refuse session termination |

If a client turns out to depend on the old behaviour, the SDK ships an escape
hatch that restores it without a code change:

```bash
MCPGODEBUG=allowsessionsinstateless=1
```

Set it on the deployment and the server reads `Mcp-Session-Id`, echoes it back,
and answers `DELETE` with `204` again, exactly as before. It is a temporary
compatibility parameter: the SDK removes it in **v1.9.0**, so treat it as a
stopgap while the client is fixed, not a permanent setting. `MCPGODEBUG` takes a
comma-separated list, and the full set of parameters for the pinned SDK version
is documented at
<https://go.sdk.modelcontextprotocol.io/mcpgodebug/> (also in-tree at
`docs/mcpgodebug.md` of the `modelcontextprotocol/go-sdk` module).

Two further notes for clients:

- The `logging` capability is no longer advertised. It was deprecated by
  SEP-2577 and this server never sent a `notifications/message`, so no client
  loses functionality. Clients on pre-`2026-07-28` revisions can still call
  `logging/setLevel`; on `2026-07-28` the SDK rejects it with `-32601`, as the
  revision removed the method.
- `tools/list` responses carry `cacheScope: "private"`. The tool list is filtered
  per OAuth token scope, so shared intermediaries must not cache it.

## 🧪 Testing

### MCP HTTP CLI

The MCP HTTP CLI is a command-line tool for interacting with the MCP HTTP
server. It provides a simple interface for testing API endpoints and performing
common tasks.

For more information check [here](../mcp-http-cli/README.md).

#### 🔍 MCP Inspector

For debugging purposes, use the [MCP Inspector tool](https://github.com/modelcontextprotocol/inspector):

```bash
NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem npx @modelcontextprotocol/inspector node build/index.js
```

> [!IMPORTANT]
> **Note**: The `NODE_EXTRA_CA_CERTS` environment variable is required when
> using OAuth2 authentication with the Let's Encrypt certification authority.
> Download the certificate [here](https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem).

## 🔍 Monitoring

The HTTP server provides comprehensive monitoring capabilities:

- **Health Checks**: `/health` and `/ready` endpoints for load balancer integration
- **Structured Logging**: JSON or text format with configurable log levels
- **OpenTelemetry**: Distributed tracing and performance monitoring
- **Metrics**: Built-in metrics for request rates, latencies, and errors