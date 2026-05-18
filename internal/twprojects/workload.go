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

var (
	userWorkloadOutputSchema *jsonschema.Schema
)

func init() {
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
			Description: "Get task allocation across users for a date range. (workload of users)",
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
						Description: "Start of the workload period.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date",
						Description: "End of the workload period.",
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			workload, err := projects.WorkloadGet(ctx, engine, workloadRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get workload")
			}
			return helpers.NewToolResultJSON(workload)
		},
	}
}
