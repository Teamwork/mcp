package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

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
	MethodUsersWorkload toolsets.Method = "twprojects-users_workload"
)

var (
	userWorkloadOutputSchema *jsonschema.Schema
)

// workloadTask is the part of a sideloaded task that explains a workload day.
// The full task record would multiply the response for every task in range.
type workloadTask struct {
	ID               int64                `json:"id"`
	Name             string               `json:"name"`
	StartAt          *twapi.Date          `json:"startDate"`
	DueAt            *twapi.Date          `json:"dueDate"`
	EstimatedMinutes int64                `json:"estimateMinutes"`
	Assignees        []twapi.Relationship `json:"assignees"`
}

func init() {
	var err error

	// generate the output schemas only once
	userWorkloadOutputSchema, err = jsonschema.For[projects.WorkloadResponse](
		helpers.WithDateTypeSchema(&jsonschema.ForOptions{IgnoreInvalidTypes: true}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for WorkloadResponse: %v", err))
	}
	workloadTasksSchema, err := jsonschema.For[map[string]workloadTask](
		helpers.WithDateTypeSchema(&jsonschema.ForOptions{}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for workload tasks: %v", err))
	}
	userWorkloadOutputSchema.Properties["included"].Properties["tasks"] = workloadTasksSchema
}

// UsersWorkload retrieves the workload of users in Teamwork.com.
func UsersWorkload(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodUsersWorkload),
			Description: "Get each user's planned task time per day, as the Workload view shows it. " +
				"capacityMinutes is the day's load (task estimates plus time off) and capacity is that as a " +
				"percentage of the user's working hours; over 100 is over capacity. Days with nothing planned " +
				"are omitted; workingHourEntries gives each weekday's hours. A task counts for its directly " +
				"assigned users when it has an estimate and a due date: the estimate is split between those " +
				"assignees and spread evenly over each one's working days from start to due date (all on the " +
				"due date without a start date), unless the user has a custom split (twprojects-set_task_split). " +
				"Allocations are not included.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Users Workload",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"start_date": {
						Type:        "string",
						Format:      "date",
						Description: "Start of the workload period; the boundary day itself is included.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date",
						Description: "End of the workload period; the boundary day itself is included.",
					},
					"user_ids": {
						Description: "Filter workload by user.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"user_company_ids": {
						Description: "Filter workload by users' client/company.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"user_team_ids": {
						Description: "Filter workload by users' team.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "Filter workload by project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"include_tasks": {
						Description: "If true, also return the counted tasks (dates, estimate, assignees) and their " +
							"custom splits, to see what fills each day before rescheduling. Narrow with user_ids.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
						Default: []byte(`false`),
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
				},
				Required: []string{"start_date", "end_date"},
			},
			OutputSchema: userWorkloadOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workloadRequest projects.WorkloadRequest
			workloadRequest.Filters.Include = []projects.WorkloadGetRequestSideload{
				projects.WorkloadGetRequestSideloadWorkingHourEntries,
			}

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			var includeTasks bool
			err := helpers.ParamGroup(arguments,
				helpers.RequiredDateParam(&workloadRequest.Filters.StartDate, "start_date"),
				helpers.RequiredDateParam(&workloadRequest.Filters.EndDate, "end_date"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.UserIDs, "user_ids"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.UserCompanyIDs, "user_company_ids"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.UserTeamIDs, "user_team_ids"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalParam(&includeTasks, "include_tasks"),
				helpers.OptionalNumericParam(&workloadRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&workloadRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if includeTasks {
				workloadRequest.Filters.Include = append(workloadRequest.Filters.Include,
					projects.WorkloadGetRequestSideloadTaskCapacities)
			}

			workload, err := projects.WorkloadGet(ctx, engine, workloadRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get workload")
			}
			if !includeTasks {
				return helpers.NewToolResultJSON(workload)
			}
			return workloadWithSlimTasks(workload)
		},
	}
}

// workloadWithSlimTasks answers the workload with each sideloaded task cut down
// to workloadTask.
func workloadWithSlimTasks(workload *projects.WorkloadResponse) (*mcp.CallToolResult, error) {
	tasks := make(map[string]workloadTask, len(workload.Included.Tasks))
	for key, task := range workload.Included.Tasks {
		tasks[key] = workloadTask{
			ID:               task.ID,
			Name:             task.Name,
			StartAt:          task.StartAt,
			DueAt:            task.DueAt,
			EstimatedMinutes: task.EstimatedMinutes,
			Assignees:        task.Assignees,
		}
	}
	workload.Included.Tasks = nil

	encoded, err := json.Marshal(workload)
	if err != nil {
		return helpers.NewToolResultTextError("failed to encode workload: %s", err.Error()), nil
	}
	var result map[string]any
	if err := json.Unmarshal(encoded, &result); err != nil {
		return helpers.NewToolResultTextError("failed to decode workload: %s", err.Error()), nil
	}
	if included, ok := result["included"].(map[string]any); ok && len(tasks) > 0 {
		included["tasks"] = tasks
	}
	return helpers.NewToolResultJSON(result)
}
