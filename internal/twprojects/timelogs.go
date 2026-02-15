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

var (
	timelogGetOutputSchema  *jsonschema.Schema
	timelogListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTimelogCreate)
	toolsets.RegisterMethod(MethodTimelogUpdate)
	toolsets.RegisterMethod(MethodTimelogDelete)
	toolsets.RegisterMethod(MethodTimelogGet)
	toolsets.RegisterMethod(MethodTimelogList)
	toolsets.RegisterMethod(MethodTimelogListByProject)
	toolsets.RegisterMethod(MethodTimelogListByTask)

	var err error

	// generate the output schemas only once
	timelogGetOutputSchema, err = jsonschema.For[projects.TimelogGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TimelogGetResponse: %v", err))
	}
	timelogListOutputSchema, err = jsonschema.For[projects.TimelogListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TimelogListResponse: %v", err))
	}
}

// TimelogCreate creates a timelog in Teamwork.com.
func TimelogCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Meta: mcp.Meta{
				"ui": map[string]any{
					"resourceUri": timelogCreateAppResourceURI,
				},
			},
			Name:        string(MethodTimelogCreate),
			Description: "Create a new timelog in Teamwork.com. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Timelog",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"description": {
						Type:        "string",
						Description: "A description of the timelog.",
					},
					"date": {
						Type:        "string",
						Format:      "date",
						Description: "The date of the timelog in the format YYYY-MM-DD.",
					},
					"time": {
						Type:        "string",
						Pattern:     `^(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d$`,
						Description: "The time of the timelog in the format HH:MM:SS.",
					},
					"is_utc": {
						Type:        "boolean",
						Description: "If true, the time is in UTC. Defaults to false.",
					},
					"hours": {
						Type:        "integer",
						Description: "The number of hours spent on the timelog. Must be a positive integer.",
					},
					"minutes": {
						Type: "integer",
						Description: "The number of minutes spent on the timelog. Must be a positive integer less than 60, " +
							"otherwise the hours attribute should be incremented.",
					},
					"billable": {
						Type:        "boolean",
						Description: "If true, the timelog is billable. Defaults to false.",
					},
					"project_id": {
						Type: "integer",
						Description: "The ID of the project to associate the timelog with. Either project_id or task_id must be " +
							"provided, but not both.",
					},
					"task_id": {
						Type: "integer",
						Description: "The ID of the task to associate the timelog with. Either project_id or task_id must be " +
							"provided, but not both.",
					},
					"user_id": {
						Type: "integer",
						Description: "The ID of the user to associate the timelog with. Defaults to the authenticated user if " +
							"not provided.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the timelog.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"date", "time", "hours", "minutes"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogCreateRequest projects.TimelogCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timelogResponse, err := projects.TimelogCreate(ctx, engine, timelogCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create timelog")
			}
			return helpers.NewToolResultText("Timelog created successfully with ID %d", timelogResponse.Timelog.ID), nil
		},
	}
}

// TimelogUpdate updates a timelog in Teamwork.com.
func TimelogUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimelogUpdate),
			Description: "Update an existing timelog in Teamwork.com. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Timelog",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timelog to update.",
					},
					"description": {
						Type:        "string",
						Description: "A description of the timelog.",
					},
					"date": {
						Type:        "string",
						Format:      "date",
						Description: "The date of the timelog in the format YYYY-MM-DD.",
					},
					"time": {
						Type:        "string",
						Pattern:     `^(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d$`,
						Description: "The time of the timelog in the format HH:MM:SS.",
					},
					"is_utc": {
						Type:        "boolean",
						Description: "If true, the time is in UTC. Defaults to false.",
					},
					"hours": {
						Type:        "integer",
						Description: "The number of hours spent on the timelog. Must be a positive integer.",
					},
					"minutes": {
						Type: "integer",
						Description: "The number of minutes spent on the timelog. Must be a positive integer less than 60, " +
							"otherwise the hours attribute should be incremented.",
					},
					"billable": {
						Type:        "boolean",
						Description: "If true, the timelog is billable. Defaults to false.",
					},
					"project_id": {
						Type: "integer",
						Description: "The ID of the project to associate the timelog with. Either project_id or task_id must be " +
							"provided, but not both.",
					},
					"task_id": {
						Type: "integer",
						Description: "The ID of the task to associate the timelog with. Either project_id or task_id must be " +
							"provided, but not both.",
					},
					"user_id": {
						Type: "integer",
						Description: "The ID of the user to associate the timelog with. Defaults to the authenticated user if " +
							"not provided.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the timelog.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogUpdateRequest projects.TimelogUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimelogUpdate(ctx, engine, timelogUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update timelog")
			}
			return helpers.NewToolResultText("Timelog updated successfully"), nil
		},
	}
}

