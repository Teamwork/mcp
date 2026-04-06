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
	MethodProjectCreate toolsets.Method = "twprojects-create_project"
	MethodProjectUpdate toolsets.Method = "twprojects-update_project"
	MethodProjectDelete toolsets.Method = "twprojects-delete_project"
	MethodProjectClone  toolsets.Method = "twprojects-clone_project"
	MethodProjectGet    toolsets.Method = "twprojects-get_project"
	MethodProjectList   toolsets.Method = "twprojects-list_projects"
)

const projectDescription = "The project feature in Teamwork.com serves as the central workspace for organizing and " +
	"managing a specific piece of work or initiative. Each project provides a dedicated area where teams can plan " +
	"tasks, assign responsibilities, set deadlines, and track progress toward shared goals. Projects include tools " +
	"for communication, file sharing, milestones, and time tracking, allowing teams to stay aligned and informed " +
	"throughout the entire lifecycle of the work. Whether it's a product launch, client engagement, or internal " +
	"initiative, projects in Teamwork.com help teams structure their efforts, collaborate more effectively, and " +
	"deliver results with greater visibility and accountability."

var (
	projectGetOutputSchema  *jsonschema.Schema
	projectListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	projectGetOutputSchema, err = jsonschema.For[projects.ProjectGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ProjectGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(projectGetOutputSchema)
	projectListOutputSchema, err = jsonschema.For[projects.ProjectListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ProjectListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(projectListOutputSchema)
}

// ProjectCreate creates a project in Teamwork.com.
func ProjectCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectCreate),
			Description: "Create a new project in Teamwork.com. " + projectDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Project",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the project.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the project.",
					},
					"start_at": {
						Type:        "string",
						Description: "The start date of the project in the format YYYYMMDD.",
					},
					"end_at": {
						Type:        "string",
						Description: "The end date of the project in the format YYYYMMDD.",
					},
					"category_id": {
						Type:        "integer",
						Description: "The ID of the category to which the project belongs.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the company associated with the project.",
					},
					"owned_id": {
						Type:        "integer",
						Description: "The ID of the user who owns the project.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the project.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCreateRequest projects.ProjectCreateRequest

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

			project, err := projects.ProjectCreate(ctx, engine, projectCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create project")
			}
			return helpers.NewToolResultText("Project created successfully with ID %d", project.ID), nil
		},
	}
}

// ProjectUpdate updates a project in Teamwork.com.
func ProjectUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectUpdate),
			Description: "Update an existing project in Teamwork.com. " + projectDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Project",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the project.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the project.",
					},
					"start_at": {
						Type:        "string",
						Description: "The start date of the project in the format YYYYMMDD.",
					},
					"end_at": {
						Type:        "string",
						Description: "The end date of the project in the format YYYYMMDD.",
					},
					"category_id": {
						Type:        "integer",
						Description: "The ID of the category to which the project belongs.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the company associated with the project.",
					},
					"owned_id": {
						Type:        "integer",
						Description: "The ID of the user who owns the project.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the project.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"status": {
						Type:        "string",
						Description: "The status of the project. Allowed values: active or archived.",
						Enum:        []any{"active", "archived"},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectUpdateRequest projects.ProjectUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&projectUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectUpdateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&projectUpdateRequest.StartAt, "start_at"),
				helpers.OptionalLegacyDatePointerParam(&projectUpdateRequest.EndAt, "end_at"),
				helpers.OptionalNumericPointerParam(&projectUpdateRequest.CategoryID, "category_id"),
				helpers.OptionalNumericPointerParam(&projectUpdateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&projectUpdateRequest.OwnerID, "owned_id"),
				helpers.OptionalNumericListParam(&projectUpdateRequest.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&projectUpdateRequest.Status, "status",
					helpers.RestrictValues(
						projects.ProjectStatusActive,
						projects.ProjectStatusArchived,
					),
				),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.ProjectUpdate(ctx, engine, projectUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update project")
			}
			return helpers.NewToolResultText("Project updated successfully"), nil
		},
	}
}

// ProjectDelete deletes a project in Teamwork.com.
func ProjectDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectDelete),
			Description: "Delete an existing project in Teamwork.com. " + projectDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Project",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectDeleteRequest projects.ProjectDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.ProjectDelete(ctx, engine, projectDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete project")
			}
			return helpers.NewToolResultText("Project deleted successfully"), nil
		},
	}
}

