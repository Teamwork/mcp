package helpers

import (
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// HandleAPIError processes an error returned from the Teamwork API and converts
// it into an appropriate MCP tool result or error.
func HandleAPIError(err error, label string) (*mcp.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	var httpErr *twapi.HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.StatusCode >= 500:
			return nil, fmt.Errorf("server error: %w", err)
		case httpErr.StatusCode >= 400:
			return mcp.NewToolResultErrorFromErr("bad request", err), nil
		default:
			return mcp.NewToolResultErrorFromErr("unexpected HTTP status", err), nil
		}
	}
	return nil, fmt.Errorf("%s: %w", label, err)
}