// TimelogDelete deletes a timelog in Teamwork.com.
func TimelogDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimelogDelete),
			Description: "Delete an existing timelog in Teamwork.com. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Timelog",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timelog to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogDeleteRequest projects.TimelogDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timelogDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TimelogDelete(ctx, engine, timelogDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete timelog")
			}
			return helpers.NewToolResultText("Timelog deleted successfully"), nil
		},
	}
}

// TimelogGet retrieves a timelog in Teamwork.com.
func TimelogGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimelogGet),
			Description: "Get an existing timelog in Teamwork.com. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Timelog",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the timelog to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: timelogGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogGetRequest projects.TimelogGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&timelogGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timelog, err := projects.TimelogGet(ctx, engine, timelogGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get timelog")
			}
			return helpers.NewToolResultJSON(timelog)
		},
	}
}

// TimelogList lists timelogs in Teamwork.com.
func TimelogList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimelogList),
			Description: "List timelogs in Teamwork.com. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Timelogs",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter timelogs by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match timelogs that have all the specified tags. If false, the " +
							"search will match timelogs that have any of the specified tags. Defaults to false.",
					},
					"start_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "Start date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "End date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"assigned_user_ids": {
						Type:        "array",
						Description: "A list of user IDs to filter timelogs by assigned users",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"assigned_company_ids": {
						Type:        "array",
						Description: "A list of company IDs to filter timelogs by assigned companies",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"assigned_team_ids": {
						Type:        "array",
						Description: "A list of team IDs to filter timelogs by assigned teams",
						Items: &jsonschema.Schema{
							Type: "integer",
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
			OutputSchema: timelogListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogListRequest projects.TimelogListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timelogList, err := projects.TimelogList(ctx, engine, timelogListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timelogs")
			}
			return helpers.NewToolResultJSON(timelogList)
		},
	}
}

// TimelogListByProject lists timelogs in Teamwork.com by project.
func TimelogListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimelogListByProject),
			Description: "List timelogs in Teamwork.com by project. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Timelogs By Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project from which to retrieve timelogs.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter timelogs by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match timelogs that have all the specified tags. If false, the " +
							"search will match timelogs that have any of the specified tags. Defaults to false.",
					},
					"start_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "Start date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "End date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"assigned_user_ids": {
						Type:        "array",
						Description: "A list of user IDs to filter timelogs by assigned users",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"assigned_company_ids": {
						Type:        "array",
						Description: "A list of company IDs to filter timelogs by assigned companies",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"assigned_team_ids": {
						Type:        "array",
						Description: "A list of team IDs to filter timelogs by assigned teams",
						Items: &jsonschema.Schema{
							Type: "integer",
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
				Required: []string{"project_id"},
			},
			OutputSchema: timelogListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogListRequest projects.TimelogListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timelogList, err := projects.TimelogList(ctx, engine, timelogListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timelogs")
			}
			return helpers.NewToolResultJSON(timelogList)
		},
	}
}

// TimelogListByTask lists timelogs in Teamwork.com by task.
func TimelogListByTask(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTimelogListByTask),
			Description: "List timelogs in Teamwork.com by task. " + timelogDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Timelogs By Task",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task_id": {
						Type:        "integer",
						Description: "The ID of the task from which to retrieve timelogs.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter timelogs by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match timelogs that have all the specified tags. If false, the " +
							"search will match timelogs that have any of the specified tags. Defaults to false.",
					},
					"start_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "Start date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"end_date": {
						Type:        "string",
						Format:      "date-time",
						Description: "End date to filter timelogs. The date format follows RFC3339 - YYYY-MM-DDTHH:MM:SSZ.",
					},
					"assigned_user_ids": {
						Type:        "array",
						Description: "A list of user IDs to filter timelogs by assigned users",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"assigned_company_ids": {
						Type:        "array",
						Description: "A list of company IDs to filter timelogs by assigned companies",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"assigned_team_ids": {
						Type:        "array",
						Description: "A list of team IDs to filter timelogs by assigned teams",
						Items: &jsonschema.Schema{
							Type: "integer",
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
				Required: []string{"task_id"},
			},
			OutputSchema: timelogListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var timelogListRequest projects.TimelogListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			timelogList, err := projects.TimelogList(ctx, engine, timelogListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list timelogs")
			}
			return helpers.NewToolResultJSON(timelogList)
		},
	}
}
