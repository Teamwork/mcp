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
	MethodTasklistBudgetList toolsets.Method = "twprojects-list_tasklist_budgets"
	MethodProjectBudgetList  toolsets.Method = "twprojects-list_project_budgets"
)

var (
	tasklistBudgetListOutputSchema *jsonschema.Schema
	projectBudgetListOutputSchema  *jsonschema.Schema
)

func init() {
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
			Description: "List project budgets (top-level project financial budgets). Filter by project_ids or status.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Project Budgets",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_ids": {
						Description: "Filter budgets by project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"status": {
						Description: "Filter budgets by status.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"upcoming", "active", "complete"}},
							{Type: "null"},
						},
					},
					"limit": {
						Description: "Maximum number of budgets to return.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page_size": helpers.PageSizeSchema(),
					"cursor": {
						Description: "Cursor for fetching the next page of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"verbose": helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(projectBudgetListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectBudgetListRequest := projects.NewProjectBudgetListRequest()

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
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
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if !verbose {
				projectBudgetListRequest.Filters.Fields.Budgets = []projects.ProjectBudgetField{
					projects.ProjectBudgetFieldID,
					projects.ProjectBudgetFieldType,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, projectBudgetListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list project budgets")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list project budgets"),
					"failed to list project budgets",
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

// TasklistBudgetList lists tasklist budgets for a project budget in Teamwork.com.
func TasklistBudgetList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTasklistBudgetList),
			Description: "List tasklist budgets nested under a project budget. Requires project_budget_id.",
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
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{"project_budget_id"},
			},
			OutputSchema: helpers.WithOptionalFields(tasklistBudgetListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectBudgetID int64

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectBudgetID, "project_budget_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			tasklistBudgetListRequest := projects.NewTasklistBudgetListRequest(projectBudgetID)
			verbose := true
			err = helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&tasklistBudgetListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tasklistBudgetListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if !verbose {
				tasklistBudgetListRequest.Filters.Fields.TasklistBudgets = []projects.TasklistBudgetField{
					projects.TasklistBudgetFieldID,
					projects.TasklistBudgetFieldType,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, tasklistBudgetListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasklist budgets")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list tasklist budgets"),
					"failed to list tasklist budgets",
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
