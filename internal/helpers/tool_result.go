package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	twapi "github.com/teamwork/twapi-go-sdk"
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

// NewRawToolResult executes requester and returns the API response body
// verbatim, with web links injected, instead of decoding it into the SDK's
// typed response.
//
// A `get_*` tool normally marshals that typed response, which is fine while it
// carries every attribute. It stops being fine under a sparse fieldset: the SDK
// structs do not use `omitempty`, so the attributes the caller excluded come
// back as zero values — `null`, `0`, `""` — which a caller cannot tell apart
// from real data. Streaming the body keeps them absent, which is what the
// selection asked for.
//
// label is used for the error messages; buildPath is the WebLinker path builder
// for the entity.
func NewRawToolResult[R twapi.HTTPRequester](
	ctx context.Context,
	engine *twapi.Engine,
	requester R,
	label string,
	buildPath func(map[string]any) string,
) (*mcp.CallToolResult, error) {
	resp, err := twapi.ExecuteRaw(ctx, engine, requester)
	if err != nil {
		return HandleAPIError(err, label)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return HandleAPIError(twapi.NewHTTPError(resp, label), label)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	linked := WebLinker(ctx, body, buildPath)
	var structured any
	if err := json.Unmarshal(linked, &structured); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: string(linked),
			},
		},
		StructuredContent: structured,
	}, nil
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
