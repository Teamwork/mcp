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
	MethodTaskCreate         toolsets.Method = "twprojects-create_task"
	MethodTaskUpdate         toolsets.Method = "twprojects-update_task"
	MethodTaskDelete         toolsets.Method = "twprojects-delete_task"
	MethodTaskGet            toolsets.Method = "twprojects-get_task"
	MethodTaskList           toolsets.Method = "twprojects-list_tasks"
	MethodTaskListByTasklist toolsets.Method = "twprojects-list_tasks_by_tasklist"
	MethodTaskListByProject  toolsets.Method = "twprojects-list_tasks_by_project"
)

const taskDescription = "In Teamwork.com, a task represents an individual unit of work assigned to one or more team " +
	"members within a project. Each task can include details such as a title, description, priority, estimated time, " +
	"assignees, and due date, along with the ability to attach files, leave comments, track time, and set dependencies " +
	"on other tasks. Tasks are organized within task lists, helping structure and sequence work logically. They serve " +
	"as the building blocks of project management in Teamwork, allowing teams to collaborate, monitor progress, and " +
	"ensure accountability throughout the project's lifecycle."

var (
	taskGetOutputSchema  *jsonschema.Schema
	taskListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTaskCreate)
	toolsets.RegisterMethod(MethodTaskUpdate)
	toolsets.RegisterMethod(MethodTaskDelete)
	toolsets.RegisterMethod(MethodTaskGet)
	toolsets.RegisterMethod(MethodTaskList)
	toolsets.RegisterMethod(MethodTaskListByTasklist)
	toolsets.RegisterMethod(MethodTaskListByProject)

	var err error

	// generate the output schemas only once
	taskGetOutputSchema, err = jsonschema.For[projects.TaskGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TaskGetResponse: %v", err))
	}
	taskListOutputSchema, err = jsonschema.For[projects.TaskListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TaskListResponse: %v", err))
	}
}

