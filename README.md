# Teamwork MCP Server

> Model Context Protocol server for Teamwork.com integration with Large Language
> Models

[![Go Report Card](https://goreportcard.com/badge/github.com/teamwork/mcp)](https://goreportcard.com/report/github.com/teamwork/mcp)
[![Go](https://img.shields.io/badge/Go-1.26.0-blue.svg)](https://golang.org/)
[![MCP](https://img.shields.io/badge/MCP-Compatible-green.svg)](https://modelcontextprotocol.io/)

📌 Are you a Teamwork.com user wanting to connect AI tools to your Teamwork.com
site right now? Jump straight to the **[Usage Guide](docs/usage/README.md)** for
tokens, enabling MCP, and client configuration examples.

## 📖 Overview

This MCP (Model Context Protocol) server enables seamless integration between
Large Language Models and Teamwork.com. It provides a standardized interface for
LLMs to interact with Teamwork.com projects, allowing AI agents to perform
various project management operations.

### 🤖 What is MCP?

Model Context Protocol (MCP) is an open protocol that standardizes how
applications provide context to LLMs. This server describes all the actions
available in Teamwork.com (tools) in a way that LLMs can understand and execute
through AI agents.

## ✨ Features

- **Multiple Transport Modes**: HTTP and STDIO interfaces for different deployment scenarios
- **Secure Authentication**: Bearer token and OAuth2 integration with Teamwork.com
- **Tool Framework**: Extensible toolset architecture for adding new capabilities
- **Production Ready**: Comprehensive logging, monitoring, and observability
- **Read-Only Mode**: Optional restriction to read-only operations for safety

## 🚀 Available Servers

This project provides three different ways to interact with the Teamwork.com MCP
server:

### 📡 HTTP Server

Production-ready HTTP server for cloud deployments and multi-client support.

**📖 [Full HTTP Server Documentation](cmd/mcp-http/README.md)**

Quick start:
```bash
TW_MCP_SERVER_ADDRESS=:8080 go run cmd/mcp-http/main.go
```

### 💬 STDIO Server

Direct STDIO interface for desktop applications and development environments.

**📖 [Full STDIO Server Documentation](cmd/mcp-stdio/README.md)**

Quick start:
```bash
TW_MCP_BEARER_TOKEN=your-token go run cmd/mcp-stdio/main.go
```

### 🛠️ HTTP CLI

Command-line tool for testing and debugging MCP server functionality.

**📖 [Full HTTP CLI Documentation](cmd/mcp-http-cli/README.md)**

Quick start:
```bash
go run cmd/mcp-http-cli/main.go -mcp-url=https://mcp.example.com list-tools
```

## 📋 Prerequisites

- Go 1.26 or later
- Valid Teamwork.com API credentials (bearer token or OAuth2 setup)

## 🧪 Development & Testing

### Running Tests
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/twprojects/
```

### MCP Inspector
For debugging purposes, use the [MCP Inspector tool](https://github.com/modelcontextprotocol/inspector):

```bash
NODE_EXTRA_CA_CERTS=letsencrypt-stg-root-x1.pem npx @modelcontextprotocol/inspector node build/index.js
```

> [!IMPORTANT]
> **Note**: The `NODE_EXTRA_CA_CERTS` environment variable is required when
> using OAuth2 authentication with the Let's Encrypt certification authority.
> Download the certificate [here](https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem).

## 🏗️ Architecture

```
├── cmd/
│   ├── mcp-http/          # HTTP server implementation
│   ├── mcp-stdio/         # STDIO server implementation
│   └── mcp-http-cli/      # CLI tool for testing via HTTP
├── internal/
│   ├── auth/              # Authentication helpers (bearer & OAuth2 token handling)
│   ├── config/            # Configuration management (env, flags)
│   ├── helpers/           # Shared utility functions (errors, link helpers, tool parsing)
│   ├── request/           # HTTP request primitives / Teamwork API wiring
│   ├── toolsets/          # Tool framework and registration logic
│   └── twprojects/        # Teamwork project/domain tools (tasks, tags, timers, etc.)
├── examples/              # Usage & integration examples (LangChain Node/Python)
├── docs/usage/            # End-user setup & connection guide
├── Makefile               # Common developer tasks
├── Dockerfile             # Container build configuration
├── CODE_OF_CONDUCT.md     # Community guidelines
├── CONTRIBUTING.md        # Contribution guide
└── SECURITY.md            # Security policy
```
