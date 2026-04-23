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
	MethodTaskComplete       toolsets.Method = "twprojects-complete_task"
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
	var err error

	// generate the output schemas only once
	taskGetOutputSchema, err = jsonschema.For[projects.TaskGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TaskGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(taskGetOutputSchema)
	taskListOutputSchema, err = jsonschema.For[projects.TaskListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TaskListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(taskListOutputSchema)
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
						Type: "integer",
						Description: "The ID of the tasklist. If you only have the project's name, use the " +
							string(MethodProjectList) + " method with the search_term parameter to find the project ID, and " +
							"then the " + string(MethodTasklistList) + " method with the project_id to choose the tasklist ID. If " +
							"you know the tasklist's name, you may also use the search_term parameter with the " +
							string(MethodTasklistList) + " method to find the tasklist ID.",
					},
					"description": {
						Description: "The description of the task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"priority": {
						Description: "The priority of the task. Possible values are: low, medium, high.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"low", "medium", "high"}},
							{Type: "null"},
						},
					},
					"progress": {
						Description: "The progress of the task, as a percentage (0-100). Only whole numbers are allowed.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer", Minimum: new(float64(0)), Maximum: new(float64(100))},
							{Type: "null"},
						},
					},
					"start_date": {
						Description: "The start date of the task in ISO 8601 format (YYYY-MM-DD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"due_date": {
						Description: "The due date of the task in ISO 8601 format (YYYY-MM-DD). When this is not provided, it " +
							"will fallback to the milestone due date if a milestone is set.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"estimated_minutes": {
						Description: "The estimated time to complete the task in minutes.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"parent_task_id": {
						Description: "The ID of the parent task if creating a subtask.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"assignees": {
						Description: "An object containing assignees for the task.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs assigned to the task.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs assigned to the task.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs assigned to the task.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to assign to the task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"predecessors": {
						Description: "List of task dependencies that must be completed before this task can start, defining its " +
							"position in the project workflow and ensuring proper sequencing of work.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
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
							{Type: "null"},
						},
					},
					"change_followers": {
						Description: "An object containing the followers of any task changes.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs following the task changes.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs following the task changes.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs following the task changes.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"comment_followers": {
						Description: "An object containing the followers of any task comments.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs following the task comments.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs following the task comments.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs following the task comments.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"complete_followers": {
						Description: "An object containing the followers of any task completions.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs following the task completions.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs following the task completions.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs following the task completions.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name", "tasklist_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskCreateRequest projects.TaskCreateRequest
			taskCreateRequest.Options.Notify = true
			taskCreateRequest.Options.CheckInvalidUsers = true

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if assignees, toolResult := parseUserGroups(
				arguments,
				"assignees",
				"assignees",
			); toolResult != nil {
				return toolResult, nil
			} else if assignees != nil {
				taskCreateRequest.Assignees = assignees
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
						return helpers.NewToolResultTextError("invalid predecessor: %s", err), nil
					}

					taskCreateRequest.Predecessors = append(taskCreateRequest.Predecessors, p)
				}
			}

			if followers, toolResult := parseUserGroups(
				arguments,
				"change_followers",
				"change followers",
			); toolResult != nil {
				return toolResult, nil
			} else if followers != nil {
				taskCreateRequest.ChangeFollowers = *followers
			}
			if followers, toolResult := parseUserGroups(
				arguments,
				"comment_followers",
				"comment followers",
			); toolResult != nil {
				return toolResult, nil
			} else if followers != nil {
				taskCreateRequest.CommentFollowers = *followers
			}
			if followers, toolResult := parseUserGroups(
				arguments,
				"complete_followers",
				"complete followers",
			); toolResult != nil {
				return toolResult, nil
			} else if followers != nil {
				taskCreateRequest.CompleteFollowers = *followers
			}

			taskResponse, err := projects.TaskCreate(ctx, engine, taskCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create task")
			}
			return helpers.NewToolResultText("Task created successfully with ID %d", taskResponse.Task.ID), nil
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
					"name": {
						Description: "The name/title of the task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tasklist_id": {
						Description: "The ID of the tasklist.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"priority": {
						Description: "The priority of the task. Possible values are: low, medium, high.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"low", "medium", "high"}},
							{Type: "null"},
						},
					},
					"progress": {
						Description: "The progress of the task, as a percentage (0-100). Only whole numbers are allowed.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer", Minimum: new(float64(0)), Maximum: new(float64(100))},
							{Type: "null"},
						},
					},
					"start_date": {
						Description: "The start date of the task in ISO 8601 format (YYYY-MM-DD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"due_date": {
						Description: "The due date of the task in ISO 8601 format (YYYY-MM-DD). When this is not provided, it " +
							"will fallback to the milestone due date if a milestone is set.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"estimated_minutes": {
						Description: "The estimated time to complete the task in minutes.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"parent_task_id": {
						Description: "The ID of the parent task if creating a subtask.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"assignees": {
						Description: "An object containing assignees for the task.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs assigned to the task.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs assigned to the task.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs assigned to the task.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to assign to the task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"predecessors": {
						Description: "List of task dependencies that must be completed before this task can start, defining its " +
							"position in the project workflow and ensuring proper sequencing of work.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
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
							{Type: "null"},
						},
					},
					"change_followers": {
						Description: "An object containing the followers of any task changes.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs following the task changes.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs following the task changes.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs following the task changes.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"comment_followers": {
						Description: "An object containing the followers of any task comments.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs following the task comments.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs following the task comments.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs following the task comments.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"complete_followers": {
						Description: "An object containing the followers of any task completions.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs following the task completions.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs following the task completions.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs following the task completions.",
										Items:       &jsonschema.Schema{Type: "integer"},
										MinItems:    new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskUpdateRequest projects.TaskUpdateRequest
			taskUpdateRequest.Options.Notify = true
			taskUpdateRequest.Options.CheckInvalidUsers = true

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if assignees, toolResult := parseUserGroups(
				arguments,
				"assignees",
				"assignees",
			); toolResult != nil {
				return toolResult, nil
			} else if assignees != nil {
				taskUpdateRequest.Assignees = assignees
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
						return helpers.NewToolResultTextError("invalid predecessor: %s", err), nil
					}

					taskUpdateRequest.Predecessors = append(taskUpdateRequest.Predecessors, p)
				}
			}

			if followers, toolResult := parseUserGroups(
				arguments,
				"change_followers",
				"change followers",
			); toolResult != nil {
				return toolResult, nil
			} else if followers != nil {
				taskUpdateRequest.ChangeFollowers = followers
			}
			if followers, toolResult := parseUserGroups(
				arguments,
				"comment_followers",
				"comment followers",
			); toolResult != nil {
				return toolResult, nil
			} else if followers != nil {
				taskUpdateRequest.CommentFollowers = followers
			}
			if followers, toolResult := parseUserGroups(
				arguments,
				"complete_followers",
				"complete followers",
			); toolResult != nil {
				return toolResult, nil
			} else if followers != nil {
				taskUpdateRequest.CompleteFollowers = followers
			}

			_, err = projects.TaskUpdate(ctx, engine, taskUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update task")
			}
			return helpers.NewToolResultText("Task updated successfully"), nil
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TaskDelete(ctx, engine, taskDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete task")
			}
			return helpers.NewToolResultText("Task deleted successfully"), nil
		},
	}
}

