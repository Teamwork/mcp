// Package testutil wires this server's product toolset groups onto the
// product-neutral mocks in pkg/testutil, so a product's tests get a ready MCP
// server from a status code and a canned body.
//
// Everything reusable lives in pkg/testutil. Only the group wiring belongs
// here — a server built on this repo writes its own equivalent of this file for
// its own groups, and shares the rest.
package testutil

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	pkgtestutil "github.com/teamwork/mcp/pkg/testutil"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// Re-exported from pkg/testutil so a product's tests need only this package.
type (
	// ProjectsSessionMock implements a mock session for twprojects testing.
	ProjectsSessionMock = pkgtestutil.SessionMock

	// ProjectsMockRoute pairs a substring match against the request URL path with
	// the status and body to return when it matches.
	ProjectsMockRoute = pkgtestutil.MockRoute

	// ProjectsRecordedRequest is one HTTP request captured by
	// ProjectsMCPServerRecordingMock.
	ProjectsRecordedRequest = pkgtestutil.RecordedRequest

	// ToolRequest represents a tool request for testing.
	ToolRequest = pkgtestutil.ToolRequest

	// ExecuteToolRequestOptions represents options for ExecuteToolRequest.
	ExecuteToolRequestOptions = pkgtestutil.ExecuteToolRequestOptions

	// ExecuteToolRequestOption is a function that modifies
	// ExecuteToolRequestOptions.
	ExecuteToolRequestOption = pkgtestutil.ExecuteToolRequestOption
)

// Re-exported from pkg/testutil so a product's tests need only this package.
var (
	// ProjectsEngineMock creates a mock twapi.Engine with the given HTTP response.
	ProjectsEngineMock = pkgtestutil.EngineMock

	// DeskClientMock creates a mock desk client with a test server.
	DeskClientMock = pkgtestutil.DeskClientMock

	// CheckMessage validates that a message represents a successful tool
	// execution.
	CheckMessage = pkgtestutil.CheckMessage

	// ExecuteToolRequest executes a tool request and validates the response.
	ExecuteToolRequest = pkgtestutil.ExecuteToolRequest

	// ExecuteToolRequestWithCheckMessage executes a tool request and validates the
	// response with a custom check function.
	ExecuteToolRequestWithCheckMessage = pkgtestutil.ExecuteToolRequestWithCheckMessage
)

// projectsMCPServer wires a twprojects toolset group backed by the given engine
// into a fresh in-memory MCP server.
func projectsMCPServer(t *testing.T, engine *twapi.Engine) *mcp.Server {
	t.Helper()
	return pkgtestutil.MCPServer(t, twprojects.DefaultToolsetGroup(false, true, engine))
}

// ProjectsMCPServerMock creates a mock MCP server for twprojects testing.
func ProjectsMCPServerMock(t *testing.T, status int, response []byte) *mcp.Server {
	t.Helper()
	return projectsMCPServer(t, pkgtestutil.EngineMock(status, response))
}

// ProjectsMCPServerMockWithRequestBody is like ProjectsMCPServerMock but also
// captures the body of the most recent HTTP request the engine sent, so tests
// can assert on the serialized request payload. The returned pointer is
// populated after a tool invokes the engine.
func ProjectsMCPServerMockWithRequestBody(t *testing.T, status int, response []byte) (*mcp.Server, *[]byte) {
	t.Helper()
	engine, lastBody := pkgtestutil.EngineMockWithRequestBody(status, response)
	return projectsMCPServer(t, engine), lastBody
}

// ProjectsMCPServerMockWithRequestURL is like ProjectsMCPServerMock but also
// captures the URL of the most recent HTTP request the engine sent, so tests
// can assert on the query string a tool actually builds.
//
// Asserting the response alone cannot catch a filter or pagination parameter
// that is never sent: the mock replies with the same canned body regardless,
// so a dropped parameter looks identical to a working one.
func ProjectsMCPServerMockWithRequestURL(t *testing.T, status int, response []byte) (*mcp.Server, *url.URL) {
	t.Helper()
	engine, lastURL := pkgtestutil.EngineMockWithRequestURL(status, response)
	return projectsMCPServer(t, engine), lastURL
}

// ProjectsMCPServerMockWithRequestURLs is like ProjectsMCPServerMockWithRequestURL
// but captures every request the engine sends rather than only the last one.
//
// Use it when the tool under test makes more than one call, because the last
// URL is then the follow-up rather than the one being asserted on:
// list_custom_item_records resolves the custom item's field schema after
// listing, so its list query string is already gone by the time the tool
// returns, and a test reading lastURL reports the ordering missing from a
// request that never carried it.
func ProjectsMCPServerMockWithRequestURLs(t *testing.T, status int, response []byte) (*mcp.Server, *[]url.URL) {
	t.Helper()
	engine, urls := pkgtestutil.EngineMockWithRequestURLs(status, response)
	return projectsMCPServer(t, engine), urls
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
	return projectsMCPServer(t, pkgtestutil.RoutedEngineMock(routes, fallbackStatus, fallbackBody))
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
	engine, lastBody := pkgtestutil.RoutedEngineMockWithRequestBody(routes, fallbackStatus, fallbackBody)
	return projectsMCPServer(t, engine), lastBody
}

