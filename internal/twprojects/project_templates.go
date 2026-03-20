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
	MethodProjectTemplateCreate toolsets.Method = "twprojects-create_project_template"
	MethodProjectTemplateList   toolsets.Method = "twprojects-list_project_templates"
)

const projectTemplateDescription = "The project template is a reusable project structure designed to standardize " +
	"workflows and streamline project setup. It typically includes predefined tasks, task lists, milestones, and " +
	"timelines that reflect a repeatable process, allowing teams to quickly spin up new projects with consistent " +
	"organization, clear responsibilities, and efficient execution from the start."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodProjectTemplateCreate)
	toolsets.RegisterMethod(MethodProjectTemplateList)
}

// ProjectTemplateCreate creates a project template in Teamwork.com.
func ProjectTemplateCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectTemplateCreate),
			Description: "Create a new project template in Teamwork.com. " + projectTemplateDescription,
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
						Type:        "string",
						Description: "The description of the project template.",
					},
					"start_at": {
						Type:        "string",
						Description: "The start date of the project template in the format YYYYMMDD.",
					},
					"end_at": {
						Type:        "string",
						Description: "The end date of the project template in the format YYYYMMDD.",
					},
					"category_id": {
						Type:        "integer",
						Description: "The ID of the category to which the project template belongs.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the company associated with the project template.",
					},
					"owned_id": {
						Type:        "integer",
						Description: "The ID of the user who owns the project template.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the project template.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCreateRequest projects.ProjectTemplateCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "List project templates in Teamwork.com. " + projectTemplateDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Project Templates",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_category_ids": {
						Type:        "array",
						Description: "A list of project category IDs to filter project templates by categories.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter project templates by name or description.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter project templates by tags.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match project templates that have all the specified tags. " +
							"If false, the search will match project templates that have any of the specified tags. " +
							"Defaults to false.",
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
			OutputSchema: projectListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectListRequest projects.ProjectTemplateListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
