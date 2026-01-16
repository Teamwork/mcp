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

func init() {
	toolsets.RegisterMethod(MethodTagCreate)
	toolsets.RegisterMethod(MethodTagUpdate)
	toolsets.RegisterMethod(MethodTagGet)
	toolsets.RegisterMethod(MethodTagList)
}

// TagGet finds a tag in Teamwork Desk.  This will find it by ID
func TagGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Tag",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific tag in Teamwork Desk by its ID. " +
				"Useful for auditing tag usage, troubleshooting ticket categorization, or " +
				"integrating Desk tag data into automation workflows.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to retrieve.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			tag, err := client.Tags.Get(ctx, arguments.GetInt("id", 0))
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
			Type:        "string",
			Description: "The name of the tag to filter by.",
		},
		"color": {
			Type:        "string",
			Description: "The color of the tag to filter by.",
		},
		"inboxIDs": {
			Type:        "array",
			Description: "The IDs of the inboxes to filter by.",
			Items: &jsonschema.Schema{
				Type: "integer",
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
			Description: "List all tags in Teamwork Desk, with optional filters for name, color, and inbox association. " +
				"Enables users to audit, analyze, or synchronize tag configurations for ticket management, " +
				"reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: properties,
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
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
			Description: "Create a new tag in Teamwork Desk by specifying its name and color. Useful for customizing " +
				"ticket workflows, introducing new categories, or adapting Desk to evolving support processes.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the tag.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the tag.",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			tag, err := client.Tags.Create(ctx, &deskmodels.TagResponse{
				Tag: deskmodels.Tag{
					Name:  arguments.GetString("name", ""),
					Color: arguments.GetString("color", ""),
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
			Description: "Update an existing tag in Teamwork Desk by ID, allowing changes to its name and color. " +
				"Supports evolving support policies, rebranding, or correcting tag attributes for improved " +
				"ticket handling.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name of the tag.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the tag.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError(err.Error()), nil
			}

			_, err = client.Tags.Update(ctx, arguments.GetInt("id", 0), &deskmodels.TagResponse{
				Tag: deskmodels.Tag{
					Name:  arguments.GetString("name", ""),
					Color: arguments.GetString("color", ""),
				},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create tag: %w", err)
			}

			return helpers.NewToolResultText("Tag updated successfully"), nil
		},
	}
}
