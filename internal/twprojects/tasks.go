package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTaskCreate)
	toolsets.RegisterMethod(MethodTaskUpdate)
	toolsets.RegisterMethod(MethodTaskDelete)
	toolsets.RegisterMethod(MethodTaskGet)
	toolsets.RegisterMethod(MethodTaskList)
	toolsets.RegisterMethod(MethodTaskListByTasklist)
	toolsets.RegisterMethod(MethodTaskListByProject)
}

// TaskCreate creates a task in Teamwork.com.
func TaskCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskCreate),
			mcp.WithDescription("Create a new task in Teamwork.com. "+taskDescription),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the task."),
			),
			mcp.WithNumber("tasklist_id",
				mcp.Required(),
				mcp.Description("The ID of the tasklist."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the task."),
			),
			mcp.WithString("priority",
				mcp.Description("The priority of the task. Possible values are: low, medium, high."),
			),
			mcp.WithNumber("progress",
				mcp.Description("The progress of the task, as a percentage (0-100). Only whole numbers are allowed."),
			),
			mcp.WithString("start_date",
				mcp.Description("The start date of the task in ISO 8601 format (YYYY-MM-DD)."),
			),
			mcp.WithString("due_date",
				mcp.Description("The due date of the task in ISO 8601 format (YYYY-MM-DD). When this is not provided, it "+
					"will fallback to the milestone due date if a milestone is set."),
			),
			mcp.WithNumber("estimated_minutes",
				mcp.Description("The estimated time to complete the task in minutes."),
			),
			mcp.WithNumber("parent_task_id",
				mcp.Description("The ID of the parent task if creating a subtask."),
			),
			mcp.WithObject("assignees",
				mcp.Description("An object containing assignees for the task."),
				mcp.Properties(map[string]any{
					"user_ids": map[string]any{
						"type":        "array",
						"description": "List of user IDs assigned to the task.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"company_ids": map[string]any{
						"type":        "array",
						"description": "List of company IDs assigned to the task.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"team_ids": map[string]any{
						"type":        "array",
						"description": "List of team IDs assigned to the task.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
				}),
				mcp.AdditionalProperties(false),
				func(schemaMap map[string]any) {
					schemaMap["minProperties"] = 1
					schemaMap["maxProperties"] = 3
					schemaMap["anyOf"] = []map[string]any{
						{"required": []string{"user_ids"}},
						{"required": []string{"company_ids"}},
						{"required": []string{"team_ids"}},
					}
				},
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to assign to the task."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("predecessors",
				mcp.Description("List of task dependencies that must be completed before this task can start, defining its "+
					"position in the project workflow and ensuring proper sequencing of work."),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task_id": map[string]any{
							"type":        "integer",
							"description": "The ID of the predecessor task.",
						},
						"type": map[string]any{
							"type": "string",
							"description": "The type of dependency. Possible values are: start or complete. 'start' means this " +
								"task can complete when the predecessor starts, 'complete' means this task can complete when the " +
								"predecessor is completed.",
							"enum": []string{"start", "complete"},
						},
					},
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskCreateRequest projects.TaskCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
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
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			if assignees, ok := request.GetArguments()["assignees"]; ok {
				assigneesMap, ok := assignees.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid assignees")
				} else if assigneesMap != nil {
					taskCreateRequest.Assignees = new(projects.UserGroups)

					err = helpers.ParamGroup(assigneesMap,
						helpers.OptionalNumericListParam(&taskCreateRequest.Assignees.UserIDs, "user_ids"),
						helpers.OptionalNumericListParam(&taskCreateRequest.Assignees.CompanyIDs, "company_ids"),
						helpers.OptionalNumericListParam(&taskCreateRequest.Assignees.TeamIDs, "team_ids"),
					)
					if err != nil {
						return nil, fmt.Errorf("invalid assignees: %w", err)
					}
				}
			}

			if predecessors, ok := request.GetArguments()["predecessors"]; ok {
				predecessorsSlice, ok := predecessors.([]any)
				if !ok {
					return nil, fmt.Errorf("invalid predecessors")
				}

				for _, predecessor := range predecessorsSlice {
					predecessorMap, ok := predecessor.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("invalid predecessor")
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
						return nil, fmt.Errorf("invalid predecessor: %w", err)
					}

					taskCreateRequest.Predecessors = append(taskCreateRequest.Predecessors, p)
				}
			}

			taskResponse, err := projects.TaskCreate(ctx, engine, taskCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create task")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Task created successfully with ID %d", taskResponse.Task.ID)), nil
		},
	}
}

