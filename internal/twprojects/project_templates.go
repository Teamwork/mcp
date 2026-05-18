package twprojects

import (
	"context"
	"encoding/json"

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
	MethodProjectTemplateCreate toolsets.Method = "twprojects-create_project_template"
	MethodProjectTemplateList   toolsets.Method = "twprojects-list_project_templates"
)

// ProjectTemplateCreate creates a project template in Teamwork.com.
func ProjectTemplateCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectTemplateCreate),
			Description: "Create project template.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Project Template",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the project template.",
					},
					"description": {
						Description: "The description of the project template.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"start_at": {
						Description: "Start date of the project template (format: YYYYMMDD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"end_at": {
						Description: "End date of the project template (format: YYYYMMDD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"category_id": {
						Description: "The ID of the category to which the project template belongs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"company_id": {
						Description: "The ID of the company associated with the project template.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"owned_id": {
						Description: "The ID of the user who owns the project template.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("project template"),
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCreateRequest projects.ProjectTemplateCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&projectCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectCreateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&projectCreateRequest.StartAt, "start_at"),
				helpers.OptionalLegacyDatePointerParam(&projectCreateRequest.EndAt, "end_at"),
				helpers.OptionalNumericPointerParam(&projectCreateRequest.CategoryID, "category_id"),
				helpers.OptionalNumericParam(&projectCreateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&projectCreateRequest.OwnerID, "owned_id"),
				helpers.OptionalNumericListParam(&projectCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			project, err := projects.ProjectTemplateCreate(ctx, engine, projectCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create project template")
			}
			return helpers.NewToolResultText("Project template created successfully with ID %d", project.ID), nil
		},
	}
}

// ProjectTemplateList lists project templates in Teamwork.com.
func ProjectTemplateList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectTemplateList),
			Description: "List project templates.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Project Templates",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_category_ids": {
						Description: "Filter project templates by category.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"search_term":    helpers.SearchTermSchema("project templates", "name or description"),
					"tag_ids":        helpers.TagIDsFilterSchema("project templates"),
					"match_all_tags": helpers.MatchAllTagsSchema(),
					"page":           helpers.PageSchema(),
					"page_size":      helpers.PageSizeSchema(),
				},
				Required: []string{},
			},
			OutputSchema: projectListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectListRequest projects.ProjectTemplateListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericListParam(&projectListRequest.Filters.ProjectCategoryIDs, "project_category_ids"),
				helpers.OptionalParam(&projectListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&projectListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&projectListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			projectList, err := projects.ProjectTemplateList(ctx, engine, projectListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list project templates")
			}

			encoded, err := json.Marshal(projectList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/projects"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, projectList,
					helpers.WebLinkerWithIDPathBuilder("/app/projects"),
				),
			}, nil
		},
	}
}
