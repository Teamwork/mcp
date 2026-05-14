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
	MethodActivityList toolsets.Method = "twprojects-list_activities"
)

var (
	activityListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	activityListOutputSchema, err = jsonschema.For[projects.ActivityListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ActivityListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(activityListOutputSchema)
}

// ActivityList lists activities in Teamwork.com.
func ActivityList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodActivityList),
			Description: "List recent activity events. Scope by project_id or omit for site-wide.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Activities",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Description: "The ID of the project to retrieve activities from. Omit to list activities across all projects.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"start_date": {
						Description: "Start date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"end_date": {
						Description: "End date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "null"},
						},
					},
					"log_item_types": {
						Description: "Filter activities by item types.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "string",
									Enum: []any{
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
								},
							},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(activityListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var activityListRequest projects.ActivityListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&activityListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalListParam(&activityListRequest.Filters.LogItemTypes, "log_item_types"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				activityListRequest.Filters.Fields.Activities = []projects.ActivityField{
					projects.ActivityFieldID,
					projects.ActivityFieldDescription,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, activityListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list activities")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list activities"), "failed to list activities")
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