// TaskUpdate updates a task in Teamwork.com.
func TaskUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskUpdate),
			mcp.WithDescription("Update an existing task in Teamwork.com. "+taskDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the task to update."),
			),
			mcp.WithNumber("tasklist_id",
				mcp.Description("The ID of the tasklist. When provided, the task will be moved to this tasklist."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the task."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the task."),
			),
			mcp.WithString("priority",
				mcp.Description("The priority of the task. Possible values are: low, medium, high."),
			),
			mcp.WithNumber("progress",
				mcp.Description("The progress of the task, as a percentage (0-100). Only whole numbers are allowed."),
			),
			mcp.WithString("start_date",
				mcp.Description("The start date of the task in ISO 8601 format (YYYY-MM-DD)."),
			),
			mcp.WithString("due_date",
				mcp.Description("The due date of the task in ISO 8601 format (YYYY-MM-DD). When this is not provided, it "+
					"will fallback to the milestone due date if a milestone is set."),
			),
			mcp.WithNumber("estimated_minutes",
				mcp.Description("The estimated time to complete the task in minutes."),
			),
			mcp.WithNumber("parent_task_id",
				mcp.Description("The ID of the parent task if creating a subtask."),
			),
			mcp.WithObject("assignees",
				mcp.Description("An object containing assignees for the task."),
				mcp.Properties(map[string]any{
					"user_ids": map[string]any{
						"type":        "array",
						"description": "List of user IDs assigned to the task.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"company_ids": map[string]any{
						"type":        "array",
						"description": "List of company IDs assigned to the task.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"team_ids": map[string]any{
						"type":        "array",
						"description": "List of team IDs assigned to the task.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
				}),
				mcp.AdditionalProperties(false),
				func(schemaMap map[string]any) {
					schemaMap["minProperties"] = 1
					schemaMap["maxProperties"] = 3
					schemaMap["anyOf"] = []map[string]any{
						{"required": []string{"user_ids"}},
						{"required": []string{"company_ids"}},
						{"required": []string{"team_ids"}},
					}
				},
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to assign to the task."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("predecessors",
				mcp.Description("List of task dependencies that must be completed before this task can start, defining its "+
					"position in the project workflow and ensuring proper sequencing of work."),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"task_id": map[string]any{
							"type":        "integer",
							"description": "The ID of the predecessor task.",
						},
						"type": map[string]any{
							"type": "string",
							"description": "The type of dependency. Possible values are: start or complete. 'start' means this " +
								"task can complete when the predecessor starts, 'complete' means this task can complete when the " +
								"predecessor is completed.",
							"enum": []string{"start", "complete"},
						},
					},
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskUpdateRequest projects.TaskUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
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
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			if assignees, ok := request.GetArguments()["assignees"]; ok {
				assigneesMap, ok := assignees.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid assignees")
				} else if assigneesMap != nil {
					taskUpdateRequest.Assignees = new(projects.UserGroups)

					err = helpers.ParamGroup(assigneesMap,
						helpers.OptionalNumericListParam(&taskUpdateRequest.Assignees.UserIDs, "user_ids"),
						helpers.OptionalNumericListParam(&taskUpdateRequest.Assignees.CompanyIDs, "company_ids"),
						helpers.OptionalNumericListParam(&taskUpdateRequest.Assignees.TeamIDs, "team_ids"),
					)
					if err != nil {
						return nil, fmt.Errorf("invalid assignees: %w", err)
					}
				}
			}

			if predecessors, ok := request.GetArguments()["predecessors"]; ok {
				predecessorsSlice, ok := predecessors.([]any)
				if !ok {
					return nil, fmt.Errorf("invalid predecessors")
				}

				for _, predecessor := range predecessorsSlice {
					predecessorMap, ok := predecessor.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("invalid predecessor")
					}

					var p projects.TaskPredecessor
					err = helpers.ParamGroup(predecessorMap,
						helpers.RequiredNumericParam(&p.ID, "task_id"),
						helpers.RequiredParam(&p.Type, "type",
							func(typ *projects.TaskPredecessorType) (bool, error) {
								if typ == nil {
									return false, nil
								}
								switch *typ {
								case projects.TaskPredecessorTypeStart, projects.TaskPredecessorTypeFinish:
									return true, nil
								default:
									return false, fmt.Errorf("invalid type: %s", *typ)
								}
							},
						),
					)
					if err != nil {
						return nil, fmt.Errorf("invalid predecessor: %w", err)
					}

					taskUpdateRequest.Predecessors = append(taskUpdateRequest.Predecessors, p)
				}
			}

			_, err = projects.TaskUpdate(ctx, engine, taskUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update task")
			}

			return mcp.NewToolResultText("Task updated successfully"), nil
		},
	}
}

// TaskDelete deletes a task in Teamwork.com.
func TaskDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskDelete),
			mcp.WithDescription("Delete an existing task in Teamwork.com. "+taskDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the task to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskDeleteRequest projects.TaskDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&taskDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TaskDelete(ctx, engine, taskDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete task")
			}

			return mcp.NewToolResultText("Task deleted successfully"), nil
		},
	}
}

// TaskGet retrieves a task in Teamwork.com.
func TaskGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskGet),
			mcp.WithDescription("Get an existing task in Teamwork.com. "+taskDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the task to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskGetRequest projects.TaskGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&taskGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			task, err := projects.TaskGet(ctx, engine, taskGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get task")
			}

			encoded, err := json.Marshal(task)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
			))), nil
		},
	}
}

// TaskList lists tasks in Teamwork.com.
func TaskList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskList),
			mcp.WithDescription("List tasks in Teamwork.com. "+taskDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter tasks by name."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter tasks by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match tasks that have all the specified tags. "+
					"If false, the search will match tasks that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			taskList, err := projects.TaskList(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}

			encoded, err := json.Marshal(taskList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
			))), nil
		},
	}
}

// TaskListByTasklist lists tasks in Teamwork.com by tasklist.
func TaskListByTasklist(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskListByTasklist),
			mcp.WithDescription("List tasks in Teamwork.com by tasklist. "+taskDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("tasklist_id",
				mcp.Required(),
				mcp.Description("The ID of the tasklist from which to retrieve tasks."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter tasks by name."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter tasks by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match tasks that have all the specified tags. "+
					"If false, the search will match tasks that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&taskListRequest.Path.TasklistID, "tasklist_id"),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			taskList, err := projects.TaskList(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}

			encoded, err := json.Marshal(taskList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
			))), nil
		},
	}
}

// TaskListByProject lists tasks in Teamwork.com by project.
func TaskListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTaskListByProject),
			mcp.WithDescription("List tasks in Teamwork.com by project. "+taskDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project from which to retrieve tasks."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter tasks by name."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter tasks by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match tasks that have all the specified tags. "+
					"If false, the search will match tasks that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&taskListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			taskList, err := projects.TaskList(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}

			encoded, err := json.Marshal(taskList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
			))), nil
		},
	}
}
