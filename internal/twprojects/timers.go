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

var (
	timerGetOutputSchema  *jsonschema.Schema
	timerListOutputSchema *jsonschema.Schema
)

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

	var err error

	// generate the output schemas only once
	timerGetOutputSchema, err = jsonschema.For[projects.TimerGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TimerGetResponse: %v", err))
	}
	timerListOutputSchema, err = jsonschema.For[projects.TimerListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TimerListResponse: %v", err))
	}
}

// TimerCreate creates a timer in Teamwork.com.
func TimerCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerCreate),
			Description: "Create a new timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"description": {
						Type:        "string",
						Description: "A description of the timer.",
					},
					"billable": {
						Type:        "boolean",
						Description: "If true, the timer is billable. Defaults to false.",
					},
					"running": {
						Type:        "boolean",
						Description: "If true, the timer will start running immediately.",
					},
					"seconds": {
						Type:        "integer",
						Description: "The number of seconds to set the timer for.",
					},
					"stop_running_timers": {
						Type:        "boolean",
						Description: "If true, any other running timers will be stopped when this timer is created.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to associate the timer with.",
					},
					"task_id": {
						Type:        "integer",
						Description: "The ID of the task to associate the timer with.",
					},
				},
				Required: []string{"project_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerCreateRequest projects.TimerCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalPointerParam(&timerCreateRequest.Description, "description"),
				helpers.OptionalPointerParam(&timerCreateRequest.Billable, "billable"),
				helpers.OptionalPointerParam(&timerCreateRequest.Running, "running"),
				helpers.OptionalNumericPointerParam(&timerCreateRequest.Seconds, "seconds"),
				helpers.OptionalPointerParam(&timerCreateRequest.StopRunningTimers, "stop_running_timers"),
				helpers.RequiredNumericParam(&timerCreateRequest.ProjectID, "project_id"),
				helpers.OptionalNumericPointerParam(&timerCreateRequest.TaskID, "task_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timerResponse, err := projects.TimerCreate(ctx, engine, timerCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create timer")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: fmt.Sprintf("Timer created successfully with ID %d", timerResponse.Timer.ID),
					},
				},
			}, nil
		},
	}
}

// TimerUpdate updates a timer in Teamwork.com.
func TimerUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerUpdate),
			Description: "Update an existing timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timer to update.",
					},
					"description": {
						Type:        "string",
						Description: "A description of the timer.",
					},
					"billable": {
						Type:        "boolean",
						Description: "If true, the timer is billable.",
					},
					"running": {
						Type:        "boolean",
						Description: "If true, the timer will start running immediately.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to associate the timer with.",
					},
					"task_id": {
						Type:        "integer",
						Description: "The ID of the task to associate the timer with.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerUpdateRequest projects.TimerUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&timerUpdateRequest.Description, "description"),
				helpers.OptionalPointerParam(&timerUpdateRequest.Billable, "billable"),
				helpers.OptionalPointerParam(&timerUpdateRequest.Running, "running"),
				helpers.OptionalNumericPointerParam(&timerUpdateRequest.ProjectID, "project_id"),
				helpers.OptionalNumericPointerParam(&timerUpdateRequest.TaskID, "task_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimerUpdate(ctx, engine, timerUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update timer")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Timer updated successfully",
					},
				},
			}, nil
		},
	}
}

// TimerPause pauses a timer in Teamwork.com.
func TimerPause(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerPause),
			Description: "Pause an existing timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Pause Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timer to pause.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerPauseRequest projects.TimerPauseRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerPauseRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimerPause(ctx, engine, timerPauseRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to pause timer")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Timer paused successfully",
					},
				},
			}, nil
		},
	}
}

// TimerResume resumes a timer in Teamwork.com.
func TimerResume(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerResume),
			Description: "Resume an existing timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Resume Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timer to resume.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerResumeRequest projects.TimerResumeRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerResumeRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimerResume(ctx, engine, timerResumeRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to resume timer")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Timer resumed successfully",
					},
				},
			}, nil
		},
	}
}

// TimerComplete completes a timer in Teamwork.com.
func TimerComplete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerComplete),
			Description: "Complete an existing timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Complete Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timer to complete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerCompleteRequest projects.TimerCompleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerCompleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimerComplete(ctx, engine, timerCompleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to complete timer")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Timer completed successfully",
					},
				},
			}, nil
		},
	}
}

// TimerDelete deletes a timer in Teamwork.com.
func TimerDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerDelete),
			Description: "Delete an existing timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timer to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerDeleteRequest projects.TimerDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimerDelete(ctx, engine, timerDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete timer")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Timer deleted successfully",
					},
				},
			}, nil
		},
	}
}

// TimerGet retrieves a timer in Teamwork.com.
func TimerGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerGet),
			Description: "Get an existing timer in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Timer",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timer to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: timerGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerGetRequest projects.TimerGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timer, err := projects.TimerGet(ctx, engine, timerGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get timer")
			}

			encoded, err := json.Marshal(timer)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/timers"),
						)),
					},
				},
			}, nil
		},
	}
}

// TimerList lists timers in Teamwork.com.
func TimerList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerList),
			Description: "List timers in Teamwork.com. " + timerDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Timers",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"user_id": {
						Type: "integer",
						Description: "The ID of the user to filter timers by. " +
							"Only timers associated with this user will be returned.",
					},
					"task_id": {
						Type: "integer",
						Description: "The ID of the task to filter timers by. " +
							"Only timers associated with this task will be returned.",
					},
					"project_id": {
						Type: "integer",
						Description: "The ID of the project to filter timers by. " +
							"Only timers associated with this project will be returned.",
					},
					"running_timers_only": {
						Type: "boolean",
						Description: "If true, only running timers will be returned. " +
							"Defaults to false, which returns all timers.",
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
			},
			OutputSchema: timerListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerListRequest projects.TimerListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&timerListRequest.Filters.UserID, "user_id"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.TaskID, "task_id"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.ProjectID, "project_id"),
				helpers.OptionalParam(&timerListRequest.Filters.RunningTimersOnly, "running_timers_only"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timerList, err := projects.TimerList(ctx, engine, timerListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timers")
			}

			encoded, err := json.Marshal(timerList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/timers"),
						)),
					},
				},
			}, nil
		},
	}
}
