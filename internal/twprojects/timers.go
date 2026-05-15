package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

var (
	timerGetOutputSchema  *jsonschema.Schema
	timerListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	timerGetOutputSchema, err = jsonschema.For[projects.TimerGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TimerGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(timerGetOutputSchema)
	timerListOutputSchema, err = jsonschema.For[projects.TimerListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TimerListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(timerListOutputSchema)
}

// TimerCreate creates a timer in Teamwork.com.
func TimerCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerCreate),
			Description: "Create and start a timer.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Timer",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"description": {
						Description: "A description of the timer.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"billable": {
						Description: "If true, the timer is billable. Defaults to false.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"running": {
						Description: "If true, the timer will start running immediately.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"seconds": {
						Description: "The number of seconds to set the timer for.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"stop_running_timers": {
						Description: "If true, any other running timers will be stopped when this timer is created.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to associate the timer with.",
					},
					"task_id": {
						Description: "The ID of the task to associate the timer with.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"project_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerCreateRequest projects.TimerCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			timerResponse, err := projects.TimerCreate(ctx, engine, timerCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create timer")
			}
			return helpers.NewToolResultText("Timer created successfully with ID %d", timerResponse.Timer.ID), nil
		},
	}
}

// TimerUpdate updates a timer in Teamwork.com.
func TimerUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerUpdate),
			Description: "Update timer.",
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
						Description: "A description of the timer.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"billable": {
						Description: "If true, the timer is billable.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"running": {
						Description: "If true, the timer will start running immediately.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to associate the timer with.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"task_id": {
						Description: "The ID of the task to associate the timer with.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerUpdateRequest projects.TimerUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TimerUpdate(ctx, engine, timerUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update timer")
			}
			return helpers.NewToolResultText("Timer updated successfully"), nil
		},
	}
}

// TimerPause pauses a timer in Teamwork.com.
func TimerPause(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerPause),
			Description: "Pause a running timer; can be resumed later. Use complete_timer to stop permanently.",
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerPauseRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TimerPause(ctx, engine, timerPauseRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to pause timer")
			}
			return helpers.NewToolResultText("Timer paused successfully"), nil
		},
	}
}

// TimerResume resumes a timer in Teamwork.com.
func TimerResume(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerResume),
			Description: "Resume a paused timer back to running.",
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerResumeRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TimerResume(ctx, engine, timerResumeRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to resume timer")
			}
			return helpers.NewToolResultText("Timer resumed successfully"), nil
		},
	}
}

// TimerComplete completes a timer in Teamwork.com.
func TimerComplete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerComplete),
			Description: "Stop a timer permanently and convert it to a timelog. Use pause_timer to pause without converting.",
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerCompleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TimerComplete(ctx, engine, timerCompleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to complete timer")
			}
			return helpers.NewToolResultText("Timer completed successfully"), nil
		},
	}
}

// TimerDelete deletes a timer in Teamwork.com.
func TimerDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerDelete),
			Description: "Delete timer.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Timer",
				DestructiveHint: new(true),
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.TimerDelete(ctx, engine, timerDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete timer")
			}
			return helpers.NewToolResultText("Timer deleted successfully"), nil
		},
	}
}

// TimerGet retrieves a timer in Teamwork.com.
func TimerGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerGet),
			Description: "Get timer.",
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timerGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
				StructuredContent: helpers.StructuredWebLinker(ctx, timer,
					helpers.WebLinkerWithIDPathBuilder("/app/timers"),
				),
			}, nil
		},
	}
}

// TimerList lists timers in Teamwork.com.
func TimerList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimerList),
			Description: "List timers.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Timers",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"user_id": {
						Description: "The ID of the user to filter timers by. " +
							"Only timers associated with this user will be returned.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"task_id": {
						Description: "The ID of the task to filter timers by. " +
							"Only timers associated with this task will be returned.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to filter timers by. " +
							"Only timers associated with this project will be returned.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"running_timers_only": {
						Description: "If true, only running timers will be returned. " +
							"Defaults to false, which returns all timers.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(timerListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timerListRequest projects.TimerListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&timerListRequest.Filters.UserID, "user_id"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.TaskID, "task_id"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.ProjectID, "project_id"),
				helpers.OptionalParam(&timerListRequest.Filters.RunningTimersOnly, "running_timers_only"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&timerListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				timerListRequest.Filters.Fields.Timers = []projects.TimerField{
					projects.TimerFieldID,
					projects.TimerFieldDescription,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, timerListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timers")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list timers"), "failed to list timers")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/me/timers"))
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
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
