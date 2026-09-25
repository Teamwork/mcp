package twdesk

import (
	"context"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCustomFieldList toolsets.Method = "twdesk-list_custom_fields"
)

// customFieldService addresses the ticket custom field routes, which the SDK
// models no service for.
func customFieldService(client *deskclient.Client) *deskclient.Service[map[string]any, map[string]any] {
	return deskclient.NewService[map[string]any, map[string]any](client,
		deskclient.NewDefaultPathHandler("customfields"),
	)
}

// CustomFieldList lists the ticket custom fields of a Teamwork Desk
// installation, with the options of the fields that have them.
func CustomFieldList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := paginationOptions(map[string]*jsonschema.Schema{
		"inboxIDs": {
			Description: "Filter to the custom fields used by these inboxes. Use twdesk-list_inboxes to discover.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
	})

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomFieldList),
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Custom Fields",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			Description: "List ticket custom fields with their kind (text, textarea, number, date, dropdown " +
				"or checkboxes). A dropdown or checkboxes field lists its options, named under " +
				"included.customfieldoptions. A field ID goes in customFields on twdesk-search_tickets: " +
				"a text, number or date field is matched with value, and a dropdown or checkboxes " +
				"field with values, holding option IDs.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           properties,
				Required:             append(paginationRequiredKeys(), "inboxIDs"),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			params := url.Values{}
			setPagination(&params, arguments)
			// The options are what a dropdown or checkboxes condition is written
			// in, and a field row carries only their IDs.
			params.Set("includes", "customfieldoptions")

			if inboxIDs := arguments.GetIntSlice("inboxIDs", nil); len(inboxIDs) > 0 {
				params.Set("filter", deskclient.NewFilter().In("inboxes.id", helpers.SliceToAny(inboxIDs)).Build())
			}

			fields, err := customFieldService(client).List(ctx, params)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list custom fields")
			}
			return helpers.NewToolResultJSON(*fields)
		},
	}
}
