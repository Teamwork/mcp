package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTaskCreate   toolsets.Method = "twprojects-create_task"
	MethodTaskUpdate   toolsets.Method = "twprojects-update_task"
	MethodTaskDelete   toolsets.Method = "twprojects-delete_task"
	MethodTaskComplete toolsets.Method = "twprojects-complete_task"
	MethodTaskGet      toolsets.Method = "twprojects-get_task"
	MethodTaskList     toolsets.Method = "twprojects-list_tasks"
	MethodTaskMove     toolsets.Method = "twprojects-move_tasks"
)

var (
	taskGetOutputSchema  *jsonschema.Schema
	taskListOutputSchema *jsonschema.Schema
)

// taskOrdering is the order-by vocabulary of the tasks list endpoint.
var taskOrdering = newOrdering("tasks",
	projects.TaskOrderByID,
	projects.TaskOrderByStartDate,
	projects.TaskOrderByCreatedAt,
	projects.TaskOrderByPriority,
	projects.TaskOrderByProject,
	projects.TaskOrderByFlattenedTasklist,
	projects.TaskOrderByCompany,
	projects.TaskOrderByManual,
	projects.TaskOrderByActive,
	projects.TaskOrderByCompletedAt,
	projects.TaskOrderByDueStartDate,
	projects.TaskOrderByAllDates,
	projects.TaskOrderByTasklistName,
	projects.TaskOrderByTasklistDisplayOrder,
	projects.TaskOrderByTasklistID,
	projects.TaskOrderByDueDate,
	projects.TaskOrderByUpdatedAt,
	projects.TaskOrderByTaskName,
	projects.TaskOrderByCreatedBy,
	projects.TaskOrderByCompletedBy,
	projects.TaskOrderByAssignedTo,
	projects.TaskOrderByTaskStatus,
	projects.TaskOrderByTaskDueDate,
	projects.TaskOrderByCustomField,
	projects.TaskOrderByEstimatedTime,
	projects.TaskOrderByBoardColumn,
	projects.TaskOrderByTaskGroupID,
	projects.TaskOrderByTaskGroupName,
	projects.TaskOrderByTaskGroup,
	projects.TaskOrderByDisplayOrder,
	projects.TaskOrderByProjectManual,
	projects.TaskOrderByStageDisplayOrder,
	projects.TaskOrderByStage,
	projects.TaskOrderByParentTask,
)

// taskDateFilters is the vocabulary of the endpoint's taskFilter parameter: the
// scheduling states the task UI puts in front of users, which is why callers
// ask for them by those words. Publishing them is what keeps a model from
// reading a whole project and comparing dates itself.
//
// Every value is a statement about dates, and only about dates. The endpoint's
// `all` and `completed` are not: they switch completed work on, which
// show_completed and only_completed already own, and each one replaced the date
// filter rather than combining with it. TaskDateFilterNewTaskDefaults is absent
// too — it selects the per-project new-task template rows, not work anybody
// asked about — and so is TaskDateFilterCreated, which restricts no date on its
// own and only means something alongside a created-date filter this tool does
// not expose.
var taskDateFilters = newVocabulary(
	projects.TaskDateFilterAnytime,
	projects.TaskDateFilterOverdue,
	projects.TaskDateFilterToday,
	projects.TaskDateFilterTomorrow,
	projects.TaskDateFilterYesterday,
	projects.TaskDateFilterThisWeek,
	projects.TaskDateFilterUpcoming,
	projects.TaskDateFilterStarted,
	projects.TaskDateFilterWithin7,
	projects.TaskDateFilterWithin14,
	projects.TaskDateFilterWithin30,
	projects.TaskDateFilterWithin365,
	projects.TaskDateFilterNoDate,
	projects.TaskDateFilterNoDueDate,
	projects.TaskDateFilterNoStartDate,
	projects.TaskDateFilterHasDate,
)

