// Package testutil provides shared testing utilities for MCP server tests.
package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk"
)

// ProjectsSessionMock implements a mock session for twprojects testing
type ProjectsSessionMock struct{}

// Authenticate implements the Authenticate method for ProjectsSessionMock
func (s ProjectsSessionMock) Authenticate(context.Context, *http.Request) error {
	return nil
}

// Server implements the Server method for ProjectsSessionMock
func (s ProjectsSessionMock) Server() string {
	return "https://example.com"
}

// ProjectsEngineMock creates a mock twapi.Engine with the given HTTP response
func ProjectsEngineMock(status int, response []byte) *twapi.Engine {
	return twapi.NewEngine(ProjectsSessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Status:     http.StatusText(status),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(response))),
			}, nil
		})
	}))
}

// DeskClientMock creates a mock desk client with a test server
func DeskClientMock(status int, response []byte) (*deskclient.Client, *httptest.Server) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, err := w.Write(response)
		if err != nil {
			slog.Error("failed to write response", "error", err.Error())
		}
	}))

	client := deskclient.NewClient(server.URL, deskclient.WithAPIKey("test-token"))
	return client, server
}

// ProjectsMCPServerMock creates a mock MCP server for twprojects testing
func ProjectsMCPServerMock(t *testing.T, status int, response []byte) *mcp.Server {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	toolsetGroup := twprojects.DefaultToolsetGroup(false, true, ProjectsEngineMock(status, response))
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer
}

// DeskMCPServerMock creates a mock MCP server for twdesk testing
// It injects the test server URL into the request context so handlers use the correct endpoint
func DeskMCPServerMock(t *testing.T, status int, response []byte) (*mcp.Server, func()) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	_, testServer := DeskClientMock(status, response)
	testServerURL := testServer.URL
	cleanup := func() {
		testServer.Close()
	}

	httpClient := testServer.Client()
	toolsetGroup := twdesk.DefaultToolsetGroup(httpClient)
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		cleanup()
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	// Add middleware to inject test server URL into context so handlers route correctly
	mcpServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			// Inject the test server URL as the customer URL
			ctx = config.WithCustomerURL(ctx, testServerURL)
			return next(ctx, method, req)
		}
	})

	return mcpServer, cleanup
}

// ToolRequest represents a tool request for testing
type ToolRequest struct {
	mcp.CallToolRequest

	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
}

// CheckMessage validates that a message represents a successful tool execution
func CheckMessage(t *testing.T, result mcp.Result) {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Errorf("unexpected result type: %T", result)
		return
	}
	if toolResult.IsError {
		var msg any = toolResult.Content
		if len(toolResult.Content) == 1 {
			if textContent, ok := toolResult.Content[0].(*mcp.TextContent); ok {
				msg = textContent.Text
			}
		}
		t.Errorf("tool failed to execute: %v", msg)
	}
}

// ExecuteToolRequestOptions represents options for ExecuteToolRequest.
type ExecuteToolRequestOptions struct {
	checkMessage func(t *testing.T, result mcp.Result)
}

// ExecuteToolRequestOption is a function that modifies
// ExecuteToolRequestOptions.
type ExecuteToolRequestOption func(*ExecuteToolRequestOptions)

// ExecuteToolRequestWithCheckMessage executes a tool request and validates the
// response with a custom check function. Any nil function will be ignored.
func ExecuteToolRequestWithCheckMessage(f func(t *testing.T, result mcp.Result)) ExecuteToolRequestOption {
	return func(opts *ExecuteToolRequestOptions) {
		if f != nil {
			opts.checkMessage = f
		}
	}
}

// ExecuteToolRequest executes a tool request and validates the response
func ExecuteToolRequest(
	t *testing.T,
	mcpServer *mcp.Server,
	toolName string,
	args map[string]any,
	optFuncs ...ExecuteToolRequestOption,
) {
	t.Helper()

	options := &ExecuteToolRequestOptions{
		checkMessage: CheckMessage,
	}
	for _, fn := range optFuncs {
		fn(options)
	}

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	_, err := mcpServer.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect to client: %v", err)
	}
	defer clientSession.Close() //nolint:errcheck

	result, err := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("failed to call tool: %v", err)
	}

	options.checkMessage(t, result)
}
