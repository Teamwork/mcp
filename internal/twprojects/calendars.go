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
	MethodCalendarDelete toolsets.Method = "twprojects-delete_calendar"
	MethodCalendarList   toolsets.Method = "twprojects-list_calendars"
)

var calendarListOutputSchema *jsonschema.Schema

func init() {
	var err error

	// generate the output schema only once
	calendarListOutputSchema, err = jsonschema.For[projects.CalendarListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CalendarListResponse: %v", err))
	}
}

// CalendarDelete deletes a calendar in Teamwork.com.
func CalendarDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCalendarDelete),
			Description: "Delete calendar.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Calendar",
				DestructiveHint: new(true),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the calendar to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var calendarDeleteRequest projects.CalendarDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&calendarDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CalendarDelete(ctx, engine, calendarDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete calendar")
			}
			return helpers.NewToolResultText("Calendar deleted successfully"), nil
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
