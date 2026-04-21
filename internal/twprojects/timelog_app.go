package twprojects

import (
	"context"
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/toolsets"
)

const (
	mcpAppMimeType                = "text/html;profile=mcp-app"
	timelogCreateAppURI           = "ui://teamwork/timelog-create"
	timelogCreateAppResourceTitle = "Create Timelog App"
	timelogCreateAppDescription   = "Interactive form for creating Teamwork timelogs."
)

var timelogCreateWidgetCSP = map[string]any{
	"connectDomains":  []string{},
	"resourceDomains": []string{},
	"frameDomains":    []string{},
	"baseUriDomains":  []string{},
}

var timelogCreateWidgetPermissions = map[string]any{
	"camera":         map[string]any{},
	"microphone":     map[string]any{},
	"geolocation":    map[string]any{},
	"clipboardWrite": map[string]any{},
}

var timelogCreateResourceMeta = mcp.Meta{
	"ui": map[string]any{
		"csp":           timelogCreateWidgetCSP,
		"permissions":   timelogCreateWidgetPermissions,
		"prefersBorder": true,
	},
	"openai/widgetDescription":   timelogCreateAppDescription,
	"openai/widgetPrefersBorder": true,
	"openai/widgetCSP":           timelogCreateWidgetCSP,
}

//go:embed apps/timelog_create.html
var timelogCreateAppHTML string

func timelogCreateReadHandler(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      timelogCreateAppURI,
				MIMEType: mcpAppMimeType,
				Text:     timelogCreateAppHTML,
				Meta:     timelogCreateResourceMeta,
			},
		},
	}, nil
}

// TimelogCreateAppResource returns the MCP Apps plain resource so it appears
// in resources/list.
//
// https://github.com/modelcontextprotocol/ext-apps/blob/main/specification/2026-01-26/apps.mdx#ui-resource-format
func TimelogCreateAppResource() toolsets.ServerResource {
	return toolsets.NewServerResource(
		&mcp.Resource{
			Name:        "twprojects-create_timelog-ui",
			Title:       timelogCreateAppResourceTitle,
			Description: timelogCreateAppDescription,
			MIMEType:    mcpAppMimeType,
			URI:         timelogCreateAppURI,
			Meta:        timelogCreateResourceMeta,
		},
		timelogCreateReadHandler,
	)
}
