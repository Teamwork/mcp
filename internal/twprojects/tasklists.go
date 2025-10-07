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
	MethodTasklistCreate        toolsets.Method = "twprojects-create_tasklist"
	MethodTasklistUpdate        toolsets.Method = "twprojects-update_tasklist"
	MethodTasklistDelete        toolsets.Method = "twprojects-delete_tasklist"
	MethodTasklistGet           toolsets.Method = "twprojects-get_tasklist"
	MethodTasklistList          toolsets.Method = "twprojects-list_tasklists"
	MethodTasklistListByProject toolsets.Method = "twprojects-list_tasklists_by_project"
)

const tasklistDescription = "In the context of Teamwork.com, a task list is a way to group related tasks within a " +
	"project, helping teams organize their work into meaningful sections such as phases, categories, or deliverables. " +
	"Each task list belongs to a specific project and can include multiple tasks that are typically aligned with a " +
	"common goal. Task lists can be associated with milestones, and they support privacy settings that control who " +
	"can view or interact with the tasks they contain. This structure helps teams manage progress, assign " +
	"responsibilities, and maintain clarity across complex projects."

var (
	tasklistGetOutputSchema  *jsonschema.Schema
	tasklistListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTasklistCreate)
	toolsets.RegisterMethod(MethodTasklistUpdate)
	toolsets.RegisterMethod(MethodTasklistDelete)
	toolsets.RegisterMethod(MethodTasklistGet)
	toolsets.RegisterMethod(MethodTasklistList)
	toolsets.RegisterMethod(MethodTasklistListByProject)

	var err error

	// generate the output schemas only once
	tasklistGetOutputSchema, err = jsonschema.For[projects.TasklistGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TasklistGetResponse: %v", err))
	}
	tasklistListOutputSchema, err = jsonschema.For[projects.TasklistListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TasklistListResponse: %v", err))
	}
}

// TasklistCreate creates a tasklist in Teamwork.com.
func TasklistCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistCreate),
			Description: "Create a new tasklist in Teamwork.com. " + tasklistDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Tasklist",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the tasklist.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to create the tasklist in.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the tasklist.",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone to associate with the tasklist.",
					},
				},
				Required: []string{"name", "project_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistCreateRequest projects.TasklistCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&tasklistCreateRequest.Name, "name"),
				helpers.RequiredNumericParam(&tasklistCreateRequest.Path.ProjectID, "project_id"),
				helpers.OptionalPointerParam(&tasklistCreateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&tasklistCreateRequest.MilestoneID, "milestone_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tasklist, err := projects.TasklistCreate(ctx, engine, tasklistCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create tasklist")
			}
			return helpers.NewToolResultText("Tasklist created successfully with ID %d", tasklist.ID), nil
		},
	}
}

// TasklistUpdate updates a tasklist in Teamwork.com.
func TasklistUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistUpdate),
			Description: "Update an existing tasklist in Teamwork.com. " + tasklistDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Tasklist",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tasklist to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the tasklist.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the tasklist.",
					},
					"milestone_id": {
						Type:        "integer",
						Description: "The ID of the milestone to associate with the tasklist.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistUpdateRequest projects.TasklistUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tasklistUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&tasklistUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&tasklistUpdateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&tasklistUpdateRequest.MilestoneID, "milestone_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TasklistUpdate(ctx, engine, tasklistUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update tasklist")
			}
			return helpers.NewToolResultText("Tasklist updated successfully"), nil
		},
	}
}

// TasklistDelete deletes a tasklist in Teamwork.com.
func TasklistDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistDelete),
			Description: "Delete an existing tasklist in Teamwork.com. " + tasklistDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Tasklist",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tasklist to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistDeleteRequest projects.TasklistDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tasklistDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TasklistDelete(ctx, engine, tasklistDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete tasklist")
			}
			return helpers.NewToolResultText("Tasklist deleted successfully"), nil
		},
	}
}

// TasklistGet retrieves a tasklist in Teamwork.com.
func TasklistGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistGet),
			Description: "Get an existing tasklist in Teamwork.com. " + tasklistDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Tasklist",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the tasklist to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: tasklistGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistGetRequest projects.TasklistGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tasklistGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tasklist, err := projects.TasklistGet(ctx, engine, tasklistGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get tasklist")
			}

			encoded, err := json.Marshal(tasklist)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasklists"),
						)),
					},
				},
				StructuredContent: tasklist,
			}, nil
		},
	}
}

// TasklistList lists tasklists in Teamwork.com.
func TasklistList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistList),
			Description: "List tasklists in Teamwork.com. " + tasklistDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tasklists",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type:        "string",
						Description: "A search term to filter tasklists by name.",
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
			OutputSchema: tasklistListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistListRequest projects.TasklistListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&tasklistListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tasklistList, err := projects.TasklistList(ctx, engine, tasklistListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasklists")
			}

			encoded, err := json.Marshal(tasklistList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasklists"),
						)),
					},
				},
				StructuredContent: tasklistList,
			}, nil
		},
	}
}

// TasklistListByProject lists tasklists in Teamwork.com by project.
func TasklistListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistListByProject),
			Description: "List tasklists in Teamwork.com by project. " + tasklistDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tasklists By Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project from which to retrieve tasklists.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter tasklists by name.",
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
				Required: []string{"project_id"},
			},
			OutputSchema: tasklistListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistListRequest projects.TasklistListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&tasklistListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&tasklistListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tasklistList, err := projects.TasklistList(ctx, engine, tasklistListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasklists")
			}

			encoded, err := json.Marshal(tasklistList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasklists"),
						)),
					},
				},
				StructuredContent: tasklistList,
			}, nil
		},
	}
}
