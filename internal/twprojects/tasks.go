package twprojects

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
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

// taskMoveMaxDepth bounds the walk up a task's ancestor chain. Real subtask
// nesting is a handful of levels; the cap only stops a corrupt chain from
// looping forever.
const taskMoveMaxDepth = 20

var (
	taskGetOutputSchema  *jsonschema.Schema
	taskListOutputSchema *jsonschema.Schema
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
					"assignees": helpers.UserGroupsSchema("Assignees for the task.", false),
					"tag_ids":   helpers.TagIDsAssociateSchema("task"),
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
						Description: "The ID of the tasklist. This moves the task alone and leaves its " +
							"subtasks behind; use twprojects-move_tasks to move a task together with its " +
							"subtasks.",
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
					"tag_ids": helpers.TagIDsAssociateSchema("task"),
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

// taskMoveV1Request moves a task to another tasklist through the v1 API.
//
// v3's PATCH /projects/api/v3/tasks/{id}.json moves only the task it names,
// leaving the subtasks behind in a state that violates the API's own rule that
// a subtask shares its parent's tasklist. The v1 endpoint cascades instead: one
// call moves the task and its whole subtree, and answers with every task ID it
// touched. Verified against a live installation, where moving one parent
// returned six affected IDs.
type taskMoveV1Request struct {
	TaskID     int64
	TasklistID int64

	// DetachFromParent clears the task's parent link in the same call. Needed
	// when the parent stays behind, since the API rejects a subtask whose
	// tasklist differs from its parent's.
	DetachFromParent bool
}

// HTTPRequest builds the PUT /tasks/{id}.json request. v1 lives at the
// installation root, with no /projects/api prefix.
func (t taskMoveV1Request) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	uri := server + "/tasks/" + strconv.FormatInt(t.TaskID, 10) + ".json"

	type todoItem struct {
		ID         int64 `json:"id"`
		TasklistID int64 `json:"taskListId"`
		// ParentTaskID is a pointer so the zero that detaches is distinguishable
		// from "leave the link alone", which is what omitting the field means.
		ParentTaskID *int64 `json:"parentTaskId,omitempty"`
	}
	payload := struct {
		TodoItem todoItem `json:"todo-item"`
	}{TodoItem: todoItem{ID: t.TaskID, TasklistID: t.TasklistID}}
	if t.DetachFromParent {
		payload.TodoItem.ParentTaskID = new(int64(0))
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("failed to encode move task request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uri, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// taskMoveV1Response carries the IDs the cascade touched, as a comma-separated
// string.
type taskMoveV1Response struct {
	AffectedTaskIDs string `json:"affectedTaskIds"`
	Status          string `json:"STATUS"`
}

// HandleHTTPResponse handles the HTTP response for the taskMoveV1Response.
func (t *taskMoveV1Response) HandleHTTPResponse(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		return twapi.NewHTTPError(resp, "failed to move task")
	}
	if err := json.NewDecoder(resp.Body).Decode(t); err != nil {
		return fmt.Errorf("failed to decode move task response: %w", err)
	}
	return nil
}

// affected parses the comma-separated affectedTaskIds into IDs, skipping
// anything unparseable rather than failing a move that already happened.
func (t *taskMoveV1Response) affected() []int64 {
	var ids []int64
	for _, field := range strings.Split(t.AffectedTaskIDs, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(field), 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

// TaskMove moves tasks, along with every subtask beneath them, to another
// tasklist in Teamwork.com.
//
// The move itself is one request per task, because the v1 endpoint carries the
// whole subtree. What the caller cannot do cheaply is decide whether a task
// needs detaching first: the API rejects a subtask whose tasklist would differ
// from its parent's, and discovering that rule costs a failed call. This
// resolves each task, detaches only the ones whose parent stays behind, and
// drops any task an earlier cascade already carried.
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

			requested := make(map[int64]bool, len(taskIDs))
			roots := make([]int64, 0, len(taskIDs))
			for _, id := range taskIDs {
				if !requested[id] {
					requested[id] = true
					roots = append(roots, id)
				}
			}

			// Resolving each task is memoised because the ancestor walk below
			// revisits the same parents across siblings.
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

			for _, id := range roots {
				if _, err := resolve(id); err != nil {
					return helpers.HandleAPIError(err, fmt.Sprintf("failed to load task %d", id))
				}
			}

			// Naming both a task and one of its descendants would move the
			// descendant twice, and the second move would detach it from the parent
			// that just carried it across. Only the topmost of each pair is moved.
			// Skipped when a single task is requested, where nothing can be an
			// ancestor of anything else, to keep that case at one GET.
			descendsFromRequested := func(id int64) (bool, error) {
				if len(roots) == 1 {
					return false, nil
				}
				for depth := 0; depth < taskMoveMaxDepth; depth++ {
					task, err := resolve(id)
					if err != nil {
						return false, err
					}
					if task.ParentTask == nil {
						return false, nil
					}
					if requested[task.ParentTask.ID] {
						return true, nil
					}
					id = task.ParentTask.ID
				}
				return false, nil
			}

			var moved, detached, alreadyThere, carried []int64
			var failures []string
			movedByCascade := make(map[int64]bool)

			for _, id := range roots {
				task := tasks[id]

				if task.Tasklist.ID == tasklistID {
					alreadyThere = append(alreadyThere, id)
					continue
				}
				if movedByCascade[id] {
					carried = append(carried, id)
					continue
				}
				nested, err := descendsFromRequested(id)
				if err != nil {
					return helpers.HandleAPIError(err, fmt.Sprintf("failed to load the parents of task %d", id))
				}
				if nested {
					carried = append(carried, id)
					continue
				}

				// Any parent this task still has is staying behind: a requested
				// ancestor would have sent it down the carried path above.
				moveRequest := taskMoveV1Request{
					TaskID:           id,
					TasklistID:       tasklistID,
					DetachFromParent: task.ParentTask != nil,
				}
				moveResponse, err := twapi.Execute[taskMoveV1Request, *taskMoveV1Response](ctx, engine, moveRequest)
				if err != nil {
					failures = append(failures, fmt.Sprintf("task %d: %s", id, err.Error()))
					continue
				}

				affected := moveResponse.affected()
				for _, affectedID := range affected {
					movedByCascade[affectedID] = true
				}
				moved = append(moved, affected...)
				if moveRequest.DetachFromParent {
					detached = append(detached, id)
				}
			}

			var report strings.Builder
			fmt.Fprintf(&report, "Moved %d tasks to tasklist %d.", len(moved), tasklistID)
			if len(moved) > 0 {
				fmt.Fprintf(&report, "\nMoved, including subtasks carried along: %s.", joinTaskIDs(moved))
			}
			if len(detached) > 0 {
				fmt.Fprintf(&report, "\nDetached from a parent left behind, now top-level: %s.",
					joinTaskIDs(detached))
			}
			if len(carried) > 0 {
				fmt.Fprintf(&report, "\nAlready carried by another requested task: %s.", joinTaskIDs(carried))
			}
			if len(alreadyThere) > 0 {
				fmt.Fprintf(&report, "\nAlready in the destination tasklist: %s.", joinTaskIDs(alreadyThere))
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
					"fields": helpers.FieldsSchema("task"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(taskGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskGetRequest projects.TaskGetRequest
			taskGetRequest.Filters.IncludeRelatedTasks = true

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
				// they would return the bulk it exists to avoid.
				taskGetRequest.Filters = projects.TaskRequestFilters{}
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
				"For keyword search use search.",
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
					"tag_ids":        helpers.TagIDsFilterSchema("tasks"),
					"match_all_tags": helpers.MatchAllTagsSchema(),
					"created_after":  helpers.DateTimeFilterSchema("Filter tasks created after."),
					"created_before": helpers.DateTimeFilterSchema("Filter tasks created before."),
					"created_by_user_ids": {
						Description: "Filter tasks by creator.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"updated_after":    helpers.DateTimeFilterSchema("Filter tasks updated after."),
					"updated_before":   helpers.DateTimeFilterSchema("Filter tasks updated before."),
					"completed_after":  helpers.DateTimeFilterSchema("Filter tasks completed after."),
					"completed_before": helpers.DateTimeFilterSchema("Filter tasks completed before."),
					"due_after": {
						Description: "Filter tasks due after.",
						Examples:    []any{"2023-01-01"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"due_before": {
						Description: "Filter tasks due before.",
						Examples:    []any{"2023-12-31"},
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
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
					"fields":    helpers.FieldsSchema("task"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(taskListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskListRequest projects.TaskListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var showCompleted *bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&taskListRequest.Path.TasklistID, "tasklist_id"),
				helpers.OptionalNumericParam(&taskListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&taskListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.AssigneeUserIDs, "assignee_user_ids"),
				helpers.OptionalNumericListParam(&taskListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.MatchAllTags, "match_all_tags"),
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
				helpers.OptionalDatePointerParam(&taskListRequest.Filters.DueAfter, "due_after"),
				helpers.OptionalDatePointerParam(&taskListRequest.Filters.DueBefore, "due_before"),
				helpers.OptionalPointerParam(&showCompleted, "show_completed"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.OnlyUnassigned, "only_unassigned"),
				helpers.OptionalPointerParam(&taskListRequest.Filters.OnlyUnplanned, "only_unplanned"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalFieldsParam[projects.Task](&taskListRequest.Filters.Fields.Tasks, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if showCompleted != nil {
				// A single flag drives both SDK filters: completed tasks are hidden by
				// default, and so are tasks living inside a completed tasklist. Callers
				// asking to see completed work expect both.
				taskListRequest.Filters.IncludeCompletedTasks = showCompleted
				taskListRequest.Filters.IncludeTasksFromCompletedTasklists = showCompleted
			}
			switch {
			case len(taskListRequest.Filters.Fields.Tasks) > 0:
				// An explicit field selection overrides both defaults below: the
				// caller has already said what it wants, and sideloading custom
				// fields would smuggle back the bulk the selection exists to avoid.

			case verbose:
				// Include custom fields and values in task list response for richer
				// context, as they are commonly used in Teamwork projects and provide
				// valuable information about the task.
				taskListRequest.Filters.Include = []projects.TaskRequestSideload{
					projects.TaskRequestSideloadCustomFields,
					projects.TaskRequestSideloadCustomFieldValues,
				}
				// Ensure predecessors are included in the response
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
