package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"

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
	MethodProjectCategoryCreate toolsets.Method = "twprojects-create_project_category"
	MethodProjectCategoryUpdate toolsets.Method = "twprojects-update_project_category"
	MethodProjectCategoryDelete toolsets.Method = "twprojects-delete_project_category"
	MethodProjectCategoryGet    toolsets.Method = "twprojects-get_project_category"
	MethodProjectCategoryList   toolsets.Method = "twprojects-list_project_categories"
)

const projectCategoryDescription = "The project category is a way to group and label related projects so teams can " +
	"organize their work more clearly across the platform. By assigning a category, you create a higher-level " +
	"structure that makes it easier to filter, report on, and navigate multiple projects, ensuring that departments, " +
	"workflows, or strategic areas remain neatly aligned and easier to manage."

var (
	projectCategoryGetOutputSchema  *jsonschema.Schema
	projectCategoryListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodProjectCategoryCreate)
	toolsets.RegisterMethod(MethodProjectCategoryUpdate)
	toolsets.RegisterMethod(MethodProjectCategoryDelete)
	toolsets.RegisterMethod(MethodProjectCategoryGet)
	toolsets.RegisterMethod(MethodProjectCategoryList)

	var err error

	// generate the output schemas only once
	projectCategoryGetOutputSchema, err = jsonschema.For[projects.ProjectCategoryGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ProjectCategoryGetResponse: %v", err))
	}
	projectCategoryListOutputSchema, err = jsonschema.For[projects.ProjectCategoryListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ProjectCategoryListResponse: %v", err))
	}
}

// ProjectCategoryCreate creates a project category in Teamwork.com.
func ProjectCategoryCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectCategoryCreate),
			Description: "Create a new project category in Teamwork.com. " + projectCategoryDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Project Category",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the project category.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the project category in hex format (e.g., #FF5733).",
					},
					"parent_id": {
						Type:        "integer",
						Description: "The ID of the parent project category, if any. This allows for nested categories.",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCategoryCreateRequest projects.ProjectCategoryCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&projectCategoryCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectCategoryCreateRequest.Color, "color"),
				helpers.OptionalNumericPointerParam(&projectCategoryCreateRequest.ParentID, "parent_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			projectCategory, err := projects.ProjectCategoryCreate(ctx, engine, projectCategoryCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create project category")
			}
			return helpers.NewToolResultText("Project category created successfully with ID %d", projectCategory.ID), nil
		},
	}
}

// ProjectCategoryUpdate updates a project category in Teamwork.com.
func ProjectCategoryUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectCategoryUpdate),
			Description: "Update an existing project category in Teamwork.com. " + projectCategoryDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Project Category",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project category to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the project category.",
					},
					"color": {
						Type:        "string",
						Description: "The color of the project category in hex format (e.g., #FF5733).",
					},
					"parent_id": {
						Type:        "integer",
						Description: "The ID of the parent project category, if any. This allows for nested categories.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCategoryUpdateRequest projects.ProjectCategoryUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectCategoryUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&projectCategoryUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectCategoryUpdateRequest.Color, "color"),
				helpers.OptionalNumericPointerParam(&projectCategoryUpdateRequest.ParentID, "parent_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.ProjectCategoryUpdate(ctx, engine, projectCategoryUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update project category")
			}
			return helpers.NewToolResultText("Project category updated successfully"), nil
		},
	}
}

// ProjectCategoryDelete deletes a project category in Teamwork.com.
func ProjectCategoryDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectCategoryDelete),
			Description: "Delete an existing project category in Teamwork.com. " + projectCategoryDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Project Category",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project category to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCategoryDeleteRequest projects.ProjectCategoryDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectCategoryDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.ProjectCategoryDelete(ctx, engine, projectCategoryDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete project category")
			}
			return helpers.NewToolResultText("Project category deleted successfully"), nil
		},
	}
}

// ProjectCategoryGet retrieves a project category in Teamwork.com.
func ProjectCategoryGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectCategoryGet),
			Description: "Get an existing project category in Teamwork.com. " + projectCategoryDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Project Category",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project category to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: projectCategoryGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCategoryGetRequest projects.ProjectCategoryGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectCategoryGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			projectCategory, err := projects.ProjectCategoryGet(ctx, engine, projectCategoryGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get project category")
			}

			encoded, err := json.Marshal(projectCategory)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, projectCategoryPathBuilder)),
					},
				},
				StructuredContent: projectCategory,
			}, nil
		},
	}
}

// ProjectCategoryList lists project categories in Teamwork.com.
func ProjectCategoryList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectCategoryList),
			Description: "List project categories in Teamwork.com. " + projectCategoryDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Project Categories",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type:        "string",
						Description: "A search term to filter project categories by name.",
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
			OutputSchema: projectCategoryListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCategoryListRequest projects.ProjectCategoryListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&projectCategoryListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&projectCategoryListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&projectCategoryListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			projectCategoryList, err := projects.ProjectCategoryList(ctx, engine, projectCategoryListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list projects")
			}

			encoded, err := json.Marshal(projectCategoryList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded, projectCategoryPathBuilder)),
					},
				},
				StructuredContent: projectCategoryList,
			}, nil
		},
	}
}

func projectCategoryPathBuilder(object map[string]any) string {
	id, ok := object["id"]
	if !ok {
		return ""
	}
	if id == reflect.Zero(reflect.TypeOf(id)).Interface() {
		return ""
	}
	// round float64 IDs to int64 to avoid decimal points in URLs
	if numeric, ok := id.(float64); ok && math.Trunc(numeric) == numeric {
		id = int64(numeric)
	}
	return fmt.Sprintf("app/projects/list?catid=%v", id)
}