// ProjectClone clones a project in Teamwork.com.
func ProjectClone(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodProjectClone),
			Description: "Create a new project by cloning/copying an existing one or generating it from a project " +
				"template. " + projectDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Clone Project",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to clone.",
					},
					"name": {
						Type: "string",
						Description: "The name of the new cloned project. If not provided, the name of the original project " +
							"will be used with an incremental suffix (e.g., 'Project Name (1)').",
					},
					"description": {
						Type: "string",
						Description: "The description of the new cloned project. If not provided, the description of the " +
							"original project will be used.",
					},
					"company_id": {
						Type: "integer",
						Description: "The ID of the company associated with the new cloned project. If not provided, the company " +
							"of the original project will be used.",
					},
					"new_from_template": {
						Type:        "boolean",
						Description: "Indicates whether the new project should be a regular one created from a template.",
					},
					"to_template": {
						Type:        "boolean",
						Description: "Indicates whether the new project should be set as a template.",
					},
					"template_date_target": {
						Type: "string",
						Description: "Specifies whether target_date represents the project's " +
							"start or end date. When 'end', the start date is calculated by subtracting the template project's duration " +
							"from target_date. Only applicable when new_from_template=true.",
						Enum:    []any{"start", "end"},
						Default: json.RawMessage(`"start"`),
					},
					"target_date": {
						Type: "string",
						Description: "Target date is the desired start or end date for the cloned project " +
							"(determined by template_date_target). Used only when creating a project from " +
							"a template (new_from_template=true). Accepted format: YYYYMMDD string. " +
							"Defaults to the current user date if omitted.",
					},
					"days_offset": {
						Type: "integer",
						Description: "DaysOffset is the number of days to shift all scheduled dates in the cloned " +
							"project relative to the base date. When cloning from a template, it defines " +
							"the project duration span. When copying an existing project, it shifts the " +
							"original start and end dates by this many days. If omitted, calculated " +
							"automatically from the source project's date range.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCloneRequest projects.ProjectCloneRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectCloneRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&projectCloneRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectCloneRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&projectCloneRequest.CompanyID, "company_id"),
				helpers.OptionalPointerParam(&projectCloneRequest.NewFromTemplate, "new_from_template"),
				helpers.OptionalPointerParam(&projectCloneRequest.ToTemplate, "to_template"),
				helpers.OptionalPointerParam(&projectCloneRequest.TemplateDateTarget, "template_date_target",
					helpers.RestrictValues(
						projects.ProjectCloneTemplateDateTargetStart,
						projects.ProjectCloneTemplateDateTargetEnd,
					),
				),
				helpers.OptionalLegacyDatePointerParam(&projectCloneRequest.TargetDate, "target_date"),
				helpers.OptionalNumericPointerParam(&projectCloneRequest.DaysOffset, "days_offset"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			projectCloneRequest.Action = new(projects.ProjectCloneActionCopy)
			projectCloneRequest.CopyFiles = new(true)
			projectCloneRequest.CopyMessages = new(true)
			projectCloneRequest.CopyMilestones = new(true)
			projectCloneRequest.CopyTasks = new(true)
			projectCloneRequest.CopyTasklists = new(true)
			projectCloneRequest.CopyNotebooks = new(true)
			projectCloneRequest.CopyLinks = new(true)
			projectCloneRequest.CopyComments = new(true)
			projectCloneRequest.CopyFollowers = new(true)
			projectCloneRequest.CopyInvoices = new(true)
			projectCloneRequest.CopyTimelogs = new(true)
			projectCloneRequest.CopyExpenses = new(true)
			projectCloneRequest.CopyWebhooks = new(true)
			projectCloneRequest.CopyProjectRoles = new(true)
			projectCloneRequest.CopyCustomFields = new(true)
			projectCloneRequest.CopyCustomItems = new(true)
			projectCloneRequest.CopyProjectUpdates = new(true)
			projectCloneRequest.CopyRisks = new(true)
			projectCloneRequest.CopyForms = new(true)
			projectCloneRequest.CopyAutomations = new(true)
			projectCloneRequest.CopyPeople = new(true)
			projectCloneRequest.CopyProjectPrivacy = new(true)
			projectCloneRequest.CopyBudgets = new(true)
			projectCloneRequest.CopyAllocations = new(true)
			projectCloneRequest.CopyLogo = new(true)
			projectCloneRequest.CopyProjectPreferences = new(true)

			project, err := projects.ProjectClone(ctx, engine, projectCloneRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to clone project")
			}
			return helpers.NewToolResultText("Project cloned successfully with ID %d", project.ID), nil
		},
	}
}

// ProjectGet retrieves a project in Teamwork.com.
func ProjectGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectGet),
			Description: "Get an existing project in Teamwork.com. " + projectDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: projectGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectGetRequest projects.ProjectGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			project, err := projects.ProjectGet(ctx, engine, projectGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get project")
			}

			encoded, err := json.Marshal(project)
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
				StructuredContent: helpers.StructuredWebLinker(ctx, project,
					helpers.WebLinkerWithIDPathBuilder("/app/projects"),
				),
			}, nil
		},
	}
}

// ProjectList lists projects in Teamwork.com.
func ProjectList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectList),
			Description: "List projects in Teamwork.com. " + projectDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Projects",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_category_ids": {
						Type:        "array",
						Description: "A list of project category IDs to filter projects by categories.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter projects by name or description.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter projects by tags.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match projects that have all the specified tags. If false, the " +
							"search will match projects that have any of the specified tags. Defaults to false.",
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
			var projectListRequest projects.ProjectListRequest

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

			projectList, err := projects.ProjectList(ctx, engine, projectListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list projects")
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
