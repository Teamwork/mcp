# Test Utilities

This package provides shared testing utilities for the MCP server tests in both `twprojects` and `twdesk` packages.

## Overview

The `testutil` package centralizes common test infrastructure to avoid code duplication and provide consistent testing patterns across all MCP tool implementations.

## Usage

### For Teamwork Projects Tests

```go
import "github.com/teamwork/mcp/internal/testutil"

func TestSomething(t *testing.T) {
    mcpServer := testutil.ProjectsMCPServerMock(t, http.StatusOK, []byte(`{"id": 123}`))

    // Use testutil.ExecuteToolRequest for simple cases
    testutil.ExecuteToolRequest(t, mcpServer, "twprojects-get_comment", map[string]any{
        "id": float64(123),
    })
}
```

### For Teamwork Desk Tests

```go
import "github.com/teamwork/mcp/internal/testutil"

func TestSomething(t *testing.T) {
    mcpServer, cleanup := testutil.DeskMCPServerMock(t, http.StatusOK, []byte(`{"ticket_priority": {"id": 123}}`))
    defer cleanup()

    // Use testutil.ExecuteToolRequest for simple cases
    testutil.ExecuteToolRequest(t, mcpServer, "twdesk-get_priority", map[string]any{
        "id": "123",
    })
}
```

## Components

- **ProjectsMCPServerMock**: Creates a mock MCP server for testing twprojects tools
- **DeskMCPServerMock**: Creates a mock MCP server for testing twdesk tools (with cleanup function)
- **CheckMessage**: Validates that a tool execution was successful
- **ExecuteToolRequest**: Helper to execute a tool request and validate the response
- **ToolRequest**: Type alias for tool request structures

## Migration Guide

To migrate existing tests to use the shared infrastructure:

1. Replace your local `mcpServerMock` function calls with `testutil.ProjectsMCPServerMock` or `testutil.DeskMCPServerMock`
2. Replace your local `checkMessage` function with `testutil.CheckMessage`
3. Replace your local `toolRequest` type with `testutil.ToolRequest`
4. Update imports to include `"github.com/teamwork/mcp/internal/testutil"`

This approach ensures consistency across all test suites and makes it easier to add new tool test suites in the future.
