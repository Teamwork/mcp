package helpers

import (
	"errors"
	"fmt"

	mcp2 "github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
)

// HandleAPIError processes an error returned from the Teamwork API and converts
// it into an appropriate MCP tool result or error.
func HandleAPIError(err error, label string) (*mcp2.CallToolResult, error) {
	if err == nil {
		return nil, nil
	}

	var httpErr *twapi.HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.StatusCode >= 500:
			return &mcp2.CallToolResult{
				IsError: true,
				Content: []mcp2.Content{
					&mcp2.TextContent{
						Text: fmt.Sprintf("server error: %s", err.Error()),
					},
				},
			}, nil
		case httpErr.StatusCode >= 400:
			return &mcp2.CallToolResult{
				IsError: true,
				Content: []mcp2.Content{
					&mcp2.TextContent{
						Text: fmt.Sprintf("bad request: %s", err.Error()),
					},
				},
			}, nil
		default:
			return &mcp2.CallToolResult{
				IsError: true,
				Content: []mcp2.Content{
					&mcp2.TextContent{
						Text: fmt.Sprintf("unexpected HTTP status: %s", err.Error()),
					},
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", label, err)
}