// TaskCreate creates a task in Teamwork.com.
func TaskCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskCreate),
			Description: "Create a new task in Teamwork.com. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Task",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the task.",
					},
					"tasklist_id": {
						Type:        "integer",
						Description: "The ID of the tasklist.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the task.",
					},
					"priority": {
						Type:        "string",
						Description: "The priority of the task. Possible values are: low, medium, high.",
						Enum:        []any{"low", "medium", "high"},
					},
					"progress": {
						Type:        "integer",
						Description: "The progress of the task, as a percentage (0-100). Only whole numbers are allowed.",
						Minimum:     twapi.Ptr(float64(0)),
						Maximum:     twapi.Ptr(float64(100)),
					},
					"start_date": {
						Type:        "string",
						Format:      "date",
						Description: "The start date of the task in ISO 8601 format (YYYY-MM-DD).",
					},
					"due_date": {
						Type:   "string",
						Format: "date",
						Description: "The due date of the task in ISO 8601 format (YYYY-MM-DD). When this is not provided, it " +
							"will fallback to the milestone due date if a milestone is set.",
					},
					"estimated_minutes": {
						Type:        "integer",
						Description: "The estimated time to complete the task in minutes.",
					},
					"parent_task_id": {
						Type:        "integer",
						Description: "The ID of the parent task if creating a subtask.",
					},
					"assignees": {
						Type:        "object",
						Description: "An object containing assignees for the task.",
						Properties: map[string]*jsonschema.Schema{
							"user_ids": {
								Type:        "array",
								Description: "List of user IDs assigned to the task.",
								Items:       &jsonschema.Schema{Type: "integer"},
								MinItems:    twapi.Ptr(1),
							},
							"company_ids": {
								Type:        "array",
								Description: "List of company IDs assigned to the task.",
								Items:       &jsonschema.Schema{Type: "integer"},
								MinItems:    twapi.Ptr(1),
							},
							"team_ids": {
								Type:        "array",
								Description: "List of team IDs assigned to the task.",
								Items:       &jsonschema.Schema{Type: "integer"},
								MinItems:    twapi.Ptr(1),
							},
						},
						MinProperties: twapi.Ptr(1),
						MaxProperties: twapi.Ptr(3),
						AnyOf: []*jsonschema.Schema{
							{Required: []string{"user_ids"}},
							{Required: []string{"company_ids"}},
							{Required: []string{"team_ids"}},
						},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to assign to the task.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"predecessors": {
						Type: "array",
						Description: "List of task dependencies that must be completed before this task can start, defining its " +
							"position in the project workflow and ensuring proper sequencing of work.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"task_id": {
									Type:        "integer",
									Description: "The ID of the predecessor task.",
								},
								"type": {
									Type: "string",
									Description: "The type of dependency. Possible values are: start or complete. 'start' means this " +
										"task can complete when the predecessor starts, 'complete' means this task can complete when " +
										"the predecessor completes.",
									Enum: []any{"start", "complete"},
								},
							},
						},
					},
				},
				Required: []string{"name", "tasklist_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskCreateRequest projects.TaskCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&taskCreateRequest.Name, "name"),
				helpers.RequiredNumericParam(&taskCreateRequest.Path.TasklistID, "tasklist_id"),
				helpers.OptionalPointerParam(&taskCreateRequest.Description, "description"),
				helpers.OptionalPointerParam(&taskCreateRequest.Priority, "priority",
					helpers.RestrictValues("low", "medium", "high"),
				),
				helpers.OptionalNumericPointerParam(&taskCreateRequest.Progress, "progress"),
				helpers.OptionalDatePointerParam(&taskCreateRequest.StartAt, "start_date"),
				helpers.OptionalDatePointerParam(&taskCreateRequest.DueAt, "due_date"),
				helpers.OptionalNumericPointerParam(&taskCreateRequest.EstimatedMinutes, "estimated_minutes"),
				helpers.OptionalNumericPointerParam(&taskCreateRequest.ParentTaskID, "parent_task_id"),
				helpers.OptionalNumericListParam(&taskCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			if assignees, ok := arguments["assignees"]; ok {
				assigneesMap, ok := assignees.(map[string]any)
				if !ok {
					return helpers.NewToolResultTextError("invalid assignees"), nil
				} else if assigneesMap != nil {
					taskCreateRequest.Assignees = new(projects.UserGroups)

					err = helpers.ParamGroup(assigneesMap,
						helpers.OptionalNumericListParam(&taskCreateRequest.Assignees.UserIDs, "user_ids"),
						helpers.OptionalNumericListParam(&taskCreateRequest.Assignees.CompanyIDs, "company_ids"),
						helpers.OptionalNumericListParam(&taskCreateRequest.Assignees.TeamIDs, "team_ids"),
					)
					if err != nil {
						return helpers.NewToolResultTextError(fmt.Sprintf("invalid assignees: %s", err)), nil
					}
				}
			}

			if predecessors, ok := arguments["predecessors"]; ok {
				predecessorsSlice, ok := predecessors.([]any)
				if !ok {
					return helpers.NewToolResultTextError("invalid predecessors"), nil
				}

				for _, predecessor := range predecessorsSlice {
					predecessorMap, ok := predecessor.(map[string]any)
					if !ok {
						return helpers.NewToolResultTextError("invalid predecessors"), nil
					}

					var p projects.TaskPredecessor
					err = helpers.ParamGroup(predecessorMap,
						helpers.RequiredNumericParam(&p.ID, "task_id"),
						helpers.RequiredParam(&p.Type, "type",
							helpers.RestrictValues(
								projects.TaskPredecessorTypeStart,
								projects.TaskPredecessorTypeFinish,
							),
						),
					)
					if err != nil {
						return helpers.NewToolResultTextError(fmt.Sprintf("invalid predecessor: %s", err)), nil
					}

					taskCreateRequest.Predecessors = append(taskCreateRequest.Predecessors, p)
				}
			}

			taskResponse, err := projects.TaskCreate(ctx, engine, taskCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create task")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Task created successfully with ID %d", taskResponse.Task.ID),
					},
				},
			}, nil
		},
	}
}

