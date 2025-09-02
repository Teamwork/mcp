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
	MethodTimelogCreate        toolsets.Method = "twprojects-create_timelog"
	MethodTimelogUpdate        toolsets.Method = "twprojects-update_timelog"
	MethodTimelogDelete        toolsets.Method = "twprojects-delete_timelog"
	MethodTimelogGet           toolsets.Method = "twprojects-get_timelog"
	MethodTimelogList          toolsets.Method = "twprojects-list_timelogs"
	MethodTimelogListByProject toolsets.Method = "twprojects-list_timelogs_by_project"
	MethodTimelogListByTask    toolsets.Method = "twprojects-list_timelogs_by_task"
)

const timelogDescription = "Timelog refers to a recorded entry that tracks the amount of time a person has spent " +
	"working on a specific task, project, or piece of work. These entries typically include details such as the " +
	"duration of time worked, the date and time it was logged, who logged it, and any optional notes describing what " +
	"was done during that period. Timelogs are essential for understanding how time is being allocated across " +
	"projects, enabling teams to manage resources more effectively, invoice clients accurately, and assess " +
	"productivity. They can be created manually or with timers, and are often used for reporting and billing purposes."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTimelogCreate)
	toolsets.RegisterMethod(MethodTimelogUpdate)
	toolsets.RegisterMethod(MethodTimelogDelete)
	toolsets.RegisterMethod(MethodTimelogGet)
	toolsets.RegisterMethod(MethodTimelogList)
	toolsets.RegisterMethod(MethodTimelogListByProject)
	toolsets.RegisterMethod(MethodTimelogListByTask)
}

// TimelogCreate creates a timelog in Teamwork.com.
func TimelogCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogCreate),
			mcp.WithDescription("Create a new timelog in Teamwork.com. "+timelogDescription),
			mcp.WithString("description",
				mcp.Description("A description of the timelog."),
			),
			mcp.WithString("date",
				mcp.Required(),
				mcp.Description("The date of the timelog in the format YYYY-MM-DD."),
			),
			mcp.WithString("time",
				mcp.Required(),
				mcp.Description("The time of the timelog in the format HH:MM:SS."),
			),
			mcp.WithBoolean("is_utc",
				mcp.Description("If true, the time is in UTC. Defaults to false."),
			),
			mcp.WithNumber("hours",
				mcp.Required(),
				mcp.Description("The number of hours spent on the timelog. Must be a positive integer."),
			),
			mcp.WithNumber("minutes",
				mcp.Required(),
				mcp.Description("The number of minutes spent on the timelog. Must be a positive integer less than 60, "+
					"otherwise the hours attribute should be incremented."),
			),
			mcp.WithBoolean("billable",
				mcp.Description("If true, the timelog is billable. Defaults to false."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project to associate the timelog with. "+
					"Either project_id or task_id must be provided, but not both."),
			),
			mcp.WithNumber("task_id",
				mcp.Description("The ID of the task to associate the timelog with. "+
					"Either project_id or task_id must be provided, but not both."),
			),
			mcp.WithNumber("user_id",
				mcp.Description("The ID of the user to associate the timelog with. "+
					"Defaults to the authenticated user if not provided."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the timelog."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogCreateRequest projects.TimelogCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalNumericParam(&timelogCreateRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericParam(&timelogCreateRequest.Path.TaskID, "task_id"),
				helpers.OptionalPointerParam(&timelogCreateRequest.Description, "description"),
				helpers.RequiredDateParam(&timelogCreateRequest.Date, "date"),
				helpers.RequiredTimeOnlyParam(&timelogCreateRequest.Time, "time"),
				helpers.OptionalParam(&timelogCreateRequest.IsUTC, "is_utc"),
				helpers.RequiredNumericParam(&timelogCreateRequest.Hours, "hours"),
				helpers.RequiredNumericParam(&timelogCreateRequest.Minutes, "minutes"),
				helpers.OptionalParam(&timelogCreateRequest.Billable, "billable"),
				helpers.OptionalNumericPointerParam(&timelogCreateRequest.UserID, "user_id"),
				helpers.OptionalNumericListParam(&timelogCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timelogResponse, err := projects.TimelogCreate(ctx, engine, timelogCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create timelog")
			}

			id := timelogResponse.Timelog.ID
			return mcp.NewToolResultText(fmt.Sprintf("Timelog created successfully with ID %d", id)), nil
		},
	}
}

// TimelogUpdate updates a timelog in Teamwork.com.
func TimelogUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogUpdate),
			mcp.WithDescription("Update an existing timelog in Teamwork.com. "+timelogDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timelog to update."),
			),
			mcp.WithString("description",
				mcp.Description("A description of the timelog."),
			),
			mcp.WithString("date",
				mcp.Description("The date of the timelog in the format YYYY-MM-DD."),
			),
			mcp.WithString("time",
				mcp.Description("The time of the timelog in the format HH:MM:SS."),
			),
			mcp.WithBoolean("is_utc",
				mcp.Description("If true, the time is in UTC."),
			),
			mcp.WithNumber("hours",
				mcp.Description("The number of hours spent on the timelog. Must be a positive integer."),
			),
			mcp.WithNumber("minutes",
				mcp.Description("The number of minutes spent on the timelog. Must be a positive integer less than 60, "+
					"otherwise the hours attribute should be incremented."),
			),
			mcp.WithBoolean("billable",
				mcp.Description("If true, the timelog is billable."),
			),
			mcp.WithNumber("user_id",
				mcp.Description("The ID of the user to associate the timelog with."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the timelog."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogUpdateRequest projects.TimelogUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timelogUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&timelogUpdateRequest.Description, "description"),
				helpers.OptionalDatePointerParam(&timelogUpdateRequest.Date, "date"),
				helpers.OptionalTimeOnlyPointerParam(&timelogUpdateRequest.Time, "time"),
				helpers.OptionalPointerParam(&timelogUpdateRequest.IsUTC, "is_utc"),
				helpers.OptionalNumericPointerParam(&timelogUpdateRequest.Hours, "hours"),
				helpers.OptionalNumericPointerParam(&timelogUpdateRequest.Minutes, "minutes"),
				helpers.OptionalPointerParam(&timelogUpdateRequest.Billable, "billable"),
				helpers.OptionalNumericPointerParam(&timelogUpdateRequest.UserID, "user_id"),
				helpers.OptionalNumericListParam(&timelogUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimelogUpdate(ctx, engine, timelogUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update timelog")
			}

			return mcp.NewToolResultText("Timelog updated successfully"), nil
		},
	}
}

