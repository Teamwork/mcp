// Package testutil provides the product-neutral building blocks for MCP server
// tests: engines and HTTP servers that answer with a canned response while
// capturing what was sent, an in-memory MCP server assembled from any toolset
// group, and a helper that calls one tool through it.
//
// Nothing here knows about a particular Teamwork product. A product's test
// helpers wire their own toolset group on top — see internal/testutil for this
// server's, which is also the shape a server built on this repo should follow.
package testutil

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/mcp/pkg/twctx"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// SessionMock implements a twapi session that authenticates every request
// against a fixed example host.
type SessionMock struct{}

// Authenticate implements the Authenticate method for SessionMock.
func (s SessionMock) Authenticate(context.Context, *http.Request) error {
	return nil
}

// Server implements the Server method for SessionMock.
func (s SessionMock) Server() string {
	return "https://example.com"
}

// MockRoute pairs a substring match against the request URL path with the
// status and body to return when it matches.
type MockRoute struct {
	Match string
	// Method restricts the route to a single HTTP method. An empty value matches
	// any method, which is what path-only routing needs; set it when one path
	// serves several verbs, as /tasks/{id}.json does for get and update.
	Method string
	Status int
	Body   []byte
}

// RecordedRequest is one HTTP request captured by RecordingEngineMock.
type RecordedRequest struct {
	Method string
	// URL is the whole request URL, not only its path: a tool that reaches an
	// endpoint outside the API, as the pre-signed file upload does, is only
	// identifiable by its host, and a step that carries its parameters in the
	// query string has nothing in its body to assert on.
	URL  url.URL
	Body []byte
}

// NewMockHTTPResponse builds a complete http.Response around a status and body,
// as an engine middleware must return one.
func NewMockHTTPResponse(status int, body []byte) *http.Response {
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

// newEngine builds a twapi.Engine whose only middleware is the given round
// tripper, so no request ever leaves the process.
func newEngine(do func(*http.Request) (*http.Response, error)) *twapi.Engine {
	return twapi.NewEngine(SessionMock{}, twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
		return twapi.HTTPClientFunc(do)
	}))
}

// EngineMock creates a mock twapi.Engine that answers every request with the
// given HTTP response.
func EngineMock(status int, response []byte) *twapi.Engine {
	return newEngine(func(*http.Request) (*http.Response, error) {
		return NewMockHTTPResponse(status, response), nil
	})
}

// EngineMockWithRequestBody is like EngineMock but also captures the body of the
// most recent request the engine sent, so tests can assert on the serialized
// request payload. The returned pointer is populated once a tool calls out.
func EngineMockWithRequestBody(status int, response []byte) (*twapi.Engine, *[]byte) {
	var lastBody []byte
	engine := newEngine(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			lastBody = body
		}
		return NewMockHTTPResponse(status, response), nil
	})
	return engine, &lastBody
}

// EngineMockWithRequestURL is like EngineMock but also captures the URL of the
// most recent request the engine sent, so tests can assert on the query string a
// tool actually builds.
//
// Asserting the response alone cannot catch a filter or pagination parameter
// that is never sent: the mock replies with the same canned body regardless, so
// a dropped parameter looks identical to a working one.
func EngineMockWithRequestURL(status int, response []byte) (*twapi.Engine, *url.URL) {
	var lastURL url.URL
	engine := newEngine(func(req *http.Request) (*http.Response, error) {
		if req.URL != nil {
			lastURL = *req.URL
		}
		return NewMockHTTPResponse(status, response), nil
	})
	return engine, &lastURL
}

// EngineMockWithRequestURLs is like EngineMockWithRequestURL but captures every
// request rather than only the last one.
//
// Use it when the tool under test makes more than one call, because the last URL
// is then the follow-up rather than the one being asserted on: a tool that
// resolves a schema after listing has already discarded its list query string by
// the time it returns, and a test reading the last URL reports a parameter
// missing from a request that never carried it.
func EngineMockWithRequestURLs(status int, response []byte) (*twapi.Engine, *[]url.URL) {
	var urls []url.URL
	engine := newEngine(func(req *http.Request) (*http.Response, error) {
		if req.URL != nil {
			urls = append(urls, *req.URL)
		}
		return NewMockHTTPResponse(status, response), nil
	})
	return engine, &urls
}

// matchRoute returns the response for the first route matching the request, or
// the fallback when none does.
func matchRoute(req *http.Request, routes []MockRoute, fallbackStatus int, fallbackBody []byte) *http.Response {
	for _, route := range routes {
		if route.Method != "" && route.Method != req.Method {
			continue
		}
		if strings.Contains(req.URL.Path, route.Match) {
			return NewMockHTTPResponse(route.Status, route.Body)
		}
	}
	return NewMockHTTPResponse(fallbackStatus, fallbackBody)
}

// RoutedEngineMock creates a mock engine that returns different responses based
// on a substring match against the request URL path. Use this when a single tool
// dispatches calls to multiple endpoints that need distinct status codes (e.g.
// record create, which lists fields with 200 before posting the record with
// 201). Requests matching no route fall back to fallbackStatus/fallbackBody.
func RoutedEngineMock(routes []MockRoute, fallbackStatus int, fallbackBody []byte) *twapi.Engine {
	return newEngine(func(req *http.Request) (*http.Response, error) {
		return matchRoute(req, routes, fallbackStatus, fallbackBody), nil
	})
}

