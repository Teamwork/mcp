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
- **HTTP Streaming**: Efficient real-time communication with LLMs

## 🚀 Quick Start

### 📋 Prerequisites

- Go 1.24 or later
- Valid Teamwork API credentials

### 🧪 Testing

```bash
# Run tests
go test ./...

# Run specific package tests
go test ./internal/twprojects/
```

### 🤓 Debugging

#### 🚀 Running the Server

Execute the MCP server with optional configuration:

```bash
DEVENV_INSTALLATION=example.dev.stg.teamworkops.com SERVER_ADDRESS=:8012 go run cmd/mcp/main.go
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
├── cmd/mcp/           # Application entry point
├── internal/
│   ├── config/        # Configuration management
│   ├── helpers/       # Utility functions
│   ├── toolsets/      # Tool framework and management
│   └── twprojects/    # Teamwork project operations
├── chart/             # Kubernetes Helm chart
└── Dockerfile         # Container build configuration
```

### 🧩 Key Components

- **Toolsets**: Modular framework for organizing and managing tools
- **Config**: Environment-based configuration with resource management
- **TWProjects**: Teamwork Projects tools
- **HTTP Server**: Streamable HTTP server for MCP communication