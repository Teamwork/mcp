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
	MethodUsersWorkload toolsets.Method = "twprojects-users_workload"
)

const workloadDescription = "Workload is a visual representation of how tasks are distributed across team members, " +
	"helping you understand who is overloaded, who has capacity, and how work is balanced within a project or " +
	"across multiple projects. It takes into account assigned tasks, due dates, estimated time, and working " +
	"hours to give managers and teams a clear picture of availability and resource allocation. By providing " +
	"this insight, workload makes it easier to plan effectively, prevent burnout, and ensure that deadlines are " +
	"met without placing too much pressure on any single person."

var (
	userWorkloadOutputSchema *jsonschema.Schema
)

func init() {
	toolsets.RegisterMethod(MethodUsersWorkload)

	var err error

	// generate the output schemas only once
	userWorkloadOutputSchema, err = jsonschema.For[projects.WorkloadResponse](&jsonschema.ForOptions{
		IgnoreInvalidTypes: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for WorkloadResponse: %v", err))
	}
}

// UsersWorkload retrieves the workload of users in Teamwork.com.
func UsersWorkload(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUsersWorkload),
			Description: "Get the workload of users in Teamwork.com. " + workloadDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Users Workload",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"start_date": {
						Type:        "string",
						Format:      "date",
						Description: "The start date of the workload period. The date must be in the format YYYY-MM-DD.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date",
						Description: "The end date of the workload period. The date must be in the format YYYY-MM-DD.",
					},
					"user_ids": {
						Type:        "array",
						Description: "List of user IDs to filter the workload by.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"user_company_ids": {
						Type:        "array",
						Description: "List of users' client/company IDs to filter the workload by.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"user_team_ids": {
						Type:        "array",
						Description: "List of users' team IDs to filter the workload by.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"project_ids": {
						Type:        "array",
						Description: "List of project IDs to filter the workload by.",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredDateParam(&workloadRequest.Filters.StartDate, "start_date"),
				helpers.RequiredDateParam(&workloadRequest.Filters.EndDate, "end_date"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.UserIDs, "user_ids"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.UserCompanyIDs, "user_company_ids"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.UserTeamIDs, "user_team_ids"),
				helpers.OptionalNumericListParam(&workloadRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericParam(&workloadRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&workloadRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			workload, err := projects.WorkloadGet(ctx, engine, workloadRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get workload")
			}
			return helpers.NewToolResultJSON(workload)
		},
	}
}
