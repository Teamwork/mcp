// Package testutil provides shared testing utilities for MCP server tests.
package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/config"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
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
	return projectsMCPServer(t, ProjectsEngineMock(status, response))
}

// ProjectsMCPServerMockWithRequestBody is like ProjectsMCPServerMock but also
// captures the body of the most recent HTTP request the engine sent, so tests
// can assert on the serialized request payload. The returned pointer is
// populated after a tool invokes the engine.
func ProjectsMCPServerMockWithRequestBody(t *testing.T, status int, response []byte) (*mcp.Server, *[]byte) {
	var lastBody []byte
	engine := twapi.NewEngine(ProjectsSessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
			if req.Body != nil {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				lastBody = body
			}
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
	return projectsMCPServer(t, engine), &lastBody
}

// projectsMCPServer wires a twprojects toolset group backed by the given engine
// into a fresh in-memory MCP server.
func projectsMCPServer(t *testing.T, engine *twapi.Engine) *mcp.Server {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	toolsetGroup := twprojects.DefaultToolsetGroup(false, true, engine)
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer
}

// ChatMCPServerMock creates a mock MCP server for twchat testing. The twchat
// tools ride the shared twapi.Engine, so it reuses ProjectsEngineMock to return
// the canned HTTP response.
func ChatMCPServerMock(t *testing.T, status int, response []byte) *mcp.Server {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	toolsetGroup := twchat.DefaultToolsetGroup(false, ProjectsEngineMock(status, response))
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer
}

// ProjectsMockRoute pairs a substring match against the request URL path with
// the status and body to return when it matches.
type ProjectsMockRoute struct {
	Match  string
	Status int
	Body   []byte
}

// ProjectsMCPServerRoutedMock creates a mock MCP server for twprojects testing
// whose engine returns different responses based on a substring match against
// the request URL. Use this when a single tool dispatches calls to multiple
// endpoints that need distinct status codes (e.g. record create, which lists
// fields with 200 before posting the record with 201). Requests that don't
// match any route fall back to fallbackStatus/fallbackBody.
func ProjectsMCPServerRoutedMock(
	t *testing.T,
	routes []ProjectsMockRoute,
	fallbackStatus int,
	fallbackBody []byte,
) *mcp.Server {
	t.Helper()

	engine := twapi.NewEngine(ProjectsSessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path
			for _, route := range routes {
				if strings.Contains(path, route.Match) {
					return newProjectsMockHTTPResponse(route.Status, route.Body), nil
				}
			}
			return newProjectsMockHTTPResponse(fallbackStatus, fallbackBody), nil
		})
	}))

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	toolsetGroup := twprojects.DefaultToolsetGroup(false, true, engine)
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	return mcpServer
}

// ProjectsMCPServerRoutedMockWithRequestBody is like ProjectsMCPServerRoutedMock
// but also captures the body of the most recent HTTP request that carried one,
// so tests can assert on the serialized payload of the final write while still
// serving distinct responses per endpoint (e.g. a field-type GET at 200
// followed by a value POST at 201).
func ProjectsMCPServerRoutedMockWithRequestBody(
	t *testing.T,
	routes []ProjectsMockRoute,
	fallbackStatus int,
	fallbackBody []byte,
) (*mcp.Server, *[]byte) {
	t.Helper()

	var lastBody []byte
	engine := twapi.NewEngine(ProjectsSessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(req *http.Request) (*http.Response, error) {
			if req.Body != nil {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				lastBody = body
			}
			path := req.URL.Path
			for _, route := range routes {
				if strings.Contains(path, route.Match) {
					return newProjectsMockHTTPResponse(route.Status, route.Body), nil
				}
			}
			return newProjectsMockHTTPResponse(fallbackStatus, fallbackBody), nil
		})
	}))

	return projectsMCPServer(t, engine), &lastBody
}

// ProjectsMCPServerSequencedMock creates a mock MCP server for twprojects
// testing whose engine returns the given response bodies in order, one per HTTP
// request the engine makes. Once the sequence is exhausted the final body is
// repeated. This lets tests drive a tool's internal pagination loop with a
// distinct body per page, or exercise a never-ending "hasMore" by supplying a
// single always-more body. All responses share the same status code.
func ProjectsMCPServerSequencedMock(t *testing.T, status int, responses ...[]byte) *mcp.Server {
	t.Helper()

	if len(responses) == 0 {
		t.Fatal("ProjectsMCPServerSequencedMock requires at least one response body")
	}

	var mu sync.Mutex
	var idx int
	engine := twapi.NewEngine(ProjectsSessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
			mu.Lock()
			body := responses[len(responses)-1]
			if idx < len(responses) {
				body = responses[idx]
			}
			idx++
			mu.Unlock()
			return newProjectsMockHTTPResponse(status, body), nil
		})
	}))

	return projectsMCPServer(t, engine)
}

func newProjectsMockHTTPResponse(status int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
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
	toolsetGroup := twdesk.DefaultToolsetGroup(false, httpClient)
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

// SpacesMCPServerMock creates a mock MCP server for twspaces testing.
// It injects the test server URL into the request context so handlers use the correct endpoint.
func SpacesMCPServerMock(t *testing.T, status int, response []byte) (*mcp.Server, func()) {
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, err := w.Write(response)
		if err != nil {
			slog.Error("failed to write response", "error", err.Error())
		}
	}))
	testServerURL := testServer.URL
	cleanup := func() {
		testServer.Close()
	}

	httpClient := testServer.Client()
	toolsetGroup := twspaces.DefaultToolsetGroup(false, true, httpClient)
	if err := toolsetGroup.EnableToolsets(toolsets.MethodAll); err != nil {
		cleanup()
		t.Fatalf("failed to enable toolsets: %v", err)
	}
	toolsetGroup.RegisterAll(mcpServer)

	// Add middleware to inject test server URL into context so handlers route correctly
	mcpServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (result mcp.Result, err error) {
			ctx = config.WithCustomerURL(ctx, testServerURL)
			return next(ctx, method, req)
		}
	})

	return mcpServer, cleanup
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