// TaskUpdate updates a task in Teamwork.com.
func TaskUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskUpdate),
			Description: "Update an existing task in Teamwork.com. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Task",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the task to update.",
					},
					"tasklist_id": {
						Type:        "integer",
						Description: "The ID of the tasklist.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the task.",
					},
					"priority": {
						Type:        "string",
						Description: "The priority of the task. Possible values are: low, medium, high.",
						Enum:        []any{"low", "medium", "high"},
					},
					"progress": {
						Type:        "integer",
						Description: "The progress of the task, as a percentage (0-100). Only whole numbers are allowed.",
						Minimum:     twapi.Ptr(float64(0)),
						Maximum:     twapi.Ptr(float64(100)),
					},
					"start_date": {
						Type:        "string",
						Format:      "date",
						Description: "The start date of the task in ISO 8601 format (YYYY-MM-DD).",
					},
					"due_date": {
						Type:   "string",
						Format: "date",
						Description: "The due date of the task in ISO 8601 format (YYYY-MM-DD). When this is not provided, it " +
							"will fallback to the milestone due date if a milestone is set.",
					},
					"estimated_minutes": {
						Type:        "integer",
						Description: "The estimated time to complete the task in minutes.",
					},
					"parent_task_id": {
						Type:        "integer",
						Description: "The ID of the parent task if creating a subtask.",
					},
					"assignees": {
						Type:        "object",
						Description: "An object containing assignees for the task.",
						Properties: map[string]*jsonschema.Schema{
							"user_ids": {
								Type:        "array",
								Description: "List of user IDs assigned to the task.",
								Items:       &jsonschema.Schema{Type: "integer"},
								MinItems:    twapi.Ptr(1),
							},
							"company_ids": {
								Type:        "array",
								Description: "List of company IDs assigned to the task.",
								Items:       &jsonschema.Schema{Type: "integer"},
								MinItems:    twapi.Ptr(1),
							},
							"team_ids": {
								Type:        "array",
								Description: "List of team IDs assigned to the task.",
								Items:       &jsonschema.Schema{Type: "integer"},
								MinItems:    twapi.Ptr(1),
							},
						},
						MinProperties: twapi.Ptr(1),
						MaxProperties: twapi.Ptr(3),
						AnyOf: []*jsonschema.Schema{
							{Required: []string{"user_ids"}},
							{Required: []string{"company_ids"}},
							{Required: []string{"team_ids"}},
						},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to assign to the task.",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"predecessors": {
						Type: "array",
						Description: "List of task dependencies that must be completed before this task can start, defining its " +
							"position in the project workflow and ensuring proper sequencing of work.",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"task_id": {
									Type:        "integer",
									Description: "The ID of the predecessor task.",
								},
								"type": {
									Type: "string",
									Description: "The type of dependency. Possible values are: start or complete. 'start' means this " +
										"task can complete when the predecessor starts, 'complete' means this task can complete when the " +
										"predecessor completes.",
									Enum: []any{"start", "complete"},
								},
							},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskUpdateRequest projects.TaskUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskUpdateRequest.Path.ID, "id"),
				helpers.OptionalNumericPointerParam(&taskUpdateRequest.TasklistID, "tasklist_id"),
				helpers.OptionalPointerParam(&taskUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&taskUpdateRequest.Description, "description"),
				helpers.OptionalPointerParam(&taskUpdateRequest.Priority, "priority",
					helpers.RestrictValues("low", "medium", "high"),
				),
				helpers.OptionalNumericPointerParam(&taskUpdateRequest.Progress, "progress"),
				helpers.OptionalDatePointerParam(&taskUpdateRequest.StartAt, "start_date"),
				helpers.OptionalDatePointerParam(&taskUpdateRequest.DueAt, "due_date"),
				helpers.OptionalNumericPointerParam(&taskUpdateRequest.EstimatedMinutes, "estimated_minutes"),
				helpers.OptionalNumericPointerParam(&taskUpdateRequest.ParentTaskID, "parent_task_id"),
				helpers.OptionalNumericListParam(&taskUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			if assignees, ok := arguments["assignees"]; ok {
				assigneesMap, ok := assignees.(map[string]any)
				if !ok {
					return helpers.NewToolResultTextError("invalid assignees"), nil
				} else if assigneesMap != nil {
					taskUpdateRequest.Assignees = new(projects.UserGroups)

					err = helpers.ParamGroup(assigneesMap,
						helpers.OptionalNumericListParam(&taskUpdateRequest.Assignees.UserIDs, "user_ids"),
						helpers.OptionalNumericListParam(&taskUpdateRequest.Assignees.CompanyIDs, "company_ids"),
						helpers.OptionalNumericListParam(&taskUpdateRequest.Assignees.TeamIDs, "team_ids"),
					)
					if err != nil {
						return helpers.NewToolResultTextError(fmt.Sprintf("invalid assignees: %s", err)), nil
					}
				}
			}

			if predecessors, ok := arguments["predecessors"]; ok {
				predecessorsSlice, ok := predecessors.([]any)
				if !ok {
					return helpers.NewToolResultTextError("invalid predecessors"), nil
				}

				for _, predecessor := range predecessorsSlice {
					predecessorMap, ok := predecessor.(map[string]any)
					if !ok {
						return helpers.NewToolResultTextError("invalid predecessors"), nil
					}

					var p projects.TaskPredecessor
					err = helpers.ParamGroup(predecessorMap,
						helpers.RequiredNumericParam(&p.ID, "task_id"),
						helpers.RequiredParam(&p.Type, "type",
							helpers.RestrictValues(
								projects.TaskPredecessorTypeStart,
								projects.TaskPredecessorTypeFinish,
							),
						),
					)
					if err != nil {
						return helpers.NewToolResultTextError(fmt.Sprintf("invalid predecessor: %s", err)), nil
					}

					taskUpdateRequest.Predecessors = append(taskUpdateRequest.Predecessors, p)
				}
			}

			_, err = projects.TaskUpdate(ctx, engine, taskUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update task")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Task updated successfully",
					},
				},
			}, nil
		},
	}
}

