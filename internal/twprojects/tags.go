package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

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

const tagDescription = "In the context of Teamwork.com, a tag is a customizable label that can be applied to various " +
	"items such as tasks, projects, milestones, messages, and more, to help categorize and organize work efficiently. " +
	"Tags provide a flexible way to filter, search, and group related items across the platform, making it easier for " +
	"teams to manage complex workflows, highlight priorities, or track themes and statuses. Since tags are " +
	"user-defined, they adapt to each team’s specific needs and can be color-coded for better visual clarity."

var (
	tagGetOutputSchema  *jsonschema.Schema
	tagListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTagCreate)
	toolsets.RegisterMethod(MethodTagUpdate)
	toolsets.RegisterMethod(MethodTagDelete)
	toolsets.RegisterMethod(MethodTagGet)
	toolsets.RegisterMethod(MethodTagList)

	var err error

	// generate the output schemas only once
	tagGetOutputSchema, err = jsonschema.For[projects.TagGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TagGetResponse: %v", err))
	}
	tagListOutputSchema, err = jsonschema.For[projects.TagListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TagListResponse: %v", err))
	}
}

// TagCreate creates a tag in Teamwork.com.
func TagCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTagCreate),
			Description: "Create a new tag in Teamwork.com. " + tagDescription,
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
						Type:        "integer",
						Description: "The ID of the project to associate the tag with. This is for project-scoped tags.",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagCreateRequest projects.TagCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&tagCreateRequest.Name, "name"),
				helpers.OptionalNumericPointerParam(&tagCreateRequest.ProjectID, "project_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Update an existing tag in Teamwork.com. " + tagDescription,
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
						Type:        "string",
						Description: "The name of the tag. It must have less than 50 characters.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to associate the tag with. This is for project-scoped tags.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagUpdateRequest projects.TagUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tagUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&tagUpdateRequest.Name, "name"),
				helpers.OptionalNumericPointerParam(&tagUpdateRequest.ProjectID, "project_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Delete an existing tag in Teamwork.com. " + tagDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Tag",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tagDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Get an existing tag in Teamwork.com. " + tagDescription,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tagGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "List tags in Teamwork.com. " + tagDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tags",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter tags by name. Each word from the search term is used to match " +
							"against the tag name.",
					},
					"item_type": {
						Type: "string",
						Description: "The type of item to filter tags by. Valid values are 'project', 'task', 'tasklist', " +
							"'milestone', 'message', 'timelog', 'notebook', 'file', 'company' and 'link'.",
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
					"project_ids": {
						Type:        "array",
						Description: "A list of project IDs to filter tags by projects",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"page": {
						Type:        "integer",
						Description: "Page number for pagination of results.",
					},
					"page_size": {
						Type:        "integer",
						Description: "Number of results per page for pagination.",
					},
				},
			},
			OutputSchema: tagListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagListRequest projects.TagListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&tagListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&tagListRequest.Filters.ItemType, "item_type",
					helpers.RestrictValues("project", "task", "tasklist", "milestone", "message", "timelog", "notebook",
						"file", "company", "link"),
				),
				helpers.OptionalNumericListParam(&tagListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericParam(&tagListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tagListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tagList, err := projects.TagList(ctx, engine, tagListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tags")
			}
			return helpers.NewToolResultJSON(tagList)
		},
	}
}
