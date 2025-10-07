package helpers

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewToolResultText creates a new text-based tool result.
func NewToolResultText(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf(format, args...),
			},
		},
	}
}

// NewToolResultJSON creates a new JSON-based tool result.
func NewToolResultJSON(v any) (*mcp.CallToolResult, error) {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResult{
		// For backward compatibility, we still return the JSON as text content
		// even though we have structured content.
		//
		// https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(encoded),
			},
		},
		StructuredContent: v,
	}, nil
}