func init() {
	var err error

	// generate the output schemas only once
	taskGetOutputSchema, err = jsonschema.For[projects.TaskGetResponse](helpers.WithDateTypeSchema(&jsonschema.ForOptions{}))
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TaskGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(taskGetOutputSchema)
	taskListOutputSchema, err = jsonschema.For[projects.TaskListResponse](helpers.WithDateTypeSchema(&jsonschema.ForOptions{}))
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
			Description: "Create task in a tasklist.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Task",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Description: "Tasklist ID. Use twprojects-list_tasklists to find one.",
					},
					"workflow_id": {
						Description: "The ID of the workflow to place the new task in a stage of. Required " +
							"together with stage_id, and must be a workflow attached to the task's own project: " +
							"another one is ignored and the task lands in the backlog, with nothing in the " +
							"response saying so. Use twprojects-list_workflows to find one.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"stage_id": {
						Description: "The ID of the workflow stage to place the new task in. Required together " +
							"with workflow_id. Omit both to leave the task in the workflow's backlog. Use " +
							"twprojects-list_workflow_stages to find one.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the task. Support for plain text and Markdown formatting.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"priority": {
						Description: "The priority of the task.",
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
					"start_date": helpers.DateFilterSchema("The start date of the task."),
					"due_date": helpers.DateFilterSchema(
						"The due date of the task. If omitted, falls back to the milestone due date when one is set.",
					),
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
					"assignees":           helpers.UserGroupsSchema("Assignees for the task.", false),
					"tag_ids":             helpers.TagIDsAssociateSchema("task"),
					"attachment_refs":     attachmentRefsSchema("task"),
					"attachment_file_ids": attachmentFileIDsSchema("task"),
					"predecessors": {
						Description: "Task dependencies that must be completed before this task can start.",
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
											Description: "'start' means this task can complete when the predecessor starts; " +
												"'complete' means this task can complete when the predecessor completes.",
											Enum: []any{"start", "complete"},
										},
									},
								},
							},
							{Type: "null"},
						},
					},
					"change_followers":   helpers.UserGroupsSchema("Followers of any task changes.", false),
					"comment_followers":  helpers.UserGroupsSchema("Followers of any task comments.", false),
					"complete_followers": helpers.UserGroupsSchema("Followers of any task completions.", false),
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
				helpers.OptionalNumericPointerParam(&taskCreateRequest.Workflows.WorkflowID, "workflow_id"),
				helpers.OptionalNumericPointerParam(&taskCreateRequest.Workflows.StageID, "stage_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// The endpoint drops either ID sent alone and still answers 201, so
			// say it here rather than let the placement vanish.
			if (taskCreateRequest.Workflows.WorkflowID == nil) != (taskCreateRequest.Workflows.StageID == nil) {
				return helpers.NewToolResultTextError(
					"workflow_id and stage_id must be provided together"), nil
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

			// Only set attachments when the caller named one: the field is a
			// sibling of the task in the request body, so an empty one would be a
			// payload change for every caller that attaches nothing.
			if attachments, toolResult := parseTaskAttachments(arguments); toolResult != nil {
				return toolResult, nil
			} else if attachments != nil {
				taskCreateRequest.Attachments = *attachments
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
			Description: "Update task.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Task",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Description: "The ID of the tasklist. Moving the task carries its subtasks along " +
							"and detaches it from any parent staying behind. Use twprojects-move_tasks to " +
							"move several tasks at once.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the task. Support for plain text and Markdown formatting.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"priority": {
						Description: "The priority of the task.",
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
					"start_date": helpers.DateFilterSchema("The start date of the task."),
					"due_date": helpers.DateFilterSchema(
						"The due date of the task. If omitted, falls back to the milestone due date when one is set.",
					),
					"estimated_minutes": {
						Description: "The estimated time to complete the task in minutes.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"parent_task_id": {
						Description: "The ID of the parent task, making this task a subtask. A subtask must " +
							"live in the same tasklist as its parent, so moving one with tasklist_id fails " +
							"until the parent has moved or the link is cleared. To detach the task from its " +
							"parent, use clear_parent_task instead.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"clear_parent_task": {
						Description: "If true, detaches the task from its parent, promoting it to a top-level " +
							"task. Cannot be combined with parent_task_id.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"assignees": helpers.UserGroupsSchema("Assignees for the task. To remove all "+
						"assignees, use clear_assignees instead.", false),
					"clear_assignees": {
						Description: "If true, removes all assignees from the task, leaving it unassigned. " +
							"Cannot be combined with a non-empty assignees value.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"tag_ids":             helpers.TagIDsAssociateSchema("task"),
					"attachment_refs":     attachmentRefsSchema("task"),
					"attachment_file_ids": attachmentFileIDsSchema("task"),
					"predecessors": {
						Description: "Task dependencies that must be completed before this task can start.",
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
											Description: "'start' means this task can complete when the predecessor starts; " +
												"'complete' means this task can complete when the predecessor completes.",
											Enum: []any{"start", "complete"},
										},
									},
								},
							},
							{Type: "null"},
						},
					},
					"change_followers":   helpers.UserGroupsSchema("Followers of any task changes.", false),
					"comment_followers":  helpers.UserGroupsSchema("Followers of any task comments.", false),
					"complete_followers": helpers.UserGroupsSchema("Followers of any task completions.", false),
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

			var clearAssignees bool
			if err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&clearAssignees, "clear_assignees"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid clear_assignees: %s", err.Error()), nil
			}

			var clearParentTask bool
			if err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&clearParentTask, "clear_parent_task"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid clear_parent_task: %s", err.Error()), nil
			}
			if clearParentTask {
				if taskUpdateRequest.ParentTaskID != nil {
					return helpers.NewToolResultTextError(
						"clear_parent_task cannot be combined with parent_task_id",
					), nil
				}
				// A null parent_task_id cannot express this: null means "not
				// provided" for every optional parameter, because OpenAI strict mode
				// requires clients to send every property and fill the unset ones
				// with null. Zero is the sentinel the v3 API accepts to detach, and
				// it survives the SDK's omitempty because only a nil pointer is
				// omitted.
				taskUpdateRequest.ParentTaskID = new(int64(0))
			}

			if assignees, toolResult := parseUserGroups(
				arguments,
				"assignees",
				"assignees",
			); toolResult != nil {
				return toolResult, nil
			} else if assignees != nil {
				assigneesEmpty := len(assignees.UserIDs) == 0 &&
					len(assignees.CompanyIDs) == 0 &&
					len(assignees.TeamIDs) == 0 &&
					len(assignees.JobRoleIDs) == 0
				if clearAssignees && !assigneesEmpty {
					return helpers.NewToolResultTextError(
						"clear_assignees cannot be combined with a non-empty assignees value",
					), nil
				}
				taskUpdateRequest.Assignees = assignees
			}

			// Only set attachments when the caller named one. Attaching is additive
			// server side, so this never disturbs the files the task already has.
			if attachments, toolResult := parseTaskAttachments(arguments); toolResult != nil {
				return toolResult, nil
			} else if attachments != nil {
				taskUpdateRequest.Attachments = *attachments
			}

			if clearAssignees {
				// Empty arrays unassign every assignee dimension. Job roles are
				// included because the SDK now sends the "Jobroles-Enabled: true"
				// header on every request, so the API honours jobRoleIds here.
				taskUpdateRequest.Assignees = &projects.UserGroups{
					UserIDs:    []int64{},
					CompanyIDs: []int64{},
					TeamIDs:    []int64{},
					JobRoleIDs: []int64{},
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

const (
	// taskMoveMaxDepth bounds the walk up a task's ancestor chain. Real nesting is
	// a handful of levels; the cap only stops a corrupt chain looping forever.
	taskMoveMaxDepth = 20

	// taskMoveMaxTasks bounds the fan-out. Each task costs a read and a write, run
	// in sequence, so a longer list would outlive the request.
	taskMoveMaxTasks = 50
)

// taskMoveCarriedTasks reports which of the requested tasks another one will
// carry, so they are not moved a second time.
//
// The order the caller listed them in decides whether that second move is
// harmless. An ancestor moved first carries the descendant, and updating the
// descendant afterwards is a no-op because it is already in the destination. A
// descendant moved first, though, leaves its parent behind and so gets detached,
// flattening the subtree the tool promises to preserve.
func taskMoveCarriedTasks(
	ctx context.Context,
	engine *twapi.Engine,
	roots []int64,
	requested map[int64]bool,
	tasklistID int64,
) (map[int64]bool, error) {
	carried := make(map[int64]bool)
	if len(roots) == 1 {
		// Nothing can be an ancestor of anything else, so skip the reads.
		return carried, nil
	}

	tasks := make(map[int64]projects.Task, len(roots))
	resolve := func(id int64) (projects.Task, error) {
		if task, ok := tasks[id]; ok {
			return task, nil
		}
		taskGetResponse, err := projects.TaskGet(ctx, engine, projects.NewTaskGetRequest(id))
		if err != nil {
			return projects.Task{}, err
		}
		tasks[id] = taskGetResponse.Task
		return taskGetResponse.Task, nil
	}

	for _, root := range roots {
		id := root
		for range taskMoveMaxDepth {
			task, err := resolve(id)
			if err != nil {
				return nil, err
			}
			if task.ParentTask == nil {
				break
			}
			parent, err := resolve(task.ParentTask.ID)
			if err != nil {
				return nil, err
			}
			// An ancestor already in the destination stays put, so it carries
			// nothing and the descendant still has to move itself.
			if requested[parent.ID] && parent.Tasklist.ID != tasklistID {
				carried[root] = true
				break
			}
			id = parent.ID
		}
	}
	return carried, nil
}

func sortedTaskIDs(ids map[int64]bool) []int64 {
	sorted := make([]int64, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	slices.Sort(sorted)
	return sorted
}

// TaskMove moves tasks, along with every subtask beneath them, to another
// tasklist in Teamwork.com.
//
// One update per task is all this takes: the API moves the whole subtree and
// drops an inherited parent the move would invalidate, so no detaching happens
// here. What the tool adds is the batch, plus a skip for any task another one
// will carry — moving a descendant before its ancestor would detach it.
func TaskMove(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTaskMove),
			Description: "Move tasks and all their subtasks to another tasklist, preserving the " +
				"parent/child structure. Subtasks move with their parent automatically, so only the " +
				"topmost task of each subtree needs to be listed. A task whose parent is not part of " +
				"the move is detached from it, becoming a top-level task in the destination.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Move Tasks",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_ids": {
						Type:        "array",
						Items:       &jsonschema.Schema{Type: "integer"},
						Description: "The IDs of the tasks to move. Subtasks are moved automatically.",
					},
					"tasklist_id": {
						Type:        "integer",
						Description: "The ID of the destination tasklist.",
					},
				},
				Required: []string{"task_ids", "tasklist_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var taskIDs []int64
			var tasklistID int64
			if err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericListParam(&taskIDs, "task_ids"),
				helpers.RequiredNumericParam(&tasklistID, "tasklist_id"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if len(taskIDs) == 0 {
				return helpers.NewToolResultTextError("task_ids must contain at least one task ID"), nil
			}
			if len(taskIDs) > taskMoveMaxTasks {
				return helpers.NewToolResultTextError(
					"task_ids accepts at most %d tasks, got %d; subtasks move automatically, so only "+
						"the topmost task of each subtree needs to be listed",
					taskMoveMaxTasks, len(taskIDs),
				), nil
			}

			seen := make(map[int64]bool, len(taskIDs))
			roots := make([]int64, 0, len(taskIDs))
			for _, id := range taskIDs {
				if !seen[id] {
					seen[id] = true
					roots = append(roots, id)
				}
			}

			carried, err := taskMoveCarriedTasks(ctx, engine, roots, seen, tasklistID)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to load the parents of the requested tasks")
			}

			var moved []int64
			var failures []string
			for _, id := range roots {
				if carried[id] {
					continue
				}

				taskUpdateRequest := projects.NewTaskUpdateRequest(id)
				// One move can touch a whole subtree; notifying on each would bury
				// every follower under a reorganisation they did not ask about.
				taskUpdateRequest.Options.Notify = false
				taskUpdateRequest.TasklistID = &tasklistID

				if _, err := projects.TaskUpdate(ctx, engine, taskUpdateRequest); err != nil {
					failures = append(failures, fmt.Sprintf("task %d: %s", id, err.Error()))
					continue
				}
				moved = append(moved, id)
			}

			var report strings.Builder
			fmt.Fprintf(&report, "Moved %d of %d tasks to tasklist %d, each with its subtasks.",
				len(moved), len(roots), tasklistID)
			if len(moved) > 0 {
				fmt.Fprintf(&report, "\nMoved: %s.", joinTaskIDs(moved))
			}
			if len(carried) > 0 {
				fmt.Fprintf(&report, "\nCarried by another requested task: %s.", joinTaskIDs(sortedTaskIDs(carried)))
			}
			for _, failure := range failures {
				fmt.Fprintf(&report, "\nFailed: %s.", failure)
			}
			if len(failures) > 0 {
				return helpers.NewToolResultTextError("%s", report.String()), nil
			}
			return helpers.NewToolResultText("%s", report.String()), nil
		},
	}
}

