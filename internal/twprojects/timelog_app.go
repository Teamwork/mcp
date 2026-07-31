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

// Standard _meta.ui.csp surface. Empty: the app is self-contained.
var timelogCreateWidgetCSP = map[string]any{
	"connectDomains":  []string{},
	"resourceDomains": []string{},
	"frameDomains":    []string{},
	"baseUriDomains":  []string{},
}

// Same lists for ChatGPT's legacy openai/widgetCSP key, which only recognises
// snake_case names. camelCase here fails OpenAI's tool scan.
var timelogCreateOpenAIWidgetCSP = map[string]any{
	"connect_domains":  []string{},
	"resource_domains": []string{},
	"frame_domains":    []string{},
	"redirect_domains": []string{},
}

var timelogCreateResourceMeta = mcp.Meta{
	"ui": map[string]any{
		"csp":           timelogCreateWidgetCSP,
		"prefersBorder": true,
	},
	"openai/widgetDescription":   timelogCreateAppDescription,
	"openai/widgetPrefersBorder": true,
	"openai/widgetCSP":           timelogCreateOpenAIWidgetCSP,
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