// RoutedEngineMockWithRequestBody is like RoutedEngineMock but also captures the
// body of the most recent request that carried one, so tests can assert on the
// serialized payload of the final write while still serving distinct responses
// per endpoint (e.g. a field-type GET at 200 followed by a value POST at 201).
func RoutedEngineMockWithRequestBody(
	routes []MockRoute,
	fallbackStatus int,
	fallbackBody []byte,
) (*twapi.Engine, *[]byte) {
	var lastBody []byte
	engine := newEngine(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			lastBody = body
		}
		return matchRoute(req, routes, fallbackStatus, fallbackBody), nil
	})
	return engine, &lastBody
}

// RecordingEngineMock is like RoutedEngineMock but records every request in
// order rather than only the last body. Tools that fan one call out into many
// writes need this: the order of those writes is part of the contract, and a
// single captured body cannot show it.
func RecordingEngineMock(
	routes []MockRoute,
	fallbackStatus int,
	fallbackBody []byte,
) (*twapi.Engine, *[]RecordedRequest) {
	var recorded []RecordedRequest
	engine := newEngine(func(req *http.Request) (*http.Response, error) {
		entry := RecordedRequest{Method: req.Method, URL: *req.URL}
		if req.Body != nil {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			entry.Body = body
		}
		recorded = append(recorded, entry)
		return matchRoute(req, routes, fallbackStatus, fallbackBody), nil
	})
	return engine, &recorded
}

// SequencedEngineMock creates a mock engine that returns the given response
// bodies in order, one per request. Once the sequence is exhausted the final
// body is repeated. This lets tests drive a tool's internal pagination loop with
// a distinct body per page, or exercise a never-ending "hasMore" by supplying a
// single always-more body. All responses share the same status code.
func SequencedEngineMock(t *testing.T, status int, responses ...[]byte) *twapi.Engine {
	t.Helper()

	if len(responses) == 0 {
		t.Fatal("SequencedEngineMock requires at least one response body")
	}

	var mu sync.Mutex
	var idx int
	return newEngine(func(*http.Request) (*http.Response, error) {
		mu.Lock()
		body := responses[len(responses)-1]
		if idx < len(responses) {
			body = responses[idx]
		}
		idx++
		mu.Unlock()
		return NewMockHTTPResponse(status, body), nil
	})
}

// HTTPServerMock starts a test server answering every request with the given
// status and body. Tools reached through an *http.Client rather than a
// twapi.Engine are tested against one of these.
func HTTPServerMock(status int, response []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		if _, err := w.Write(response); err != nil {
			slog.Error("failed to write response", "error", err.Error())
		}
	}))
}

// RecordingHTTPServerMock is like HTTPServerMock but also reports the method and
// URL of the most recent request.
//
// The method is what separates some tools from each other: a link and an unlink
// tool may address the same path and differ only in POST versus DELETE, so a
// test that checks the URL alone passes when the two are swapped.
//
// The request is returned through an accessor rather than a pointer because the
// capture happens on the test server's own goroutine.
func RecordingHTTPServerMock(status int, response []byte) (*httptest.Server, func() (string, url.URL)) {
	var mu sync.Mutex
	var lastMethod string
	var lastURL url.URL

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		lastMethod = r.Method
		lastURL = *r.URL
		mu.Unlock()

		w.WriteHeader(status)
		if _, err := w.Write(response); err != nil {
			slog.Error("failed to write response", "error", err.Error())
		}
	}))

	return server, func() (string, url.URL) {
		mu.Lock()
		defer mu.Unlock()
		return lastMethod, lastURL
	}
}

// DeskClientMock creates a Desk SDK client pointed at a test server answering
// with the given status and body. The caller owns closing the server.
func DeskClientMock(status int, response []byte) (*deskclient.Client, *httptest.Server) {
	server := HTTPServerMock(status, response)
	return deskclient.NewClient(server.URL, deskclient.WithAPIKey("test-token")), server
}

// MCPServer assembles an in-memory MCP server with every toolset of the given
// groups enabled, including their write and delete tools.
func MCPServer(t *testing.T, groups ...*toolsets.ToolsetGroup) *mcp.Server {
	t.Helper()

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{})

	for _, group := range groups {
		if err := group.EnableToolsets(toolsets.MethodAll); err != nil {
			t.Fatalf("failed to enable toolsets: %v", err)
		}
		group.RegisterAll(mcpServer)
	}

	return mcpServer
}

// MCPServerWithCustomerURL is like MCPServer but injects a customer URL into
// every request's context. Tools that resolve their endpoint from the caller's
// installation — anything built on an *http.Client rather than a pre-configured
// engine — reach the test server only because of this.
func MCPServerWithCustomerURL(
	t *testing.T,
	customerURL string,
	groups ...*toolsets.ToolsetGroup,
) *mcp.Server {
	t.Helper()

	mcpServer := MCPServer(t, groups...)
	mcpServer.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			return next(twctx.WithCustomerURL(ctx, customerURL), method, req)
		}
	})
	return mcpServer
}

// ToolRequest represents a tool request for testing.
type ToolRequest struct {
	mcp.CallToolRequest

	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
}

// CheckMessage validates that a message represents a successful tool execution.
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

// ExecuteToolRequest executes a tool request and validates the response.
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
	if _, err := mcpServer.Connect(t.Context(), serverTransport, nil); err != nil {
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
