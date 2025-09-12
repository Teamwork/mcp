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
	MethodUsersWorkload toolsets.Method = "twprojects-users_workload"
)

const workloadDescription = "Workload is a visual representation of how tasks are distributed across team members, " +
	"helping you understand who is overloaded, who has capacity, and how work is balanced within a project or " +
	"across multiple projects. It takes into account assigned tasks, due dates, estimated time, and working " +
	"hours to give managers and teams a clear picture of availability and resource allocation. By providing " +
	"this insight, workload makes it easier to plan effectively, prevent burnout, and ensure that deadlines are " +
	"met without placing too much pressure on any single person."

// UsersWorkload retrieves the workload of users in Teamwork.com.
func UsersWorkload(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(MethodUsersWorkload.String(),
			mcp.WithDescription(workloadDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithString("start_date",
				mcp.Required(),
				mcp.Description("The start date of the workload period. The date must be in the format YYYY-MM-DD."),
			),
			mcp.WithString("end_date",
				mcp.Required(),
				mcp.Description("The end date of the workload period. The date must be in the format YYYY-MM-DD."),
			),
			mcp.WithArray("user_ids",
				mcp.Description("List of user IDs to filter the workload by."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithArray("user_company_ids",
				mcp.Description("List of users' client/company IDs to filter the workload by."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithArray("user_team_ids",
				mcp.Description("List of users' team IDs to filter the workload by."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithArray("project_ids",
				mcp.Description("List of project IDs to filter the workload by."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workloadRequest projects.WorkloadRequest
			workloadRequest.Filters.Include = []projects.WorkloadGetRequestSideload{
				projects.WorkloadGetRequestSideloadWorkingHourEntries,
			}

			err := helpers.ParamGroup(request.GetArguments(),
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
				return nil, fmt.Errorf("invalid parameters: %w", err)
			}

			workload, err := projects.WorkloadGet(ctx, engine, workloadRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get workload")
			}

			encoded, err := json.Marshal(workload)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}
