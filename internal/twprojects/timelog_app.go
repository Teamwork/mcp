package twprojects

import (
	"context"
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/toolsets"
)

const (
	mcpAppMimeType                   = "text/html;profile=mcp-app"
	timelogCreateAppResourceURI      = "ui://teamwork/timelog-create"
	timelogCreateAppResourceTitle    = "Create Timelog App"
	timelogCreateAppResourceTemplate = "ui://teamwork/timelog-create"
)

//go:embed apps/timelog_create.html
var timelogCreateAppHTML string

// TimelogCreateAppResourceTemplate registers the MCP Apps resource used by the
// twprojects-create_timelog tool.
func TimelogCreateAppResourceTemplate() toolsets.ServerResourceTemplate {
	return toolsets.NewServerResourceTemplate(
		&mcp.ResourceTemplate{
			Name:        "twprojects-create_timelog-ui",
			Title:       timelogCreateAppResourceTitle,
			Description: "Interactive form for creating Teamwork timelogs.",
			MIMEType:    mcpAppMimeType,
			URITemplate: timelogCreateAppResourceTemplate,
		},
		func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: mcpAppMimeType,
						Text:     timelogCreateAppHTML,
					},
				},
			}, nil
		},
	)
}