func joinTaskIDs(ids []int64) string {
	formatted := make([]string, 0, len(ids))
	for _, id := range ids {
		formatted = append(formatted, strconv.FormatInt(id, 10))
	}
	return strings.Join(formatted, ", ")
}

// TaskDelete deletes a task in Teamwork.com.
func TaskDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskDelete),
			Description: "Delete task.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Task",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
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
			Description: "Mark task complete.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Complete Task",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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

// taskFieldsNeedRelatedTasks reports whether a sparse-field selection names an
// attribute the API leaves empty unless the request also asks for related
// tasks. Both are computed rather than stored, so the selection alone is not
// enough to populate them.
func taskFieldsNeedRelatedTasks(fields []projects.TaskField) bool {
	return slices.Contains(fields, projects.TaskFieldPredecessors) ||
		slices.Contains(fields, projects.TaskFieldSubTaskIDs)
}

// TaskGet retrieves a task in Teamwork.com.
func TaskGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTaskGet),
			Description: "Get task.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Task",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the task to get.",
					},
					"fields": helpers.FieldsSchema[projects.Task]("task"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(taskGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskGetRequest projects.TaskGetRequest
			taskGetRequest.Filters.IncludeRelatedTasks = true

			// The related-task filter reports *active* subtasks, dependencies and
			// predecessors only, so a task whose subtasks are all done answers with an
			// empty subTaskIds — indistinguishable from a task that never had any. The
			// flag below widens the whole related-task response, not the predecessors
			// its name suggests, and it cannot drop the row: this endpoint addresses
			// the task by ID.
			taskGetRequest.Filters.IncludeCompletedPredecessors = true

			// Always include custom fields and values in task get response for richer
			// context, as they are commonly used in Teamwork projects and provide
			// valuable information about the task.
			taskGetRequest.Filters.Include = []projects.TaskRequestSideload{
				projects.TaskRequestSideloadCustomFields,
				projects.TaskRequestSideloadCustomFieldValues,
			}

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.Task](&taskGetRequest.Fields.Task, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(taskGetRequest.Fields.Task) > 0 {
				// Drop the sideloads: they are not what the selection named, and
				// they would return the bulk it exists to avoid. IncludeRelatedTasks
				// is not one of them — it adds no entity to the response, it is what
				// makes `predecessors` and `subTaskIds` non-empty at all — so put it
				// back, with the two completed-work flags it gates, when the selection
				// names either attribute.
				relatedTasks := taskFieldsNeedRelatedTasks(taskGetRequest.Fields.Task)
				taskGetRequest.Filters = projects.TaskRequestFilters{
					IncludeRelatedTasks:          relatedTasks,
					IncludeCompletedPredecessors: relatedTasks,
				}
				return helpers.NewRawToolResult(ctx, engine, taskGetRequest, "failed to get task",
					helpers.WebLinkerWithIDPathBuilder("/app/tasks"),
				)
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
			Name: string(MethodTaskList),
			Description: "List tasks with structured filters (tasklist_id, project_id, or site-wide). " +
				"For keyword search use search. Completed tasks and tasks in completed tasklists are " +
				"excluded unless show_completed is true, so an empty result may mean the matching work " +
				"is already done rather than missing. Ask for \"late\", \"overdue\", \"due today\", " +
				"\"started\" or \"upcoming\" work through date_filter, and leave people out through " +
				"exclude_assignee_user_ids, rather than reading rows and filtering them yourself.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Tasks",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tasklist_id": {
						Description: "The ID of the tasklist from which to retrieve tasks. Takes precedence over project_id.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project from which to retrieve tasks. Omit to list tasks across all projects.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"search_term": helpers.SearchTermSchema("tasks", "name"),
					"assignee_user_ids": {
						Description: "Filter tasks by assignee.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"exclude_assignee_user_ids": {
						Description: "Leave out tasks assigned to any of these users. A task is dropped when any " +
							"one of the listed users is assigned to it, even when it also carries assignees you " +
							"did not exclude. A user reached only through a team, company or job-role assignment " +
							"on the task is not matched. Combines with assignee_user_ids and every other filter.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"tag_ids":        helpers.TagIDsFilterSchema("tasks"),
					"match_all_tags": helpers.MatchAllTagsSchema(),
					"created_after": helpers.DateTimeFilterSchema(
						"Only include tasks created at or after this moment; the boundary itself matches."),
					"created_before": helpers.DateTimeFilterSchema(
						"Only include tasks created at or before this moment; the boundary itself matches."),
					"created_by_user_ids": {
						Description: "Filter tasks by creator.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"updated_after": helpers.DateTimeFilterSchema(
						"Only include tasks updated strictly after this moment; the boundary itself does not match."),
					"updated_before": helpers.DateTimeFilterSchema(
						"Only include tasks updated strictly before this moment; the boundary itself does not match."),
					"completed_after": helpers.DateTimeFilterSchema(
						"Only include tasks completed at or after this moment; the boundary itself matches. " +
							"Setting it narrows the result to completed tasks."),
					"completed_before": helpers.DateTimeFilterSchema(
						"Only include tasks completed at or before this moment; the boundary itself matches. " +
							"Setting it narrows the result to completed tasks."),
					"date_filter": taskDateFilters.schema(
						"Where the task's dates fall relative to today, in your own timezone. This is the " +
							"filter for \"late\", \"overdue\", \"due today\", \"started\" and \"upcoming\"; " +
							"omit it for no date restriction. overdue: due before today and not completed — this " +
							"is \"late\", and it never returns completed tasks whatever show_completed says. " +
							"today: due today, without adding the overdue ones. thisweek: the calendar week " +
							"containing today, the days of it already past included. upcoming: due today or " +
							"later. started: the start date has arrived and the due date has not passed — start " +
							"date on or before today, and either no due date at all or one falling today or " +
							"later. within7, within14, within30, within365: due between today and that many days " +
							"from today, both days included. nodate: no start date, no due date and no " +
							"milestone. anytime: no date restriction, what the endpoint applies when this is " +
							"omitted. Completed tasks stay hidden unless show_completed is true, and overdue " +
							"never returns one even then; for completed work alone use only_completed, which " +
							"combines with any value here except overdue. A task with no due date of its own is " +
							"matched on its milestone's."),
					"start_after": {
						Description: "Only include tasks whose own start date falls on or after this date; the " +
							"day itself matches. A task with no start date never matches — there is no " +
							"milestone fallback. There is no upper bound on the start date, so for work that " +
							"has already begun use date_filter=started instead.",
						Examples: []any{"2023-01-01"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"due_after": {
						Description: "Only include tasks due after this date, excluding the day itself — " +
							"unless due_before is set too, which makes both bounds inclusive. A task with " +
							"no due date is matched on its milestone's.",
						Examples: []any{"2023-01-01"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"due_before": {
						Description: "Only include tasks due before this date, excluding the day itself — " +
							"unless due_after is set too, which makes both bounds inclusive. A task with " +
							"no due date is matched on its milestone's.",
						Examples: []any{"2023-12-31"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"show_completed": {
						Description: "If true, include completed tasks and tasks belonging to completed tasklists; " +
							"both excluded by default.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
						Default: []byte(`false`),
					},
					"only_completed": {
						Description: "If true, return only completed tasks. It combines with every other " +
							"filter, date_filter included — except date_filter=overdue, which never matches a " +
							"completed task and so returns nothing. Tasks in completed tasklists still need " +
							"show_completed.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"only_unassigned": {
						Description: "If true, only return tasks that have no assignee.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"only_unplanned": {
						Description: "If true, only return tasks that are unplanned, meaning they are missing an " +
							"assignee, a due date, or estimated time.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"order_by":                 taskOrdering.orderBySchema(),
					"order_mode":               orderModeSchema(),
					"order_by_custom_field_id": orderByFieldIDSchema("tasks", "customfield"),
					"page":                     helpers.PageSchema(),
					"page_size":                helpers.PageSizeSchema(),
					"verbose":                  helpers.VerboseSchema(),
					"count_only":               helpers.CountOnlySchema("tasks"),
					"fields":                   helpers.FieldsSchema[projects.Task]("task"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(
				helpers.WithOptionalFields(withSuggestionsSchema(taskListOutputSchema)),
			),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			var showCompleted *bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&taskListRequest.Path.TasklistID, "tasklist_id"),
				helpers.OptionalNumericParam(&taskListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.AssigneeUserIDs, "assignee_user_ids"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.ExcludeAssigneeUserIDs,
					"exclude_assignee_user_ids"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
				taskOrdering.param(&taskListRequest.Filters.OrderBy, &taskListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&taskListRequest.Filters.OrderByCustomFieldID, "order_by_custom_field_id"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&taskListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CreatedAfter, "created_after"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CreatedBefore, "created_before",
					helpers.EndOfDay()),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.CreatedByUserIDs, "created_by_user_ids"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.UpdatedBefore, "updated_before",
					helpers.EndOfDay()),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CompletedAfter, "completed_after"),
				helpers.OptionalTimePointerParam(&taskListRequest.Filters.CompletedBefore, "completed_before",
					helpers.EndOfDay()),
				taskDateFilters.param(&taskListRequest.Filters.DateFilter, "date_filter"),
				helpers.OptionalDatePointerParam(&taskListRequest.Filters.StartAfter, "start_after"),
				helpers.OptionalDatePointerParam(&taskListRequest.Filters.DueAfter, "due_after"),
				helpers.OptionalDatePointerParam(&taskListRequest.Filters.DueBefore, "due_before"),
				helpers.OptionalPointerParam(&showCompleted, "show_completed"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.OnlyCompleted, "only_completed"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.OnlyUnassigned, "only_unassigned"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.OnlyUnplanned, "only_unplanned"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.Task](&taskListRequest.Filters.Fields.Tasks, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if showCompleted != nil {
				// A single flag drives all three SDK filters: completed tasks are hidden
				// by default, so are tasks living inside a completed tasklist, and so are
				// the completed entries of the related-task lists a row carries. Callers
				// asking to see completed work expect all of them, and a caller who asked
				// to hide it would not expect completed IDs in `subTaskIds`.
				taskListRequest.Filters.IncludeCompleted = showCompleted
				taskListRequest.Filters.IncludeCompletedTasklists = showCompleted
				taskListRequest.Filters.IncludeCompletedPredecessors = *showCompleted
			}
			if countOnly {
				// Ahead of the row wiring below: the count returns no rows.
				return helpers.NewCountToolResult(ctx, engine, taskListRequest, "failed to count tasks")
			}
			switch {
			case len(taskListRequest.Filters.Fields.Tasks) > 0:
				// An explicit field selection overrides both defaults below: the
				// caller has already said what it wants, and sideloading custom
				// fields would smuggle back the bulk the selection exists to avoid.
				//
				// `predecessors` and `subTaskIds` are the exception: the API only
				// populates them when the request also asks for related tasks, so
				// selecting either without the filter returns an empty array on every
				// row — indistinguishable from a task nothing blocks and a task with
				// no subtasks, which is how a dependency question ends up answered
				// "nothing is blocking". The filter is not a sideload; it adds no
				// other entity to the response.
				taskListRequest.Filters.IncludeRelatedTasks = taskFieldsNeedRelatedTasks(
					taskListRequest.Filters.Fields.Tasks)

			case verbose:
				// Include custom fields and values in task list response for richer
				// context, as they are commonly used in Teamwork projects and provide
				// valuable information about the task.
				taskListRequest.Filters.Include = []projects.TaskRequestSideload{
					projects.TaskRequestSideloadCustomFields,
					projects.TaskRequestSideloadCustomFieldValues,
				}
				// Ensure predecessors and subtask IDs are included in the response
				taskListRequest.Filters.IncludeRelatedTasks = true

			default:
				taskListRequest.Filters.Fields.Tasks = []projects.TaskField{
					projects.TaskFieldID,
					projects.TaskFieldName,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, taskListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasks")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list tasks"), "failed to list tasks")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/tasks"))
			linked, err = withNearMissSuggestions(ctx, engine, linked, "tasks", taskListRequest.Filters.SearchTerm)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to generate near-miss suggestions")
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(linked),
					},
				},
			}
			var structured any
			if err := json.Unmarshal(linked, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
