package helpers

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// deskStatusCodePattern matches the HTTP status that the Desk SDK reports in
// the text of its errors.
//
// desksdkgo declares no error type at all: every non-2xx response leaves the
// SDK as a bare fmt.Errorf carrying only the code — "unexpected status code:
// %d" from the generic Service methods and the ticket/help-doc services
// (desksdkgo/client/resource.go, tickets.go, helpdocarticles.go), and
// "failed to upload file, status code: %d, ..." from Files.Upload — so the
// status cannot be recovered any other way. Without this, every Desk API
// failure falls through to the raw-error return below and reaches the client
// as a protocol error rather than a tool result the model can read and retry.
var deskStatusCodePattern = regexp.MustCompile(`status code: (\d{3})`)

// NewToolResultTextError creates a new MCP tool result representing an error with the
// given text message.
func NewToolResultTextError(format string, args ...any) *mcp.CallToolResult {
	text := fmt.Sprintf(format, args...)
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: text,
			},
		},
	}
}

// HandleAPIError processes an error returned from the Teamwork API and converts
// it into an appropriate MCP tool result or error.
//
// Anything the API answered with — a status the caller can act on — becomes an
// error tool result, so the model reads the failure and can retry. Only an error
// with no status behind it (a transport failure, a decode fault) is returned as
// a Go error, which the SDK turns into a protocol-level error.
//
// It reads the status from the v3 SDK's *twapi.HTTPError and, failing that, from
// the message text the Desk SDK produces. See deskStatusCodePattern.
func HandleAPIError(err error, label string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	var statusCode int
	var hasStatusCode bool
	if httpErr, ok := errors.AsType[*twapi.HTTPError](err); ok {
		statusCode, hasStatusCode = httpErr.StatusCode, true
	} else if match := deskStatusCodePattern.FindStringSubmatch(err.Error()); match != nil {
		statusCode, hasStatusCode = mustAtoi(match[1]), true
	}

	if hasStatusCode {
		switch {
		case statusCode >= 500:
			return NewToolResultTextError("server error: %s", err.Error()), nil
		case statusCode >= 400:
			return NewToolResultTextError("bad request: %s", err.Error()), nil
		default:
			return NewToolResultTextError("unexpected HTTP status: %s", err.Error()), nil
		}
	}
	return nil, fmt.Errorf("%s: %w", label, err)
}

// mustAtoi converts a string the caller already knows is three ASCII digits.
func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