// TaskDelete deletes a task in Teamwork.com.
func TaskDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskDelete),
			Description: "Delete an existing task in Teamwork.com. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Task",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the task to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskDeleteRequest projects.TaskDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TaskDelete(ctx, engine, taskDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete task")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Task deleted successfully",
					},
				},
			}, nil
		},
	}
}

// TaskGet retrieves a task in Teamwork.com.
func TaskGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskGet),
			Description: "Get an existing task in Teamwork.com. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Task",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the task to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: taskGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskGetRequest projects.TaskGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			task, err := projects.TaskGet(ctx, engine, taskGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get task")
			}

			encoded, err := json.Marshal(task)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
						)),
					},
				},
			}, nil
		},
	}
}

// TaskList lists tasks in Teamwork.com.
func TaskList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskList),
			Description: "List tasks in Teamwork.com. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tasks",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type:        "string",
						Description: "A search term to filter tasks by name.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter tasks by tags",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"assignee_user_ids": {
						Type:        "array",
						Description: "A list of user IDs to filter tasks by assigned users",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match tasks that have all the specified tags. If false, the " +
							"search will match tasks that have any of the specified tags. Defaults to false.",
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
			OutputSchema: taskListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.AssigneeUserIDs, "assignee_user_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			taskList, err := projects.TaskList(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}

			encoded, err := json.Marshal(taskList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
						)),
					},
				},
			}, nil
		},
	}
}

// TaskListByTasklist lists tasks in Teamwork.com by tasklist.
func TaskListByTasklist(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskListByTasklist),
			Description: "List tasks in Teamwork.com by tasklist. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tasks By Tasklist",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tasklist_id": {
						Type:        "integer",
						Description: "The ID of the tasklist from which to retrieve tasks.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter tasks by name.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter tasks by tags",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"assignee_user_ids": {
						Type:        "array",
						Description: "A list of user IDs to filter tasks by assigned users",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match tasks that have all the specified tags. If false, the " +
							"search will match tasks that have any of the specified tags. Defaults to false.",
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
				Required: []string{"tasklist_id"},
			},
			OutputSchema: taskListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskListRequest.Path.TasklistID, "tasklist_id"),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.AssigneeUserIDs, "assignee_user_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			taskList, err := projects.TaskList(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}

			encoded, err := json.Marshal(taskList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
						)),
					},
				},
			}, nil
		},
	}
}

// TaskListByProject lists tasks in Teamwork.com by project.
func TaskListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskListByProject),
			Description: "List tasks in Teamwork.com by project. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tasks By Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project from which to retrieve tasks.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter tasks by name.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter tasks by tags",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"assignee_user_ids": {
						Type:        "array",
						Description: "A list of user IDs to filter tasks by assigned users",
						Items:       &jsonschema.Schema{Type: "integer"},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match tasks that have all the specified tags. If false, the " +
							"search will match tasks that have any of the specified tags. Defaults to false.",
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
			OutputSchema: taskListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.AssigneeUserIDs, "assignee_user_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			taskList, err := projects.TaskList(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}

			encoded, err := json.Marshal(taskList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
						)),
					},
				},
			}, nil
		},
	}
}
