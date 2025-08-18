// Package testutil provides shared testing utilities for MCP server tests.
package testutil

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	deskclient "github.com/teamwork/desksdkgo/client"
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
func ProjectsMCPServerMock(t *testing.T, status int, response []byte) *server.MCPServer {
	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	toolsetGroup := twprojects.DefaultToolsetGroup(false, true, ProjectsEngineMock(status, response))
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer
}

// DeskMCPServerMock creates a mock MCP server for twdesk testing
func DeskMCPServerMock(t *testing.T, status int, response []byte) (*server.MCPServer, func()) {
	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	client, testServer := DeskClientMock(status, response)
	cleanup := func() {
		testServer.Close()
	}

	toolsetGroup := twdesk.DefaultToolsetGroup(client)
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		cleanup()
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer, cleanup
}

// ToolRequest represents a tool request for testing
type ToolRequest struct {
	mcp.CallToolRequest

	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
}

// CheckMessage validates that a message represents a successful tool execution
func CheckMessage(t *testing.T, message mcp.JSONRPCMessage) {
	t.Helper()

	switch m := message.(type) {
	case mcp.JSONRPCError:
		t.Errorf("tool failed to execute: %v", m.Error)
	case mcp.JSONRPCResponse:
		if toolResult, ok := m.Result.(mcp.CallToolResult); ok {
			if toolResult.IsError {
				t.Errorf("tool failed to execute: %v", toolResult.Content)
			}
		} else {
			t.Errorf("unexpected result type: %T", m.Result)
		}
	default:
		t.Errorf("unexpected message type: %T", m)
	}
}

// ExecuteToolRequest executes a tool request and validates the response
func ExecuteToolRequest(t *testing.T, mcpServer *server.MCPServer, toolName string, args map[string]any) {
	t.Helper()

	request := &ToolRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		CallToolRequest: mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsCall),
			},
		},
	}
	request.Params.Name = toolName
	request.Params.Arguments = args

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}

	CheckMessage(t, mcpServer.HandleMessage(context.Background(), encodedRequest))
}
