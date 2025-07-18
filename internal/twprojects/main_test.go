package twprojects_test

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/twapi-go-sdk"
)

type sessionMock struct{}

func (s sessionMock) Authenticate(context.Context, *http.Request) error { return nil }
func (s sessionMock) Server() string                                    { return "https://example.com" }

var engineMock = twapi.NewEngine(sessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
	return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		}, nil
	})
}))

func mcpServerMock() *server.MCPServer {
	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	mcpServer.AddTools(twprojects.ProjectCreate(engineMock))
	mcpServer.AddTools(twprojects.ProjectUpdate(engineMock))
	mcpServer.AddTools(twprojects.ProjectDelete(engineMock))
	mcpServer.AddTools(twprojects.ProjectGet(engineMock))
	mcpServer.AddTools(twprojects.ProjectList(engineMock))
	mcpServer.AddTools(twprojects.TasklistCreate(engineMock))
	mcpServer.AddTools(twprojects.TasklistUpdate(engineMock))
	mcpServer.AddTools(twprojects.TasklistDelete(engineMock))
	mcpServer.AddTools(twprojects.TasklistGet(engineMock))
	mcpServer.AddTools(twprojects.TasklistList(engineMock))
	mcpServer.AddTools(twprojects.TasklistListByProject(engineMock))
	mcpServer.AddTools(twprojects.TaskCreate(engineMock))
	mcpServer.AddTools(twprojects.TaskUpdate(engineMock))
	mcpServer.AddTools(twprojects.TaskDelete(engineMock))
	mcpServer.AddTools(twprojects.TaskGet(engineMock))
	mcpServer.AddTools(twprojects.TaskList(engineMock))
	mcpServer.AddTools(twprojects.TaskListByTasklist(engineMock))
	mcpServer.AddTools(twprojects.TaskListByProject(engineMock))
	mcpServer.AddTools(twprojects.UserCreate(engineMock))
	mcpServer.AddTools(twprojects.UserUpdate(engineMock))
	mcpServer.AddTools(twprojects.UserDelete(engineMock))
	mcpServer.AddTools(twprojects.UserGet(engineMock))
	mcpServer.AddTools(twprojects.UserGetMe(engineMock))
	mcpServer.AddTools(twprojects.UserList(engineMock))
	mcpServer.AddTools(twprojects.UserListByProject(engineMock))
	return mcpServer
}

type toolRequest struct {
	mcp.CallToolRequest

	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
}
