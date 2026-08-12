package twdesk

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodHelpDocSiteGet  toolsets.Method = "twdesk-get_helpdoc_site"
	MethodHelpDocSiteList toolsets.Method = "twdesk-list_helpdoc_sites"
)

// HelpDocSiteGet retrieves a single help doc site by ID.
func HelpDocSiteGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocSiteGet),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Help Doc Site",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "Get a help doc site (knowledge base) by ID, including its subdomain, branding and article counts.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the help doc site to retrieve.",
					},
					"fields": sparseFieldsSchema(),
				},
				Required: []string{"id", "fields"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			site, err := client.HelpDocSites.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get help doc site")
			}
			return helpers.NewToolResultJSON(site)
		},
	}
}

// HelpDocSiteList returns a list of help doc sites that apply to the filters in
// Teamwork Desk.
func HelpDocSiteList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Description: "The name of the help doc site to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
		"subdomain": {
			Description: "The subdomain of the help doc site to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
				{Type: "null"},
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodHelpDocSiteList),
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Help Doc Sites",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "List help doc sites (knowledge bases). Filter by name or subdomain. " +
				"Use this to discover the site ID required by the help doc article tools.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           properties,
				Required:             append(paginationRequiredKeys(), "name", "subdomain"),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			// Apply filters to the help doc site list
			name := arguments.GetStringSlice("name", []string{})
			subdomain := arguments.GetStringSlice("subdomain", []string{})

			filter := deskclient.NewFilter()
			if len(name) > 0 {
				filter = filter.In("name", helpers.SliceToAny(name))
			}
			if len(subdomain) > 0 {
				filter = filter.In("subdomain", helpers.SliceToAny(subdomain))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			sites, err := client.HelpDocSites.List(ctx, params)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list help doc sites")
			}
			return helpers.NewToolResultJSON(sites)
		},
	}
}
