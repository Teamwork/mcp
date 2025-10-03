package helpers

import "github.com/modelcontextprotocol/go-sdk/mcp"

// NewToolResultText creates a new text-based tool result.
func NewToolResultText(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: message,
			},
		},
	}
}
