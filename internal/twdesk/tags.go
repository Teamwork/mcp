package twdesk

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	deskclient "github.com/teamwork/desksdkgo/client"
	deskmodels "github.com/teamwork/desksdkgo/models"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTagCreate toolsets.Method = "twdesk-create_tag"
	MethodTagUpdate toolsets.Method = "twdesk-update_tag"
	MethodTagGet    toolsets.Method = "twdesk-get_tag"
	MethodTagList   toolsets.Method = "twdesk-list_tags"
)

// TagGet finds a tag in Teamwork Desk.  This will find it by ID
func TagGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Tag",
				ReadOnlyHint: true,
			},
			Description: "Get Desk tag.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to retrieve.",
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

			tag, err := client.Tags.Get(ctx, arguments.GetInt("id", 0), getParams(arguments))
			if err != nil {
				return nil, fmt.Errorf("failed to get tag: %w", err)
			}
			return helpers.NewToolResultJSON(tag)
		},
	}
}

// TagList returns a list of tags that apply to the filters in Teamwork Desk
func TagList(httpClient *http.Client) toolsets.ToolWrapper {
	properties := map[string]*jsonschema.Schema{
		"name": {
			Description: "The name of the tag to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "string"},
				{Type: "null"},
			},
		},
		"color": {
			Description: "The color of the tag to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "string"},
				{Type: "null"},
			},
		},
		"inboxIDs": {
			Description: "The IDs of the inboxes to filter by.",
			AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
				{Type: "null"},
			},
		},
	}
	properties = paginationOptions(properties)

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tags",
				ReadOnlyHint: true,
			},
			Description: "List Desk tags. Filter by name, color, or inbox.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties:           properties,
				Required:             append(paginationRequiredKeys(), "name", "color", "inboxIDs"),
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			// Apply filters to the tag list
			name := arguments.GetString("name", "")
			color := arguments.GetString("color", "")
			inboxIDs := arguments.GetIntSlice("inboxIDs", []int{})

			filter := deskclient.NewFilter()
			if name != "" {
				filter = filter.Eq("name", name)
			}
			if color != "" {
				filter = filter.Eq("color", color)
			}
			if len(inboxIDs) > 0 {
				filter = filter.In("inboxes.id", helpers.SliceToAny(inboxIDs))
			}

			params := url.Values{}
			params.Set("filter", filter.Build())
			setPagination(&params, arguments)

			tags, err := client.Tags.List(ctx, params)
			if err != nil {
				return nil, fmt.Errorf("failed to list tags: %w", err)
			}
			return helpers.NewToolResultJSON(tags)
		},
	}
}

// TagCreate creates a tag in Teamwork Desk
func TagCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Tag",
			},
			Description: "Create Desk tag.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the tag.",
					},
					"color": {
						Description: "The color of the tag.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name", "color"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			name := arguments.GetString("name", "")
			tag, err := client.Tags.Create(ctx, &deskmodels.TagResponse{
				Tag: deskmodels.Tag{
					Name:  &name,
					Color: strPtr(arguments.GetString("color", "")),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create tag: %w", err)
			}
			return helpers.NewToolResultText("Tag created successfully with ID %d", tag.Tag.ID), nil
		},
	}
}

// TagUpdate updates a tag in Teamwork Desk
func TagUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Tag",
			},
			Description: "Update Desk tag.",
			InputSchema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: falseSchema(),
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to update.",
					},
					"name": {
						Description: "The new name of the tag.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"color": {
						Description: "The color of the tag.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id", "name", "color"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			_, err = client.Tags.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TagResponse{
				Tag: deskmodels.Tag{
					Name:  strPtr(arguments.GetString("name", "")),
					Color: strPtr(arguments.GetString("color", "")),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update tag: %w", err)
			}

			return helpers.NewToolResultText("Tag updated successfully"), nil
		},
	}
}