// ProjectsMCPServerRecordingMock is like ProjectsMCPServerRoutedMock but
// records every request in order rather than only the last body. Tools that fan
// one call out into many writes need this: the order of those writes is part of
// the contract (move_tasks must patch a parent before its children), and a
// single captured body cannot show it.
func ProjectsMCPServerRecordingMock(
	t *testing.T,
	routes []ProjectsMockRoute,
	fallbackStatus int,
	fallbackBody []byte,
) (*mcp.Server, *[]ProjectsRecordedRequest) {
	t.Helper()
	engine, recorded := pkgtestutil.RecordingEngineMock(routes, fallbackStatus, fallbackBody)
	return projectsMCPServer(t, engine), recorded
}

// ProjectsMCPServerSequencedMock creates a mock MCP server for twprojects
// testing whose engine returns the given response bodies in order, one per HTTP
// request the engine makes. Once the sequence is exhausted the final body is
// repeated. This lets tests drive a tool's internal pagination loop with a
// distinct body per page, or exercise a never-ending "hasMore" by supplying a
// single always-more body. All responses share the same status code.
func ProjectsMCPServerSequencedMock(t *testing.T, status int, responses ...[]byte) *mcp.Server {
	t.Helper()
	return projectsMCPServer(t, pkgtestutil.SequencedEngineMock(t, status, responses...))
}

// ChatMCPServerMock creates a mock MCP server for twchat testing. The twchat
// tools ride the shared twapi.Engine, so it reuses the engine mock to return
// the canned HTTP response.
func ChatMCPServerMock(t *testing.T, status int, response []byte) *mcp.Server {
	t.Helper()
	engine := pkgtestutil.EngineMock(status, response)
	return pkgtestutil.MCPServer(t, twchat.DefaultToolsetGroup(false, engine))
}

// DeskMCPServerMock creates a mock MCP server for twdesk testing. It injects the
// test server URL into the request context so handlers use the correct endpoint.
func DeskMCPServerMock(t *testing.T, status int, response []byte) (*mcp.Server, func()) {
	t.Helper()
	return deskMCPServer(t, pkgtestutil.HTTPServerMock(status, response))
}

// DeskMCPServerMockWithRequestURL is like DeskMCPServerMock but also captures
// the URL of the most recent HTTP request a tool sent, so tests can assert on
// the query string the tool actually builds.
//
// Asserting the response alone cannot catch a filter or pagination parameter
// that is never sent: the mock replies with the same canned body regardless, so
// a dropped parameter looks identical to a working one.
//
// The URL is returned through an accessor rather than a pointer because the
// capture happens on the httptest server's goroutine.
func DeskMCPServerMockWithRequestURL(
	t *testing.T,
	status int,
	response []byte,
) (*mcp.Server, func() url.URL, func()) {
	t.Helper()

	mcpServer, lastRequest, cleanup := DeskMCPServerMockWithRequest(t, status, response)
	return mcpServer, func() url.URL {
		_, requestURL := lastRequest()
		return requestURL
	}, cleanup
}

// DeskMCPServerMockWithRequest is like DeskMCPServerMockWithRequestURL but also
// reports the HTTP method of the most recent request.
//
// The method is what separates some tools from each other: the ticket task link
// and unlink tools address the same path and differ only in POST versus DELETE,
// so a test that checks the URL alone passes when the two are swapped.
func DeskMCPServerMockWithRequest(
	t *testing.T,
	status int,
	response []byte,
) (*mcp.Server, func() (string, url.URL), func()) {
	t.Helper()

	testServer, lastRequest := pkgtestutil.RecordingHTTPServerMock(status, response)
	mcpServer, cleanup := deskMCPServer(t, testServer)
	return mcpServer, lastRequest, cleanup
}

// deskMCPServer wires a twdesk toolset group onto the given test server.
func deskMCPServer(t *testing.T, testServer *httptest.Server) (*mcp.Server, func()) {
	t.Helper()

	group := twdesk.DefaultToolsetGroup(false, testServer.Client())
	mcpServer := pkgtestutil.MCPServerWithCustomerURL(t, testServer.URL, group)
	return mcpServer, testServer.Close
}

// SpacesMCPServerMock creates a mock MCP server for twspaces testing. It injects
// the test server URL into the request context so handlers use the correct
// endpoint.
func SpacesMCPServerMock(t *testing.T, status int, response []byte) (*mcp.Server, func()) {
	t.Helper()

	testServer := pkgtestutil.HTTPServerMock(status, response)
	group := twspaces.DefaultToolsetGroup(false, true, testServer.Client())
	mcpServer := pkgtestutil.MCPServerWithCustomerURL(t, testServer.URL, group)
	return mcpServer, testServer.Close
}
