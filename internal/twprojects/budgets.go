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

// projectBudgetSparseFields is the attribute set returned by
// twprojects-list_project_budgets when verbose=false.
//
// A budget is only meaningful in relation to its project, so unlike most
// list_* tools the minimal set cannot be id plus a label: without projectId
// and status the response cannot be joined to anything and the caller is
// forced back to verbose. It stops there — the audit trail, repeat
// configuration and rate settings are what make the verbose row roughly 1 KB.
var projectBudgetSparseFields = []projects.ProjectBudgetField{
	projects.ProjectBudgetFieldID,
	projects.ProjectBudgetFieldProjectID,
	projects.ProjectBudgetFieldType,
	projects.ProjectBudgetFieldStatus,
	projects.ProjectBudgetFieldCapacity,
	projects.ProjectBudgetFieldCapacityUsed,
	projects.ProjectBudgetFieldStartDateTime,
	projects.ProjectBudgetFieldEndDateTime,
}

// projectBudgetSparseKeys mirrors projectBudgetSparseFields as raw JSON keys,
// for projecting the response body ourselves. The budgets endpoint does not
// implement sparse fieldsets, so fields[budgets] alone leaves verbose=false a
// no-op; see helpers.ProjectCollectionFields.
var projectBudgetSparseKeys = func() []string {
	keys := make([]string, 0, len(projectBudgetSparseFields))
	for _, field := range projectBudgetSparseFields {
		keys = append(keys, string(field))
	}
	return keys
}()

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
			Name: string(MethodProjectBudgetList),
			Description: "List project budgets (top-level project financial budgets). Filter by project_ids or status. " +
				"Returns only projects that HAVE a budget — there is no filter for projects without one, so finding " +
				"those means listing budgets and diffing against twprojects-list_projects. When checking a known set " +
				"of projects, pass project_ids to filter server-side instead of scanning every budget. " +
				"Page with page (1-based); the response reports meta.page.pageOffset, which is page-1.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Project Budgets",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Description: "Maximum number of budgets to return. Only applies alongside cursor; ignored " +
							"when paging with page/page_size.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"cursor": {
						Description: "Opaque cursor from a previous response, for cursor pagination. This is not an " +
							"offset or a page number — never construct one. Setting it makes the endpoint ignore " +
							"page and page_size. To walk pages, use page instead.",
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
				helpers.OptionalNumericParam(&projectBudgetListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&projectBudgetListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&projectBudgetListRequest.Filters.Cursor, "cursor"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if !verbose {
				projectBudgetListRequest.Filters.Fields.Budgets = projectBudgetSparseFields
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
			if !verbose {
				if body, err = helpers.ProjectCollectionFields(body, "budgets", projectBudgetSparseKeys); err != nil {
					return nil, fmt.Errorf("failed to trim project budgets: %w", err)
				}
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
				Title:           "List Tasklist Budgets",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