// TimelogDelete deletes a timelog in Teamwork.com.
func TimelogDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogDelete),
			mcp.WithDescription("Delete an existing timelog in Teamwork.com. "+timelogDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timelog to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogDeleteRequest projects.TimelogDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timelogDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimelogDelete(ctx, engine, timelogDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete timelog")
			}

			return mcp.NewToolResultText("Timelog deleted successfully"), nil
		},
	}
}

// TimelogGet retrieves a timelog in Teamwork.com.
func TimelogGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogGet),
			mcp.WithDescription("Get an existing timelog in Teamwork.com. "+timelogDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timelog to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogGetRequest projects.TimelogGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timelogGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timelog, err := projects.TimelogGet(ctx, engine, timelogGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get timelog")
			}

			encoded, err := json.Marshal(timelog)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}

// TimelogList lists timelogs in Teamwork.com.
func TimelogList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogList),
			mcp.WithDescription("List timelogs in Teamwork.com. "+timelogDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter timelogs by tags"),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match timelogs that have all the specified tags. "+
					"If false, the search will match timelogs that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithString("start_date",
				mcp.Description("Start date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithString("end_date",
				mcp.Description("End date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithArray("assigned_user_ids",
				mcp.Description("A list of user IDs to filter timelogs by assigned users"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("assigned_company_ids",
				mcp.Description("A list of company IDs to filter timelogs by assigned companies"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("assigned_team_ids",
				mcp.Description("A list of team IDs to filter timelogs by assigned teams"),
				mcp.Items(map[string]any{
					"type": "integer",
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
			var timelogListRequest projects.TimelogListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&timelogListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalTimePointerParam(&timelogListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimePointerParam(&timelogListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToUserIDs, "assigned_user_ids"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToCompanyIDs, "assigned_company_ids"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToTeamIDs, "assigned_team_ids"),
				helpers.OptionalNumericParam(&timelogListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&timelogListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timelogList, err := projects.TimelogList(ctx, engine, timelogListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timelogs")
			}

			encoded, err := json.Marshal(timelogList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}

// TimelogListByProject lists timelogs in Teamwork.com by project.
func TimelogListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogListByProject),
			mcp.WithDescription("List timelogs in Teamwork.com by project. "+timelogDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project from which to retrieve timelogs."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter timelogs by tags"),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match timelogs that have all the specified tags. "+
					"If false, the search will match timelogs that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithString("start_date",
				mcp.Description("Start date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithString("end_date",
				mcp.Description("End date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithArray("assigned_user_ids",
				mcp.Description("A list of user IDs to filter timelogs by assigned users"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("assigned_company_ids",
				mcp.Description("A list of company IDs to filter timelogs by assigned companies"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("assigned_team_ids",
				mcp.Description("A list of team IDs to filter timelogs by assigned teams"),
				mcp.Items(map[string]any{
					"type": "integer",
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
			var timelogListRequest projects.TimelogListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timelogListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&timelogListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalTimePointerParam(&timelogListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimePointerParam(&timelogListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToUserIDs, "assigned_user_ids"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToCompanyIDs, "assigned_company_ids"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToTeamIDs, "assigned_team_ids"),
				helpers.OptionalNumericParam(&timelogListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&timelogListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timelogList, err := projects.TimelogList(ctx, engine, timelogListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timelogs")
			}

			encoded, err := json.Marshal(timelogList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}

// TimelogListByTask lists timelogs in Teamwork.com by task.
func TimelogListByTask(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimelogListByTask),
			mcp.WithDescription("List timelogs in Teamwork.com by task. "+timelogDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("task_id",
				mcp.Required(),
				mcp.Description("The ID of the task from which to retrieve timelogs."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter timelogs by tags"),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match timelogs that have all the specified tags. "+
					"If false, the search will match timelogs that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithString("start_date",
				mcp.Description("Start date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithString("end_date",
				mcp.Description("End date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ."),
			),
			mcp.WithArray("assigned_user_ids",
				mcp.Description("A list of user IDs to filter timelogs by assigned users"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("assigned_company_ids",
				mcp.Description("A list of company IDs to filter timelogs by assigned companies"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("assigned_team_ids",
				mcp.Description("A list of team IDs to filter timelogs by assigned teams"),
				mcp.Items(map[string]any{
					"type": "integer",
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
			var timelogListRequest projects.TimelogListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timelogListRequest.Path.TaskID, "task_id"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&timelogListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalTimePointerParam(&timelogListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimePointerParam(&timelogListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToUserIDs, "assigned_user_ids"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToCompanyIDs, "assigned_company_ids"),
				helpers.OptionalNumericListParam(&timelogListRequest.Filters.AssignedToTeamIDs, "assigned_team_ids"),
				helpers.OptionalNumericParam(&timelogListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&timelogListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timelogList, err := projects.TimelogList(ctx, engine, timelogListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timelogs")
			}

			encoded, err := json.Marshal(timelogList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}
