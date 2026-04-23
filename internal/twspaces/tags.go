package twspaces

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	spacesmodels "github.com/teamwork/spacessdkgo/models"
)

// List of tag methods available in the Teamwork Spaces MCP service.
const (
	MethodTagCreateBatch toolsets.Method = "twspaces-create_tags"
	MethodTagUpdate      toolsets.Method = "twspaces-update_tag"
	MethodTagDelete      toolsets.Method = "twspaces-delete_tag"
	MethodTagGet         toolsets.Method = "twspaces-get_tag"
	MethodTagList        toolsets.Method = "twspaces-list_tags"
)

// TagGet retrieves a single tag by ID.
func TagGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Tag",
				ReadOnlyHint: true,
			},
			Description: "Retrieve a specific tag in Teamwork Spaces by its ID.",
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
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			tag, err := client.Tags.Get(ctx, int64(arguments.GetInt("id", 0)))
			if err != nil {
				return nil, fmt.Errorf("failed to get tag: %w", err)
			}
			return helpers.NewToolResultJSON(tag)
		},
	}
}

// TagList lists all tags.
func TagList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tags",
				ReadOnlyHint: true,
			},
			Description: "List all tags in Teamwork Spaces. Tags can be applied to pages for categorization " +
				"and filtering.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
		},
		Handler: func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)

			tags, err := client.Tags.List(ctx, url.Values{})
			if err != nil {
				return nil, fmt.Errorf("failed to list tags: %w", err)
			}
			return helpers.NewToolResultJSON(tags)
		},
	}
}

// TagCreateBatch creates one or more tags in a single request.
func TagCreateBatch(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagCreateBatch),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Tags",
			},
			Description: "Create one or more tags in Teamwork Spaces in a single request. Tags can then be " +
				"applied to pages for organization and filtering.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tags": {
						Type:        "array",
						Description: "An array of tags to create.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"name": {
									Type:        "string",
									Description: "The name of the tag.",
								},
								"color": {
									Type:        "string",
									Description: "A hex color code for the tag (e.g. \"#FF5733\").",
								},
							},
							Required: []string{"name"},
						},
					},
				},
				Required: []string{"tags"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			rawTags, ok := arguments["tags"]
			if !ok {
				return helpers.NewToolResultTextError("tags argument is required"), nil
			}

			tagsList, ok := rawTags.([]any)
			if !ok {
				return helpers.NewToolResultTextError("tags must be an array"), nil
			}

			tags := make([]spacesmodels.Tag, 0, len(tagsList))
			for _, raw := range tagsList {
				tagMap, ok := raw.(map[string]any)
				if !ok {
					continue
				}
				tag := spacesmodels.Tag{}
				if name, ok := tagMap["name"].(string); ok {
					tag.Name = name
				}
				if color, ok := tagMap["color"].(string); ok {
					tag.Color = color
				}
				tags = append(tags, tag)
			}

			created, err := client.Tags.CreateBatch(ctx, tags)
			if err != nil {
				return nil, fmt.Errorf("failed to create tags: %w", err)
			}
			return helpers.NewToolResultJSON(created)
		},
	}
}

// TagUpdate updates an existing tag.
func TagUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Tag",
			},
			Description: "Update an existing tag in Teamwork Spaces by ID. Supports changing the name and color.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name for the tag.",
					},
					"color": {
						Type:        "string",
						Description: "A new hex color code for the tag.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.TagUpdate{}

			if name := arguments.GetString("name", ""); name != "" {
				req.Name = &name
			}
			if color := arguments.GetString("color", ""); color != "" {
				req.Color = &color
			}

			tag, err := client.Tags.Update(ctx, int64(arguments.GetInt("id", 0)), req)
			if err != nil {
				return nil, fmt.Errorf("failed to update tag: %w", err)
			}
			return helpers.NewToolResultJSON(tag)
		},
	}
}

// TagDelete deletes a tag by ID.
func TagDelete(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTagDelete),
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Tag",
			},
			Description: "Delete a tag in Teamwork Spaces by its ID.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			if err := client.Tags.Delete(ctx, int64(arguments.GetInt("id", 0))); err != nil {
				return nil, fmt.Errorf("failed to delete tag: %w", err)
			}
			return helpers.NewToolResultText("Tag deleted successfully"), nil
		},
	}
}
