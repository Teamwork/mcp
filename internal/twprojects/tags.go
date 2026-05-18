package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTagCreate toolsets.Method = "twprojects-create_tag"
	MethodTagUpdate toolsets.Method = "twprojects-update_tag"
	MethodTagDelete toolsets.Method = "twprojects-delete_tag"
	MethodTagGet    toolsets.Method = "twprojects-get_tag"
	MethodTagList   toolsets.Method = "twprojects-list_tags"
)

var (
	tagGetOutputSchema  *jsonschema.Schema
	tagListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	tagGetOutputSchema, err = jsonschema.For[projects.TagGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TagGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(tagGetOutputSchema)
	tagListOutputSchema, err = jsonschema.For[projects.TagListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TagListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(tagListOutputSchema)
}

// TagCreate creates a tag in Teamwork.com.
func TagCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTagCreate),
			Description: "Create tag.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Tag",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the tag. It must have less than 50 characters.",
					},
					"project_id": {
						Description: "The ID of the project to associate the tag with. This is for project-scoped tags.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagCreateRequest projects.TagCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&tagCreateRequest.Name, "name"),
				helpers.OptionalNumericPointerParam(&tagCreateRequest.ProjectID, "project_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			tagResponse, err := projects.TagCreate(ctx, engine, tagCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create tag")
			}
			return helpers.NewToolResultText("Tag created successfully with ID %d", tagResponse.Tag.ID), nil
		},
	}
}

// TagUpdate updates a tag in Teamwork.com.
func TagUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTagUpdate),
			Description: "Update tag.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Tag",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to update.",
					},
					"name": {
						Description: "The name of the tag. It must have less than 50 characters.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to associate the tag with. This is for project-scoped tags.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagUpdateRequest projects.TagUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tagUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&tagUpdateRequest.Name, "name"),
				helpers.OptionalNumericPointerParam(&tagUpdateRequest.ProjectID, "project_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TagUpdate(ctx, engine, tagUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update tag")
			}
			return helpers.NewToolResultText("Tag updated successfully"), nil
		},
	}
}

// TagDelete deletes a tag in Teamwork.com.
func TagDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTagDelete),
			Description: "Delete tag.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Tag",
				DestructiveHint: new(true),
			},
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
			var tagDeleteRequest projects.TagDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tagDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TagDelete(ctx, engine, tagDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete tag")
			}
			return helpers.NewToolResultText("Tag deleted successfully"), nil
		},
	}
}

// TagGet retrieves a tag in Teamwork.com.
func TagGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTagGet),
			Description: "Get tag.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Tag",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tag to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: tagGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagGetRequest projects.TagGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tagGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			tag, err := projects.TagGet(ctx, engine, tagGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get tag")
			}
			return helpers.NewToolResultJSON(tag)
		},
	}
}

// TagList lists tags in Teamwork.com.
func TagList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTagList),
			Description: "List tags.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tags",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter tags by name. Each word from the search term is used to match " +
							"against the tag name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"item_type": {
						Description: "Filter tags by item type.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "string",
								Enum: []any{
									"project",
									"task",
									"tasklist",
									"milestone",
									"message",
									"timelog",
									"notebook",
									"file",
									"company",
									"link",
								},
							},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "Filter by project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(tagListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagListRequest projects.TagListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&tagListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&tagListRequest.Filters.ItemType, "item_type",
					helpers.RestrictValues("project", "task", "tasklist", "milestone", "message", "timelog", "notebook",
						"file", "company", "link"),
				),
				helpers.OptionalNumericListParam(&tagListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericParam(&tagListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tagListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				tagListRequest.Filters.Fields.Tags = []projects.TagField{
					projects.TagFieldID,
					projects.TagFieldName,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, tagListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tags")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list tags"), "failed to list tags")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(body)},
				},
			}
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
