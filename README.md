# Teamwork MCP Server

> Model Context Protocol server for Teamwork.com integration with Large Language
> Models

[![Go](https://img.shields.io/badge/Go-1.24.2-blue.svg)](https://golang.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

## 📖 Overview

This MCP (Model Context Protocol) server enables seamless integration between
Large Language Models and Teamwork.com. It provides a standardized interface for
LLMs to interact with Teamwork projects, allowing AI agents to perform various
project management operations.

### 🤖 What is MCP?

Model Context Protocol (MCP) is an open protocol that standardizes how
applications provide context to LLMs. This server describes all the actions
available in Teamwork (tools) in a way that LLMs can understand and execute
through AI agents.

## ✨ Features

- **Secure Authentication**: OAuth2 integration with Teamwork
- **Tool Framework**: Extensible toolset architecture for adding new capabilities
- **HTTP Streaming or STDIO**: Efficient real-time communication with LLMs

## 🚀 Quick Start

### 📋 Prerequisites

- Go 1.24 or later
- Valid Teamwork API credentials

### ⚙️ Environment Variables

The server can be configured using the following environment variables:

#### Server Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TW_MCP_VERSION` | Version of the MCP server | `dev` | `v1.0.0` |
| `TW_MCP_SERVER_ADDRESS` | Server bind address (HTTP mode only) | `localhost:8012` | `:8080`, `0.0.0.0:8012` |
| `TW_MCP_ENV` | Environment the app is running in | `dev` | `staging`, `production` |
| `TW_MCP_AWS_REGION` | AWS region where the app is running | `us-east-1` | `eu-west-1` |
| `TW_MCP_HAPROXY_URL` | HAProxy instance URL (HTTP mode) | _(empty)_ | `https://haproxy.example.com` |
| `TW_MCP_URL` | The base URL for the MCP server | `https://mcp.example.dev.stg.teamworkops.com` |
| `TW_MCP_API_URL` | The Teamwork API base URL | `https://example.dev.stg.teamworkops.com` |

#### Authentication Variables
| Variable | Description | Example |
|----------|-------------|---------|
| `TW_MCP_BEARER_TOKEN` | Bearer token for Teamwork API (STDIO mode) | `your-bearer-token` |

#### Logging Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TW_MCP_LOG_FORMAT` | Log output format | `text` | `json`, `text` |
| `TW_MCP_LOG_LEVEL` | Logging level | `info` | `debug`, `warn`, `error`, `fatal` |
| `TW_MCP_SENTRY_DSN` | Sentry DSN for error reporting | _(empty)_ | `https://xxx@sentry.io/xxx` |

#### Datadog APM Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `DD_APM_TRACING_ENABLED` | Enable Datadog APM tracing | `false` | `true` |
| `DD_SERVICE` | Service name for Datadog APM | `mcp-server` | `teamwork-mcp` |
| `DD_AGENT_HOST` | Datadog Agent host | `localhost` | `datadog-agent` |
| `DD_TRACE_AGENT_PORT` | Datadog trace agent port | `8126` | `8126` |
| `DD_DOGSTATSD_PORT` | DogStatsD agent port | `8125` | `8125` |
| `DD_ENV` | Environment for Datadog APM | _(uses TW_MCP_ENV)_ | `staging`, `production` |
| `DD_VERSION` | Version for Datadog APM | _(uses TW_MCP_VERSION)_ | `v1.0.0` |


### 🧪 Testing

```bash
# Run tests
go test ./...

# Run specific package tests
go test ./internal/twprojects/
```

### 🤓 Debugging

#### 🚀 Running the Server

Execute the MCP server in HTTP mode:

```bash
TW_MCP_URL=https://mcp.example.dev.stg.teamworkops.com \
  TW_MCP_API_URL=https://example.dev.stg.teamworkops.com \
  TW_MCP_SERVER_ADDRESS=:8012 \
  go run cmd/mcp-http/main.go
```

#### 🔍 MCP Inspector

For debugging purposes, use the [MCP Inspector tool](https://github.com/modelcontextprotocol/inspector):

```bash
NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem npx @modelcontextprotocol/inspector node build/index.js
```

> [!IMPORTANT]
> **Note**: The `NODE_EXTRA_CA_CERTS` environment variable is required when
> using OAuth2 authentication in the [CDE environment](https://github.com/Teamwork/dev-env-devspace).
> Download the certificate [here](https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem).

## 🏗️ Architecture

```
├── cmd/mcp-http/      # HTTP application entry point
├── cmd/mcp-stdio/     # STDIO application entry point
├── internal/
│   ├── config/        # Configuration management
│   ├── helpers/       # Utility functions
│   ├── toolsets/      # Tool framework and management
│   └── twprojects/    # Teamwork project operations
├── chart/             # Kubernetes Helm chart
└── Dockerfile         # Container build configuration
```