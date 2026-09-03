package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCalendarEventList toolsets.Method = "twprojects-list_calendar_events"
)

var calendarEventListOutputSchema *jsonschema.Schema

// calendarEventOrdering is the order-by vocabulary of the calendar events list endpoint.
var calendarEventOrdering = newOrdering("calendar events",
	projects.CalendarEventOrderByStartTime,
	projects.CalendarEventOrderByUpdated,
	projects.CalendarEventOrderByID,
)

func init() {
	var err error

	// generate the output schema only once
	calendarEventListOutputSchema, err = jsonschema.For[projects.CalendarEventListResponse](
		helpers.WithDateTypeSchema(&jsonschema.ForOptions{}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CalendarEventListResponse: %v", err))
	}
}

// CalendarEventList lists calendar events in Teamwork.com.
func CalendarEventList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCalendarEventList),
			Description: "List events from a calendar, including time-blocking events that link a calendar slot to " +
				"a Teamwork project, task or timelog. Omit calendar_id to read the calling user's own calendar. " +
				"Use twprojects-list_calendars to name a different one; the calendar of type 'blocked_time' holds " +
				"the account's time-blocking events.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Calendar Events",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"calendar_id": {
						Description: "The ID of the calendar to list events from. Omit it for the calling " +
							"user's own calendar: the connected Google or Outlook calendar when there is one, " +
							"otherwise the calendar of type 'blocked_time'. Every event reports the calendar it " +
							"came from.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"started_after_date": {
						Description: "Only include events starting on or after this day, which is itself " +
							"included (format: YYYY-MM-DD).",
						Examples: []any{"2023-01-01"},
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
					"ended_before_date": {
						Description: "Only include events ending before this day starts, so the day named " +
							"here is itself excluded — pass the day after the last one you want " +
							"(format: YYYY-MM-DD). Note the asymmetry with started_after_date.",
						Examples: []any{"2023-12-31"},
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
					"order_by":   calendarEventOrdering.orderBySchema(),
					"order_mode": orderModeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"fields":     helpers.FieldsSchema[projects.CalendarEvent]("calendar event"),
				},
				Required: []string{},
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
				helpers.OptionalNumericParam(&calendarEventListRequest.Path.CalendarID, "calendar_id"),
				helpers.OptionalDateParam(&calendarEventListRequest.Filters.StartedAfterDate, "started_after_date"),
				helpers.OptionalDateParam(&calendarEventListRequest.Filters.EndedBeforeDate, "ended_before_date"),
				calendarEventOrdering.param(&calendarEventListRequest.Filters.OrderBy, &calendarEventListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&calendarEventListRequest.Filters.Limit, "limit"),
				helpers.OptionalParam(&calendarEventListRequest.Filters.Cursor, "cursor"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalFieldsParam[projects.CalendarEvent](&calendarEventListRequest.Filters.Fields.Events, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if calendarEventListRequest.Path.CalendarID == 0 {
				calendarID, result, err := defaultCalendarID(ctx, engine)
				if result != nil || err != nil {
					return result, err
				}
				calendarEventListRequest.Path.CalendarID = calendarID
			}
			switch {
			case len(calendarEventListRequest.Filters.Fields.Events) > 0:
				// An explicit field selection overrides both defaults below: the
				// caller has already said what it wants, and the sideloads would
				// smuggle back the bulk the selection exists to avoid.

			case verbose:
				// Sideload the entities referenced by attendees and timeblocks so
				// time-blocking events can be related to their project, task and
				// timelog without extra tool calls.
				calendarEventListRequest.Filters.Include = []projects.CalendarEventListRequestSideload{
					projects.CalendarEventListRequestSideloadUsers,
					projects.CalendarEventListRequestSideloadProjects,
					projects.CalendarEventListRequestSideloadTasks,
					projects.CalendarEventListRequestSideloadTimelogs,
				}
				// Restrict sideloads to the fields that resolve ids; otherwise every
				// attendee drags a full user record into the response.
				calendarEventListRequest.Filters.Fields.Users = []projects.UserField{
					projects.UserFieldID, projects.UserFieldFirstName, projects.UserFieldLastName,
				}
				calendarEventListRequest.Filters.Fields.Projects = []projects.ProjectField{
					projects.ProjectFieldID, projects.ProjectFieldName,
				}
				calendarEventListRequest.Filters.Fields.Tasks = []projects.TaskField{
					projects.TaskFieldID, projects.TaskFieldName,
				}
				calendarEventListRequest.Filters.Fields.Timelogs = []projects.TimelogField{
					projects.TimelogFieldID, projects.TimelogFieldDescription, projects.TimelogFieldMinutes,
				}

			default:
				calendarEventListRequest.Filters.Fields.Events = []projects.CalendarEventField{
					projects.CalendarEventFieldID,
					projects.CalendarEventFieldSummary,
					projects.CalendarEventFieldStart,
					projects.CalendarEventFieldEnd,
					// keep the calendar, so a caller that named none can still see
					// which one answered.
					projects.CalendarEventFieldCalendar,
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

// defaultCalendarID resolves the calendar to list events from when the caller
// named none, saving the twprojects-list_calendars round trip.
//
// The calendars endpoint is scoped to the authenticated user, and a site allows
// one calendar integration at a time, so a connected Google or Outlook calendar
// is unambiguous when it exists. Time blocking is the fallback, since that is
// where an account's own events live without an integration.
//
// A non-nil result is the caller's answer: either the lookup failed, or nothing
// could be picked and the candidates are reported so the model can name one.
func defaultCalendarID(ctx context.Context, engine *twapi.Engine) (int64, *mcp.CallToolResult, error) {
	calendarListRequest := projects.NewCalendarListRequest()
	calendarListRequest.Filters.PageSize = 250 // one page holds a user's calendars
	calendarListRequest.Filters.Fields.Calendars = []projects.CalendarField{
		projects.CalendarFieldID,
		projects.CalendarFieldName,
		projects.CalendarFieldType,
	}
	calendars, err := projects.CalendarList(ctx, engine, calendarListRequest)
	if err != nil {
		result, err := helpers.HandleAPIError(err, "failed to list calendars")
		return 0, result, err
	}

	var blockedTime int64
	for _, calendar := range calendars.Calendars {
		switch calendar.Type {
		case projects.CalendarTypeGoogle, projects.CalendarTypeOutlook:
			return calendar.ID, nil, nil
		case projects.CalendarTypeBlockedTime:
			blockedTime = calendar.ID
		}
	}
	switch {
	case blockedTime > 0:
		return blockedTime, nil, nil
	case len(calendars.Calendars) == 1:
		return calendars.Calendars[0].ID, nil, nil
	}

	candidates, err := json.Marshal(calendars.Calendars)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to encode calendars: %w", err)
	}
	return 0, helpers.NewToolResultTextError("no calendar to default to, set calendar_id to one of: %s",
		string(candidates)), nil
}
