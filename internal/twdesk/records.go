package twdesk

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
)

// internalKeys are attributes the Desk API returns that are internal
// identifiers rather than data about the record, such as an agent's ldKey.
var internalKeys = helpers.NewKeySet("ldkey")

// minimizeDeskResponse encodes a Desk response without its internal
// identifiers and without the included sections that hold nothing. It returns
// the encoded body and its decoded form.
func minimizeDeskResponse(v any) ([]byte, any, error) {
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		// Not an object: nothing to trim.
		return encoded, v, nil
	}

	helpers.RedactKeys(decoded, internalKeys)
	if included, ok := decoded["included"].(map[string]any); ok {
		for key, section := range included {
			if section == nil {
				delete(included, key)
			}
		}
		if len(included) == 0 {
			delete(decoded, "included")
		}
	}

	minimized, err := json.Marshal(decoded)
	if err != nil {
		return nil, nil, err
	}
	return minimized, decoded, nil
}

// newToolResultJSON answers with a Desk response minimized by
// minimizeDeskResponse.
func newToolResultJSON(v any) (*mcp.CallToolResult, error) {
	encoded, decoded, err := minimizeDeskResponse(v)
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(encoded)}},
		StructuredContent: decoded,
	}, nil
}