// TaskComplete marks a task as complete in Teamwork.com.
func TaskComplete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskComplete),
			Description: "Mark an existing task as complete in Teamwork.com. " + taskDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Complete Task",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the task to mark as complete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskCompleteRequest projects.TaskCompleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskCompleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TaskComplete(ctx, engine, taskCompleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to complete task")
			}
			return helpers.NewToolResultText("Task completed successfully"), nil
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
				StructuredContent: helpers.StructuredWebLinker(ctx, task,
					helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
				),
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
						Description: "A search term to filter tasks by name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"assignee_user_ids": {
						Description: "A list of user IDs to filter tasks by assigned users",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"created_after": {
						Description: "Filter tasks created after this date and time in RFC 3339 format.",
						Examples:    []any{"2023-01-01T00:00:00Z"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"created_before": {
						Description: "Filter tasks created before this date and time in RFC 3339 format.",
						Examples:    []any{"2023-12-31T23:59:59Z"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"updated_after": {
						Description: "Filter tasks updated after this date and time in RFC 3339 format.",
						Examples:    []any{"2023-01-01T00:00:00Z"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"updated_before": {
						Description: "Filter tasks updated before this date and time in RFC 3339 format.",
						Examples:    []any{"2023-12-31T23:59:59Z"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"completed_after": {
						Description: "Filter tasks completed after this date and time in RFC 3339 format.",
						Examples:    []any{"2023-01-01T00:00:00Z"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"completed_before": {
						Description: "Filter tasks completed before this date and time in RFC 3339 format.",
						Examples:    []any{"2023-12-31T23:59:59Z"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to filter tasks by tags",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"match_all_tags": {
						Description: "If true, the search will match tasks that have all the specified tags. If false, the " +
							"search will match tasks that have any of the specified tags. Defaults to false.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"page": {
						Description: "Page number for pagination of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page_size": {
						Description: "Number of results per page for pagination.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{},
			},
			OutputSchema: taskListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.AssigneeUserIDs, "assignee_user_ids"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CreatedAfter, "created_after"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CreatedBefore, "created_before"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.UpdatedBefore, "updated_before"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CompletedAfter, "completed_after"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CompletedBefore, "completed_before"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
				StructuredContent: helpers.StructuredWebLinker(ctx, taskList,
					helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
				),
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
						Description: "A search term to filter tasks by name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to filter tasks by tags",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"assignee_user_ids": {
						Description: "A list of user IDs to filter tasks by assigned users",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"match_all_tags": {
						Description: "If true, the search will match tasks that have all the specified tags. If false, the " +
							"search will match tasks that have any of the specified tags. Defaults to false.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"page": {
						Description: "Page number for pagination of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page_size": {
						Description: "Number of results per page for pagination.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
				StructuredContent: helpers.StructuredWebLinker(ctx, taskList,
					helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
				),
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
						Description: "A search term to filter tasks by name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tag_ids": {
						Description: "A list of tag IDs to filter tasks by tags",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"assignee_user_ids": {
						Description: "A list of user IDs to filter tasks by assigned users",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"match_all_tags": {
						Description: "If true, the search will match tasks that have all the specified tags. If false, the " +
							"search will match tasks that have any of the specified tags. Defaults to false.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"page": {
						Description: "Page number for pagination of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page_size": {
						Description: "Number of results per page for pagination.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
				StructuredContent: helpers.StructuredWebLinker(ctx, taskList,
					helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
				),
			}, nil
		},
	}
}
