# Teamwork MCP STDIO Server

> STDIO transport server for the Teamwork Model Context Protocol implementation

[![Go](https://img.shields.io/badge/Go-1.26.0-blue.svg)](https://golang.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

## 📖 Overview

The Teamwork MCP STDIO Server provides a STDIO (Standard Input/Output) interface
for the Model Context Protocol, enabling direct communication between Large
Language Models and Teamwork.com through standard input and output streams. This
server implements the MCP specification over STDIO transport, making it ideal
for desktop applications and development environments.

### 🤖 What is the STDIO Server for?

This STDIO server is designed for:
- **Desktop LLM applications**: Direct integration with local AI applications
- **Development and testing**: Easy local development with MCP tools
- **Single-user environments**: Efficient communication without network overhead
- **CLI tools and scripts**: Integration with command-line workflows

## ✨ Features

- **STDIO Transport**: Direct communication through standard input/output streams
- **Tool Framework**: Extensible toolset architecture supporting all Teamwork operations
- **Read-Only Mode**: Optional restriction to read-only operations for safety
- **Selective Toolsets**: Enable specific toolsets or operations as needed
- **Secure Authentication**: Bearer token-based authentication with Teamwork

## 🚀 Quick Start

### 📋 Prerequisites

- Go 1.26 or later
- Valid Teamwork API bearer token

### 🏃 Running the Server

```bash
# Basic STDIO server with all toolsets
TW_MCP_BEARER_TOKEN=your-bearer-token \
  go run cmd/mcp-stdio/main.go

# Read-only mode (safer for testing)
TW_MCP_BEARER_TOKEN=your-bearer-token \
  go run cmd/mcp-stdio/main.go -read-only

# Enable specific toolsets only
TW_MCP_BEARER_TOKEN=your-bearer-token \
  go run cmd/mcp-stdio/main.go -toolsets=twprojects-list_projects,twprojects-get_project
```

### ⚙️ Configuration

#### Command-Line Flags

| Flag | Description | Default | Example |
|------|-------------|---------|---------|
| `-toolsets` | Comma-separated list of toolsets to enable | `all` | `twprojects-list_projects,twprojects-get_project` |
| `-read-only` | Restrict the server to read-only operations | `false` | `-read-only` |

#### Environment Variables

The server can be configured using the following environment variables:

##### Authentication Variables
| Variable | Description | Example |
|----------|-------------|---------|
| `TW_MCP_BEARER_TOKEN` | Bearer token for Teamwork API (required) | `your-bearer-token` |

##### Server Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TW_MCP_VERSION` | Version of the MCP server | `dev` | `v1.0.0` |
| `TW_MCP_API_URL` | The Teamwork API base URL | `https://teamwork.com` | `https://example.teamwork.com` |

##### Logging Configuration
| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `TW_MCP_LOG_FORMAT` | Log output format | `text` | `json`, `text` |
| `TW_MCP_LOG_LEVEL` | Logging level | `info` | `debug`, `warn`, `error`, `fatal` |

## 📝 Usage Examples

### Basic Usage

```bash
# Enable all toolsets (default)
TW_MCP_BEARER_TOKEN=your-token go run cmd/mcp-stdio/main.go

# Read-only mode for safety
TW_MCP_BEARER_TOKEN=your-token go run cmd/mcp-stdio/main.go -read-only

# Enable only project and task operations
TW_MCP_BEARER_TOKEN=your-token go run cmd/mcp-stdio/main.go \
  -toolsets=twprojects-list_projects,twprojects-get_project,twprojects-list_tasks
```

### Integration with MCP Clients

The STDIO server can be integrated with any MCP-compatible client:

```json
{
  "mcpServers": {
    "teamwork": {
      "command": "go",
      "args": [
        "run", 
        "/path/to/teamwork/mcp/cmd/mcp-stdio/main.go"
      ],
      "env": {
        "TW_MCP_BEARER_TOKEN": "your-bearer-token"
      }
    }
  }
}
```

## 🧪 Testing

### MCP Inspector

For debugging purposes, use the [MCP Inspector tool](https://github.com/modelcontextprotocol/inspector):

```bash
NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem npx @modelcontextprotocol/inspector node build/index.js
```

> [!IMPORTANT]
> **Note**: The `NODE_EXTRA_CA_CERTS` environment variable is required when
> using OAuth2 authentication with the Let's Encrypt certification authority.
> Download the certificate [here](https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem).