package twprojects_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk"
)

type sessionMock struct{}

func (s sessionMock) Authenticate(context.Context, *http.Request) error { return nil }
func (s sessionMock) Server() string                                    { return "https://example.com" }

var engineMock = func(status int, response []byte) *twapi.Engine {
	return twapi.NewEngine(sessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Status:     http.StatusText(status),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBuffer(response)),
			}, nil
		})
	}))
}

func mcpServerMock(t *testing.T, status int, response []byte) *server.MCPServer {
	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	toolsetGroup := twprojects.DefaultToolsetGroup(false, true, engineMock(status, response))
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer
}

type toolRequest struct {
	mcp.CallToolRequest

	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
}

func checkMessage(t *testing.T, message mcp.JSONRPCMessage) {
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
