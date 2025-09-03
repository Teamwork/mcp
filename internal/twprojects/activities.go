package twprojects

import (
	"context"
	"encoding/json"

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
	MethodActivityList          toolsets.Method = "twprojects-list_activities"
	MethodActivityListByProject toolsets.Method = "twprojects-list_activities_by_project"
)

const activityDescription = "Activity is a record of actions and updates that occur across your projects, tasks, and " +
	"communications, giving you a clear view of what’s happening within your workspace. Activities capture changes " +
	"such as task completions, activities added, files uploaded, or milestones updated, and present them in a " +
	"chronological feed so teams can stay aligned without needing to check each individual project or task. This " +
	"stream of information helps improve transparency, ensures accountability, and keeps everyone aware of progress " +
	"and decisions as they happen."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodActivityList)
	toolsets.RegisterMethod(MethodActivityListByProject)
}

// ActivityList lists activities in Teamwork.com.
func ActivityList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodActivityList),
			mcp.WithDescription("List activities in Teamwork.com. "+activityDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithString("start_date",
				mcp.Description("Start date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithString("end_date",
				mcp.Description("End date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithArray("log_item_types",
				mcp.Description("Filter activities by item types."),
				mcp.Items(map[string]any{
					"type": "string",
					"enum": []any{
						"message",
						"comment",
						"task",
						"tasklist",
						"taskgroup",
						"milestone",
						"file",
						"form",
						"notebook",
						"timelog",
						"task_comment",
						"notebook_comment",
						"file_comment",
						"link_comment",
						"milestone_comment",
						"project",
						"link",
						"billingInvoice",
						"risk",
						"projectUpdate",
						"reacted",
						"budget",
					},
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
			var activityListRequest projects.ActivityListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalTimeParam(&activityListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalListParam(&activityListRequest.Filters.LogItemTypes, "log_item_types"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			activityList, err := projects.ActivityList(ctx, engine, activityListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list activities")
			}

			encoded, err := json.Marshal(activityList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}

// ActivityListByProject lists activities by project in Teamwork.com.
func ActivityListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodActivityListByProject),
			mcp.WithDescription("List activities in Teamwork.com by project. "+activityDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to retrieve activities from."),
			),
			mcp.WithString("start_date",
				mcp.Description("Start date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithString("end_date",
				mcp.Description("End date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithArray("log_item_types",
				mcp.Description("Filter activities by item types."),
				mcp.Items(map[string]any{
					"type": "string",
					"enum": []any{
						"message",
						"comment",
						"task",
						"tasklist",
						"taskgroup",
						"milestone",
						"file",
						"form",
						"notebook",
						"timelog",
						"task_comment",
						"notebook_comment",
						"file_comment",
						"link_comment",
						"milestone_comment",
						"project",
						"link",
						"billingInvoice",
						"risk",
						"projectUpdate",
						"reacted",
						"budget",
					},
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
			var activityListRequest projects.ActivityListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&activityListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalListParam(&activityListRequest.Filters.LogItemTypes, "log_item_types"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			activityList, err := projects.ActivityList(ctx, engine, activityListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list activities")
			}

			encoded, err := json.Marshal(activityList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}
