package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

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
	MethodTasklistBudgetList toolsets.Method = "twprojects-list_tasklist_budgets"
	MethodProjectBudgetList  toolsets.Method = "twprojects-list_project_budgets"
)

const tasklistBudgetDescription = "In the context of Teamwork.com, a tasklist budget is a budget allocation " +
	"attached to a specific task list within a project budget. It tracks capacity (in time or money) assigned to " +
	"and consumed by a task list, helping teams monitor spend and effort at a granular level within a broader " +
	"project budget."

const projectBudgetDescription = "In the context of Teamwork.com, a project budget defines the overall budget " +
	"allocation for a project, including capacity and usage tracking over time. It can be scoped by status and " +
	"project, enabling teams to monitor financial or effort limits, track consumption, and understand budget " +
	"performance across active, upcoming, and completed budget periods."

var (
	tasklistBudgetListOutputSchema *jsonschema.Schema
	projectBudgetListOutputSchema  *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTasklistBudgetList)
	toolsets.RegisterMethod(MethodProjectBudgetList)

	var err error

	// generate the output schemas only once
	tasklistBudgetListOutputSchema, err = jsonschema.For[projects.TasklistBudgetListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TasklistBudgetListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(tasklistBudgetListOutputSchema)

	projectBudgetListOutputSchema, err = jsonschema.For[projects.ProjectBudgetListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ProjectBudgetListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(projectBudgetListOutputSchema)
}

// ProjectBudgetList lists project budgets in Teamwork.com.
func ProjectBudgetList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectBudgetList),
			Description: "List project budgets in Teamwork.com. " + projectBudgetDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Project Budgets",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_ids": {
						Type:        "array",
						Description: "A list of project IDs to filter budgets by project.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"status": {
						Type:        "string",
						Description: "Filter budgets by status. Allowed values: upcoming, active, complete.",
						Enum:        []any{"upcoming", "active", "complete"},
					},
					"limit": {
						Type:        "integer",
						Description: "Maximum number of budgets to return.",
					},
					"page_size": {
						Type:        "integer",
						Description: "Number of budgets to return per page.",
					},
					"cursor": {
						Type:        "string",
						Description: "Cursor for fetching the next page of results.",
					},
				},
			},
			OutputSchema: projectBudgetListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectBudgetListRequest := projects.NewProjectBudgetListRequest()

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericListParam(&projectBudgetListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalParam(
					&projectBudgetListRequest.Filters.Status,
					"status",
					helpers.RestrictValues(
						projects.ProjectBudgetStatusUpcoming,
						projects.ProjectBudgetStatusActive,
						projects.ProjectBudgetStatusComplete,
					),
				),
				helpers.OptionalNumericParam(&projectBudgetListRequest.Filters.Limit, "limit"),
				helpers.OptionalNumericParam(&projectBudgetListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&projectBudgetListRequest.Filters.Cursor, "cursor"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			projectBudgetList, err := projects.ProjectBudgetList(ctx, engine, projectBudgetListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list project budgets")
			}

			encoded, err := json.Marshal(projectBudgetList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: projectBudgetList,
			}, nil
		},
	}
}

// TasklistBudgetList lists tasklist budgets for a project budget in Teamwork.com.
func TasklistBudgetList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistBudgetList),
			Description: "List tasklist budgets for a project budget in Teamwork.com. " + tasklistBudgetDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Tasklist Budgets",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_budget_id": {
						Type:        "integer",
						Description: "The ID of the project budget to list tasklist budgets for.",
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
				Required: []string{"project_budget_id"},
			},
			OutputSchema: tasklistBudgetListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectBudgetID int64

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectBudgetID, "project_budget_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tasklistBudgetListRequest := projects.NewTasklistBudgetListRequest(projectBudgetID)
			err = helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&tasklistBudgetListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tasklistBudgetListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			tasklistBudgetList, err := projects.TasklistBudgetList(ctx, engine, tasklistBudgetListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasklist budgets")
			}

			encoded, err := json.Marshal(tasklistBudgetList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: tasklistBudgetList,
			}, nil
		},
	}
}
