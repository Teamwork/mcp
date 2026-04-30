# Teamwork MCP HTTP CLI

> Command-line interface for interacting with the Teamwork MCP server via HTTP

[![Go](https://img.shields.io/badge/Go-1.26.0-blue.svg)](https://golang.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

## 📖 Overview

The Teamwork MCP HTTP CLI is a command-line tool that allows you to interact
with the Teamwork MCP (Model Context Protocol) server over HTTP. This tool
provides a simple way to test MCP tools, list available capabilities, and debug
integrations without needing to set up a full LLM client.

### 🤖 What is this CLI for?

This CLI tool is designed for:
- **Testing MCP tools**: Execute individual tools and see their responses
- **Development and debugging**: Validate MCP server functionality during development
- **Exploration**: Discover available tools and their parameters
- **Exporting tool catalogues**: Dump every tool's full schema as JSON for downstream systems

## ✨ Features

- **HTTP Transport**: Connect to MCP servers via HTTP with authentication
- **Tool Listing**: Display all available tools and their descriptions
- **Tool Execution**: Call specific tools with custom parameters
- **JSON Parameter Support**: Pass complex parameters as JSON strings
- **Structured Logging**: Clear output with detailed logging information

## 🚀 Quick Start

### 📋 Prerequisites

- Go 1.26 or later
- Valid Teamwork MCP server URL and bearer token
- Running Teamwork MCP server (see main [README](../mcp/README.md))

### 🏃 Running the CLI

#### Basic Usage

```bash
# List all available tools
go run cmd/mcp-http-cli/main.go list-tools

# Call a tool without parameters
go run cmd/mcp-http-cli/main.go \
  -mcp-url=https://my-mcp.example.com \
  -mcp-token=your-bearer-token \
  call-tool twprojects-list_projects

# Call a tool with JSON parameters
go run cmd/mcp-http-cli/main.go \
  -mcp-url=https://my-mcp.example.com \
  -mcp-token=your-bearer-token \
  call-tool twprojects-get_comment '{"id": "123456"}'
```

#### Using Environment Variables

Set the bearer token via environment variable:

```bash
export TW_MCP_BEARER_TOKEN=your-bearer-token

go run cmd/mcp-http-cli/main.go \
  -mcp-url=https://my-mcp.example.com \
  list-tools
```

### ⚙️ Configuration

The CLI accepts the following command-line flags:

| Flag | Environment Variable | Description | Default |
|------|---------------------|-------------|---------|
| `-mcp-url` | - | URL of the MCP server to connect to | `https://mcp.ai.teamwork.com` |
| `-mcp-token` | `TW_MCP_BEARER_TOKEN` | Bearer token for authentication | _(from environment)_ |

### 📝 Commands

#### `list-tools`

Lists all available tools from the MCP server.

```bash
go run cmd/mcp-http-cli/main.go list-tools
```

#### `export-tools`

Dumps every tool's full schema (name, title, description, `inputSchema`,
`outputSchema`, `annotations`, `_meta`) as a JSON object keyed by tool name.
Logs are written to stderr, the JSON document to stdout, so you can redirect
output directly to a file.

```bash
go run cmd/mcp-http-cli/main.go \
  -mcp-url=https://my-mcp.example.com \
  export-tools > mcp-tools.json
```

#### `call-tool <tool-name> [parameters]`

Calls a specific tool with optional JSON parameters.

```bash
# Without parameters
go run cmd/mcp-http-cli/main.go call-tool twprojects-list_projects

# With parameters
go run cmd/mcp-http-cli/main.go call-tool twprojects-get_comment '{"id": "123456"}'

# Complex parameters
go run cmd/mcp-http-cli/main.go call-tool twprojects-create_task '{
  "tasklist_id": "123456",
  "name": "New Task"
}'
```