package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodTaskSplitGet toolsets.Method = "twprojects-get_task_split"
	MethodTaskSplitSet toolsets.Method = "twprojects-set_task_split"
)

// taskSplitMaxPages bounds the pages get_task_split walks. A page holds 50
// rows (one per user and day); a longer split is reported as truncated.
const taskSplitMaxPages = 20

// taskSplit is a task's custom splits, one row per user and day.
type taskSplit struct {
	TaskID     int64                   `json:"taskId"`
	Capacities []projects.TaskCapacity `json:"capacities"`
	// Truncated is set when the page bound stopped the read early.
	Truncated bool `json:"truncated,omitempty"`
}

var taskSplitOutputSchema *jsonschema.Schema

func init() {
	var err error
	taskSplitOutputSchema, err = jsonschema.For[taskSplit](helpers.WithDateTypeSchema(&jsonschema.ForOptions{}))
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for taskSplit: %v", err))
	}
}

// TaskSplitGet reads the custom per-day splits of a task in Teamwork.com.
func TaskSplitGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTaskSplitGet),
			Description: "Get a task's custom capacity split: the minutes each assignee spends on it per day in " +
				"Workload. A user with no rows is on the default even spread. When seconds is above zero it is " +
				"what Workload counts. truncated means more rows exist; pass user_ids to read fewer.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Task Split",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the task.",
					},
					"user_ids": {
						Description: "Only these users' splits. Omit for every user assigned to the task.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"task_id"},
			},
			OutputSchema: taskSplitOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskID int64
			var userIDs []int64

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskID, "task_id"),
				helpers.OptionalNumericListParam(&userIDs, "user_ids"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// the endpoint defaults to the caller's own rows, so every assignee
			// has to be named
			if len(userIDs) == 0 {
				taskRequest := projects.NewTaskGetRequest(taskID)
				taskRequest.Fields.Task = []projects.TaskField{projects.TaskFieldAssignees}
				task, err := projects.TaskGet(ctx, engine, taskRequest)
				if err != nil {
					return helpers.HandleAPIError(err, "failed to get task assignees")
				}
				// Workload counts a task for its user assignees only, so a split
				// held by a member of an assigned team has no effect there
				for _, assignee := range task.Task.Assignees {
					if assignee.Type == "users" {
						userIDs = append(userIDs, assignee.ID)
					}
				}
			}

			result := taskSplit{TaskID: taskID, Capacities: []projects.TaskCapacity{}}
			if len(userIDs) == 0 {
				return helpers.NewToolResultJSON(result)
			}

			listRequest := projects.NewTaskCapacityListRequest(taskID, userIDs...)
			for page := range taskSplitMaxPages {
				response, err := projects.TaskCapacityList(ctx, engine, listRequest)
				if err != nil {
					return helpers.HandleAPIError(err, "failed to get task split")
				}
				result.Capacities = append(result.Capacities, response.Capacities...)
				next := response.Iterate()
				if next == nil {
					break
				}
				if page == taskSplitMaxPages-1 {
					result.Truncated = true
					break
				}
				listRequest = *next
			}
			return helpers.NewToolResultJSON(result)
		},
	}
}

