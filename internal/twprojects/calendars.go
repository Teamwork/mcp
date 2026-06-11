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
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCalendarCreate    toolsets.Method = "twprojects-create_calendar"
	MethodCalendarList      toolsets.Method = "twprojects-list_calendars"
	MethodCalendarEventList toolsets.Method = "twprojects-list_calendar_events"
)

var (
	calendarListOutputSchema      *jsonschema.Schema
	calendarEventListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	calendarListOutputSchema, err = jsonschema.For[projects.CalendarListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CalendarListResponse: %v", err))
	}
	calendarEventListOutputSchema, err = jsonschema.For[projects.CalendarEventListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CalendarEventListResponse: %v", err))
	}
}

// CalendarCreate creates a calendar in Teamwork.com.
func CalendarCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCalendarCreate),
			Description: "Create calendar. Calendars hold events such as meetings, out-of-office periods and " +
				"time-blocking entries. To enable time blocking, create a calendar of type 'blocked_time' named " +
				"'blocked_time'; there can only be one blocked time calendar per account.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Calendar",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type: "string",
						Description: "The name of the calendar. When the type is 'blocked_time' the name must be " +
							"'blocked_time'.",
					},
					"type": {
						Description: "The type of calendar. Defaults to a standard event calendar.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"event", "blocked_time", "holiday"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var calendarCreateRequest projects.CalendarCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&calendarCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&calendarCreateRequest.Type, "type",
					helpers.RestrictValues(
						projects.CalendarTypeEvent,
						projects.CalendarTypeBlockedTime,
						projects.CalendarTypeHoliday,
					),
				),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			calendar, err := projects.CalendarCreate(ctx, engine, calendarCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create calendar")
			}
			return helpers.NewToolResultText("Calendar created successfully with ID %d", calendar.Calendar.ID), nil
		},
	}
}

// CalendarList lists calendars in Teamwork.com.
func CalendarList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCalendarList),
			Description: "List calendars. Calendars hold events such as meetings, out-of-office periods and " +
				"time-blocking entries; the calendar of type 'blocked_time' holds the account's time-blocking events.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Calendars",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(calendarListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var calendarListRequest projects.CalendarListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&calendarListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&calendarListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if !verbose {
				calendarListRequest.Filters.Fields.Calendars = []projects.CalendarField{
					projects.CalendarFieldID,
					projects.CalendarFieldName,
					projects.CalendarFieldType,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, calendarListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list calendars")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list calendars"),
					"failed to list calendars",
				)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(body)},
				},
			}
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}

// CalendarEventList lists calendar events in Teamwork.com.
func CalendarEventList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCalendarEventList),
			Description: "List events from a calendar, including time-blocking events that link a calendar slot to " +
				"a Teamwork project, task or timelog. Use twprojects-list_calendars to find the calendar ID; the " +
				"calendar of type 'blocked_time' holds the account's time-blocking events.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Calendar Events",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"calendar_id": {
						Type:        "integer",
						Description: "The ID of the calendar to list events from.",
					},
					"started_after_date": {
						Description: "Filter events that start after this date (format: YYYY-MM-DD).",
						Examples:    []any{"2023-01-01"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"ended_before_date": {
						Description: "Filter events that end before this date (format: YYYY-MM-DD).",
						Examples:    []any{"2023-12-31"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"limit": {
						Description: "Maximum number of events to return.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"cursor": {
						Description: "Cursor for fetching the next page of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"verbose": helpers.VerboseSchema(),
				},
				Required: []string{"calendar_id"},
			},
			OutputSchema: helpers.WithOptionalFields(calendarEventListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var calendarEventListRequest projects.CalendarEventListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&calendarEventListRequest.Path.CalendarID, "calendar_id"),
				helpers.OptionalDateParam(&calendarEventListRequest.Filters.StartedAfterDate, "started_after_date"),
				helpers.OptionalDateParam(&calendarEventListRequest.Filters.EndedBeforeDate, "ended_before_date"),
				helpers.OptionalNumericParam(&calendarEventListRequest.Filters.Limit, "limit"),
				helpers.OptionalParam(&calendarEventListRequest.Filters.Cursor, "cursor"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if verbose {
				// Sideload the entities referenced by attendees and timeblocks so
				// time-blocking events can be related to their project, task and
				// timelog without extra tool calls.
				calendarEventListRequest.Filters.Include = []projects.CalendarEventListRequestSideload{
					projects.CalendarEventListRequestSideloadUsers,
					projects.CalendarEventListRequestSideloadProjects,
					projects.CalendarEventListRequestSideloadTasks,
					projects.CalendarEventListRequestSideloadTimelogs,
				}
			} else {
				calendarEventListRequest.Filters.Fields.Events = []projects.CalendarEventField{
					projects.CalendarEventFieldID,
					projects.CalendarEventFieldSummary,
					projects.CalendarEventFieldStart,
					projects.CalendarEventFieldEnd,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, calendarEventListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list calendar events")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list calendar events"),
					"failed to list calendar events",
				)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(body)},
				},
			}
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
