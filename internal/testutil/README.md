# Test Utilities

This package wires this server's product toolset groups onto the product-neutral
mocks in [`pkg/testutil`](../../pkg/testutil), so a product's tests get a ready
MCP server from a status code and a canned body.

## Which package to import

- **`internal/testutil`** (this one) — writing tests for a product in this repo.
  Everything below is here, plus the product wiring.
- **`pkg/testutil`** — writing tests for a server built on this repo, whose
  toolset groups this package knows nothing about. Assemble a server from your
  own group with `testutil.MCPServer(t, group)` and drive it with the same
  `ExecuteToolRequest`.

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
        "id": 123,
    })
}
```

### For a server built on this repo

```go
import (
    "github.com/teamwork/mcp/pkg/testutil"
    "github.com/teamwork/mcp/pkg/toolsets"
)

func TestSomething(t *testing.T) {
    engine, lastURL := testutil.EngineMockWithRequestURL(http.StatusOK, []byte(`{"items": []}`))
    mcpServer := testutil.MCPServer(t, mypackage.DefaultToolsetGroup(false, engine))

    testutil.ExecuteToolRequest(t, mcpServer, "twpro-list_items", map[string]any{
        "page_size": float64(50),
    })

    // Assert on the query string, not the response: the mock replies with the
    // same canned body either way, so a dropped parameter looks identical to a
    // working one.
    if got := lastURL.Query().Get("pageSize"); got != "50" {
        t.Errorf("pageSize = %q, want %q", got, "50")
    }
}
```

## Components

Per-product server builders (this package):

- **ProjectsMCPServerMock** and its `WithRequestBody` / `WithRequestURL` /
  `WithRequestURLs` / `Routed` / `Recording` / `Sequenced` variants
- **DeskMCPServerMock**, **DeskMCPServerMockWithRequestURL**,
  **DeskMCPServerMockWithRequest** (each returns a cleanup function)
- **SpacesMCPServerMock** (returns a cleanup function)
- **ChatMCPServerMock**

Re-exported from `pkg/testutil` so a product's tests need only this package:

- **CheckMessage**: validates that a tool execution was successful
- **ExecuteToolRequest**: executes a tool request and validates the response
- **ToolRequest**: type alias for tool request structures
- **ProjectsEngineMock**, **DeskClientMock**, **ProjectsMockRoute**,
  **ProjectsRecordedRequest**

Product-neutral, in `pkg/testutil` only:

- **MCPServer** / **MCPServerWithCustomerURL**: assemble a server from any group
- **EngineMock** and its capturing, routed, recording and sequenced variants
- **HTTPServerMock** / **RecordingHTTPServerMock**: for tools reached through an
  `*http.Client` rather than a `twapi.Engine`
