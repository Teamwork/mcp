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
	MethodTimerCreate   toolsets.Method = "twprojects-create_timer"
	MethodTimerUpdate   toolsets.Method = "twprojects-update_timer"
	MethodTimerPause    toolsets.Method = "twprojects-pause_timer"
	MethodTimerResume   toolsets.Method = "twprojects-resume_timer"
	MethodTimerComplete toolsets.Method = "twprojects-complete_timer"
	MethodTimerDelete   toolsets.Method = "twprojects-delete_timer"
	MethodTimerGet      toolsets.Method = "twprojects-get_timer"
	MethodTimerList     toolsets.Method = "twprojects-list_timers"
)

const timerDescription = "Timer is a built-in tool that allows users to accurately track the time they spend working " +
	"on specific tasks, projects, or client work. Instead of manually recording hours, users can start, pause, and " +
	"stop timers directly within the platform or through the desktop and mobile apps, ensuring precise time logs " +
	"without interrupting their workflow. Once recorded, these entries are automatically linked to the relevant task " +
	"or project, making it easier to monitor productivity, manage billable hours, and generate detailed reports for " +
	"both internal tracking and client invoicing."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTimerCreate)
	toolsets.RegisterMethod(MethodTimerUpdate)
	toolsets.RegisterMethod(MethodTimerPause)
	toolsets.RegisterMethod(MethodTimerResume)
	toolsets.RegisterMethod(MethodTimerComplete)
	toolsets.RegisterMethod(MethodTimerDelete)
	toolsets.RegisterMethod(MethodTimerGet)
	toolsets.RegisterMethod(MethodTimerList)
}

// TimerCreate creates a timer in Teamwork.com.
func TimerCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerCreate),
			mcp.WithDescription("Create a new timer in Teamwork.com. "+timerDescription),
			mcp.WithString("description",
				mcp.Description("A description of the timer."),
			),
			mcp.WithBoolean("billable",
				mcp.Description("If true, the timer is billable. Defaults to false."),
			),
			mcp.WithBoolean("running",
				mcp.Description("If true, the timer will start running immediately."),
			),
			mcp.WithNumber("seconds",
				mcp.Description("The number of seconds to set the timer for."),
			),
			mcp.WithBoolean("stop_running_timers",
				mcp.Description("If true, any other running timers will be stopped when this timer is created."),
			),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to associate the timer with."),
			),
			mcp.WithNumber("task_id",
				mcp.Description("The ID of the task to associate the timer with."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerCreateRequest projects.TimerCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalPointerParam(&timerCreateRequest.Description, "description"),
				helpers.OptionalPointerParam(&timerCreateRequest.Billable, "billable"),
				helpers.OptionalPointerParam(&timerCreateRequest.Running, "running"),
				helpers.OptionalNumericPointerParam(&timerCreateRequest.Seconds, "seconds"),
				helpers.OptionalPointerParam(&timerCreateRequest.StopRunningTimers, "stop_running_timers"),
				helpers.RequiredNumericParam(&timerCreateRequest.ProjectID, "project_id"),
				helpers.OptionalNumericPointerParam(&timerCreateRequest.TaskID, "task_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timerResponse, err := projects.TimerCreate(ctx, engine, timerCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create timer")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Timer created successfully with ID %d", timerResponse.Timer.ID)), nil
		},
	}
}

// TimerUpdate updates a timer in Teamwork.com.
func TimerUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerUpdate),
			mcp.WithDescription("Update an existing timer in Teamwork.com. "+timerDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timer to update."),
			),
			mcp.WithString("description",
				mcp.Description("A description of the timer."),
			),
			mcp.WithBoolean("billable",
				mcp.Description("If true, the timer is billable."),
			),
			mcp.WithBoolean("running",
				mcp.Description("If true, the timer will start running immediately."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project to associate the timer with."),
			),
			mcp.WithNumber("task_id",
				mcp.Description("The ID of the task to associate the timer with."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerUpdateRequest projects.TimerUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timerUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&timerUpdateRequest.Description, "description"),
				helpers.OptionalPointerParam(&timerUpdateRequest.Billable, "billable"),
				helpers.OptionalPointerParam(&timerUpdateRequest.Running, "running"),
				helpers.OptionalNumericPointerParam(&timerUpdateRequest.ProjectID, "project_id"),
				helpers.OptionalNumericPointerParam(&timerUpdateRequest.TaskID, "task_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimerUpdate(ctx, engine, timerUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update timer")
			}

			return mcp.NewToolResultText("Timer updated successfully"), nil
		},
	}
}

// TimerPause pauses a timer in Teamwork.com.
func TimerPause(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerPause),
			mcp.WithDescription("Pause an existing timer in Teamwork.com. "+timerDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timer to pause."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerPauseRequest projects.TimerPauseRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timerPauseRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimerPause(ctx, engine, timerPauseRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to pause timer")
			}

			return mcp.NewToolResultText("Timer paused successfully"), nil
		},
	}
}

// TimerResume resumes a timer in Teamwork.com.
func TimerResume(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerResume),
			mcp.WithDescription("Resume an existing timer in Teamwork.com. "+timerDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timer to resume."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerResumeRequest projects.TimerResumeRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timerResumeRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimerResume(ctx, engine, timerResumeRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to resume timer")
			}

			return mcp.NewToolResultText("Timer resumed successfully"), nil
		},
	}
}

// TimerComplete completes a timer in Teamwork.com.
func TimerComplete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerComplete),
			mcp.WithDescription("Complete an existing timer in Teamwork.com. "+timerDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timer to complete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerCompleteRequest projects.TimerCompleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timerCompleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimerComplete(ctx, engine, timerCompleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to complete timer")
			}

			return mcp.NewToolResultText("Timer completed successfully"), nil
		},
	}
}

// TimerDelete deletes a timer in Teamwork.com.
func TimerDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerDelete),
			mcp.WithDescription("Delete an existing timer in Teamwork.com. "+timerDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timer to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerDeleteRequest projects.TimerDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timerDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TimerDelete(ctx, engine, timerDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete timer")
			}

			return mcp.NewToolResultText("Timer deleted successfully"), nil
		},
	}
}

// TimerGet retrieves a timer in Teamwork.com.
func TimerGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerGet),
			mcp.WithDescription("Get an existing timer in Teamwork.com. "+timerDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the timer to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerGetRequest projects.TimerGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&timerGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timer, err := projects.TimerGet(ctx, engine, timerGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get timer")
			}

			encoded, err := json.Marshal(timer)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/timers"),
			))), nil
		},
	}
}

// TimerList lists timers in Teamwork.com.
func TimerList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTimerList),
			mcp.WithDescription("List timers in Teamwork.com. "+timerDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("user_id",
				mcp.Description("The ID of the user to filter timers by. "+
					"Only timers associated with this user will be returned."),
			),
			mcp.WithNumber("task_id",
				mcp.Description("The ID of the task to filter timers by. "+
					"Only timers associated with this task will be returned."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project to filter timers by. "+
					"Only timers associated with this project will be returned."),
			),
			mcp.WithBoolean("running_timers_only",
				mcp.Description("If true, only running timers will be returned. "+
					"Defaults to false, which returns all timers."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerListRequest projects.TimerListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalNumericParam(&timerListRequest.Filters.UserID, "user_id"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.TaskID, "task_id"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.ProjectID, "project_id"),
				helpers.OptionalParam(&timerListRequest.Filters.RunningTimersOnly, "running_timers_only"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			timerList, err := projects.TimerList(ctx, engine, timerListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timers")
			}

			encoded, err := json.Marshal(timerList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/timers"),
			))), nil
		},
	}
}
