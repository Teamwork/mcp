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
	MethodActivityList          toolsets.Method = "twprojects-list_activities"
	MethodActivityListByProject toolsets.Method = "twprojects-list_activities_by_project"
)

const activityDescription = "Activity is a record of actions and updates that occur across your projects, tasks, and " +
	"communications, giving you a clear view of what’s happening within your workspace. Activities capture changes " +
	"such as task completions, activities added, files uploaded, or milestones updated, and present them in a " +
	"chronological feed so teams can stay aligned without needing to check each individual project or task. This " +
	"stream of information helps improve transparency, ensures accountability, and keeps everyone aware of progress " +
	"and decisions as they happen."

var (
	activityListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodActivityList)
	toolsets.RegisterMethod(MethodActivityListByProject)

	var err error

	// generate the output schemas only once
	activityListOutputSchema, err = jsonschema.For[projects.ActivityListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ActivityListResponse: %v", err))
	}
}

// ActivityList lists activities in Teamwork.com.
func ActivityList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodActivityList),
			Description: "List activities in Teamwork.com. " + activityDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Activities",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"start_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "Start date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "End date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"log_item_types": {
						Type:        "array",
						Description: "Filter activities by item types.",
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
			OutputSchema: activityListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var activityListRequest projects.ActivityListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalTimeParam(&activityListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalListParam(&activityListRequest.Filters.LogItemTypes, "log_item_types"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			activityList, err := projects.ActivityList(ctx, engine, activityListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list activities")
			}

			encoded, err := json.Marshal(activityList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
			}, nil
		},
	}
}

// ActivityListByProject lists activities by project in Teamwork.com.
func ActivityListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodActivityListByProject),
			Description: "List activities in Teamwork.com by project. " + activityDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Activities by Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to retrieve activities from.",
					},
					"start_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "Start date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "End date to filter activities. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"log_item_types": {
						Type:        "array",
						Description: "Filter activities by item types.",
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
			OutputSchema: activityListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var activityListRequest projects.ActivityListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&activityListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.StartDate, "start_date"),
				helpers.OptionalTimeParam(&activityListRequest.Filters.EndDate, "end_date"),
				helpers.OptionalListParam(&activityListRequest.Filters.LogItemTypes, "log_item_types"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&activityListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			activityList, err := projects.ActivityList(ctx, engine, activityListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list activities")
			}

			encoded, err := json.Marshal(activityList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
			}, nil
		},
	}
}
