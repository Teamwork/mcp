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
	MethodCalendarList toolsets.Method = "twprojects-list_calendars"
)

var calendarListOutputSchema *jsonschema.Schema

// calendarOrdering is the order-by vocabulary of the calendars list endpoint.
var calendarOrdering = newOrdering("calendars",
	projects.CalendarOrderByName,
	projects.CalendarOrderByID,
)

func init() {
	var err error

	// generate the output schema only once
	calendarListOutputSchema, err = jsonschema.For[projects.CalendarListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CalendarListResponse: %v", err))
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
				Title:           "List Calendars",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"order_by":   calendarOrdering.orderBySchema(),
					"order_mode": orderModeSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"count_only": helpers.CountOnlySchema("calendars"),
					"fields":     helpers.FieldsSchema[projects.Calendar]("calendar"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(calendarListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var calendarListRequest projects.CalendarListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				calendarOrdering.param(&calendarListRequest.Filters.OrderBy, &calendarListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&calendarListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&calendarListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.Calendar](&calendarListRequest.Filters.Fields.Calendars, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if !verbose && len(calendarListRequest.Filters.Fields.Calendars) == 0 {
				calendarListRequest.Filters.Fields.Calendars = []projects.CalendarField{
					projects.CalendarFieldID,
					projects.CalendarFieldName,
					projects.CalendarFieldType,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, calendarListRequest, "failed to count calendars")
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