// TaskSplitSet sets or clears a user's custom per-day split of a task in
// Teamwork.com.
func TaskSplitSet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodTaskSplitSet),
			Description: "Set how many minutes one assignee spends on a task each day in Workload, replacing the " +
				"default even spread. Use it to move load off an over-capacity day without changing the task's " +
				"dates. Sends the whole split: a day of the task's range left out counts as zero, and the " +
				"minutes need not add up to the estimate. The task needs an estimate and a due date, and every " +
				"date must fall within its start and due dates. Changing the task's estimate or assignees later " +
				"removes the split; changing its dates moves it with them.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set Task Split",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_id": {
						Type:        "integer",
						Minimum:     new(1.0),
						Description: "The ID of the task.",
					},
					"user_id": {
						Type:    "integer",
						Minimum: new(1.0),
						Description: "The assignee the split applies to. Workload counts only direct assignees, " +
							"not members of an assigned team.",
					},
					"dates": {
						Description: "The split, one entry per day. Required unless clear is true.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"date": {
											Type:        "string",
											Format:      "date",
											Description: "The day, as YYYY-MM-DD.",
										},
										"minutes": {
											Type:        "integer",
											Minimum:     new(0.0),
											Description: "Minutes on the task that day.",
										},
									},
									Required: []string{"date", "minutes"},
								},
							},
							{Type: "null"},
						},
					},
					"clear": {
						Description: "If true, remove the user's split and return the task to the even spread.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
						Default: []byte(`false`),
					},
				},
				Required: []string{"task_id", "user_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var taskID, userID int64
			var clear bool

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&taskID, "task_id"),
				helpers.RequiredNumericParam(&userID, "user_id"),
				helpers.OptionalParam(&clear, "clear"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			dates, errResult := parseTaskSplitDates(arguments)
			if errResult != nil {
				return errResult, nil
			}

			switch {
			case clear && len(dates) > 0:
				return helpers.NewToolResultTextError("invalid parameters: send either dates or clear, not both"), nil
			case clear:
				_, err := projects.TaskCapacityDelete(ctx, engine, projects.NewTaskCapacityDeleteRequest(taskID, userID))
				if err != nil {
					return helpers.HandleAPIError(err, "failed to clear task split")
				}
				return helpers.NewToolResultText(
					"Split removed: Workload spreads the estimate evenly over the task's dates again."), nil
			case len(dates) == 0:
				return helpers.NewToolResultTextError("invalid parameters: dates is required unless clear is true"), nil
			}

			capacities, err := writeTaskSplit(ctx, engine, taskID, userID, dates)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to set task split")
			}
			if len(capacities) == 0 {
				return helpers.NewToolResultText(
					"The split equals the even spread of the estimate, so no custom split is stored."), nil
			}
			var total int64
			for _, capacity := range capacities {
				total += capacity.Minutes
			}
			return helpers.NewToolResultJSON(struct {
				taskSplit
				TotalMinutes int64 `json:"totalMinutes"`
			}{taskSplit{TaskID: taskID, Capacities: capacities}, total})
		},
	}
}

// writeTaskSplit replaces the user's split, creating it when there is none: the
// replace route answers 404 for a user without one, and the create route 400
// for a user who has one.
func writeTaskSplit(
	ctx context.Context,
	engine *twapi.Engine,
	taskID, userID int64,
	dates []projects.TaskCapacityDate,
) ([]projects.TaskCapacity, error) {
	replaced, err := projects.TaskCapacityReplace(ctx, engine,
		projects.NewTaskCapacityReplaceRequest(taskID, userID, dates))
	if err == nil {
		return replaced.Capacities, nil
	}
	if httpErr, ok := errors.AsType[*twapi.HTTPError](err); !ok || httpErr.StatusCode != http.StatusNotFound {
		return nil, err
	}
	created, err := projects.TaskCapacityCreate(ctx, engine,
		projects.NewTaskCapacityCreateRequest(taskID, userID, dates))
	if err != nil {
		return nil, err
	}
	return created.Capacities, nil
}

// parseTaskSplitDates reads the dates argument of set_task_split.
func parseTaskSplitDates(arguments map[string]any) ([]projects.TaskCapacityDate, *mcp.CallToolResult) {
	raw, _ := arguments["dates"].([]any)
	dates := make([]projects.TaskCapacityDate, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for i, entry := range raw {
		item, ok := entry.(map[string]any)
		if !ok {
			return nil, helpers.NewToolResultTextError("invalid parameters: dates[%d] must be an object", i)
		}
		day, _ := item["date"].(string)
		parsed, err := time.Parse(time.DateOnly, day)
		if err != nil {
			return nil, helpers.NewToolResultTextError(
				"invalid parameters: dates[%d].date must be a YYYY-MM-DD date", i)
		}
		if seen[day] {
			return nil, helpers.NewToolResultTextError("invalid parameters: dates lists %s more than once", day)
		}
		seen[day] = true
		minutes, ok := item["minutes"].(float64)
		if !ok || minutes < 0 || minutes != float64(int64(minutes)) {
			return nil, helpers.NewToolResultTextError(
				"invalid parameters: dates[%d].minutes must be a whole number of minutes, zero or more", i)
		}
		dates = append(dates, projects.TaskCapacityDate{Date: twapi.Date(parsed), Minutes: int64(minutes)})
	}
	return dates, nil
}
