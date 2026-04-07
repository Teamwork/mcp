package helpers

import (
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
)

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
func HandleAPIError(err error, label string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	if httpErr, ok := errors.AsType[*twapi.HTTPError](err); ok {
		switch {
		case httpErr.StatusCode >= 500:
			return NewToolResultTextError("server error: %s", err.Error()), nil
		case httpErr.StatusCode >= 400:
			return NewToolResultTextError("bad request: %s", err.Error()), nil
		default:
			return NewToolResultTextError("unexpected HTTP status: %s", err.Error()), nil
		}
	}
	return nil, fmt.Errorf("%s: %w", label, err)
}
