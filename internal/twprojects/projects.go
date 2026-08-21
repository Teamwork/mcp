package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
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

var (
	projectGetOutputSchema  *jsonschema.Schema
	projectListOutputSchema *jsonschema.Schema
)

// projectOrdering is the order-by vocabulary of the projects list endpoint.
var projectOrdering = newOrdering("projects",
	projects.ProjectOrderByBudgetUsed,
	projects.ProjectOrderByCategoryName,
	projects.ProjectOrderByCompanyName,
	projects.ProjectOrderByCreatorName,
	projects.ProjectOrderByCustomField,
	projects.ProjectOrderByDateCreated,
	projects.ProjectOrderByDueDate,
	projects.ProjectOrderByHealth,
	projects.ProjectOrderByLastActivity,
	projects.ProjectOrderByLastWorkedOn,
	projects.ProjectOrderByMobileSpecial,
	projects.ProjectOrderByName,
	projects.ProjectOrderByNameCaseInsensitive,
	projects.ProjectOrderByOwnerCompany,
	projects.ProjectOrderByOwnerName,
	projects.ProjectOrderByStarred,
	projects.ProjectOrderByStarredCompanyName,
	projects.ProjectOrderByStarredFirst,
	projects.ProjectOrderByStartDate,
	projects.ProjectOrderByStatus,
	projects.ProjectOrderByTaskCompletion,
	projects.ProjectOrderByID,
)

// projectStatusVocabulary is the progress-state vocabulary of the projects list
// endpoint. It is the only way to ask which projects have slipped: a project row
// carries no late flag of its own — its status attribute is the storage state
// (active, archived, deleted), not the schedule — so "late" is computed by the
// endpoint from the filter and can never be read off a response.
var projectStatusVocabulary = newVocabulary(
	projects.ProjectListStatusActive,
	projects.ProjectListStatusCurrent,
	projects.ProjectListStatusLate,
	projects.ProjectListStatusUpcoming,
	projects.ProjectListStatusCompleted,
	projects.ProjectListStatusDeleted,
)

// projectHealthVocabulary is the health-rating vocabulary of the projects list
// endpoint. The API models health as an integer whose meaning is positional, so
// the tool publishes names and maps them here.
var projectHealthVocabulary = newNamedVocabulary(
	[]string{"good", "ok", "bad", "not_set"},
	[]projects.ProjectHealth{
		projects.ProjectHealthGood,
		projects.ProjectHealthOK,
		projects.ProjectHealthBad,
		projects.ProjectHealthNotSet,
	},
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
			Description: "Create project.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Project",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the project.",
					},
					"description": {
						Description: "The description of the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"start_at": {
						Description: "Start date of the project (format: YYYYMMDD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"end_at": {
						Description: "End date of the project (format: YYYYMMDD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"category_id": {
						Description: "The ID of the category to which the project belongs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"company_id": {
						Description: "The ID of the company associated with the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"owned_id": {
						Description: "The ID of the user who owns the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("project"),
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCreateRequest projects.ProjectCreateRequest

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
			Description: "Update project.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Project",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to update.",
					},
					"name": {
						Description: "The name of the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"start_at": {
						Description: "Start date of the project (format: YYYYMMDD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"end_at": {
						Description: "End date of the project (format: YYYYMMDD).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"category_id": {
						Description: "The ID of the category to which the project belongs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"company_id": {
						Description: "The ID of the company associated with the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"owned_id": {
						Description: "The ID of the user who owns the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("project"),
					"status": {
						Description: "The status of the project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"active", "archived"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectUpdateRequest projects.ProjectUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "Delete project.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Project",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Name:        string(MethodProjectClone),
			Description: "Clone/copy an existing project or instantiate one from a template.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Clone Project",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to clone.",
					},
					"name": {
						Description: "The name of the new cloned project. If not provided, the name of the original project " +
							"will be used with an incremental suffix (e.g., 'Project Name (1)').",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "The description of the new cloned project. If not provided, the description of the " +
							"original project will be used.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"company_id": {
						Description: "The ID of the company associated with the new cloned project. If not provided, the company " +
							"of the original project will be used.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"new_from_template": {
						Description: "Indicates whether the new project should be a regular one created from a template.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"to_template": {
						Description: "Indicates whether the new project should be set as a template.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"template_date_target": {
						Description: "Specifies whether target_date represents the project's " +
							"start or end date. When 'end', the start date is calculated by subtracting the template project's duration " +
							"from target_date. Only applicable when new_from_template=true.",
						Default: json.RawMessage(`"start"`),
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"start", "end"}},
							{Type: "null"},
						},
					},
					"target_date": {
						Description: "Desired start or end date for the cloned project (chosen by template_date_target). " +
							"Only applies when new_from_template=true. Format: YYYYMMDD. " +
							"Defaults to today.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"days_offset": {
						Description: "DaysOffset is the number of days to shift all scheduled dates in the cloned " +
							"project relative to the base date. When cloning from a template, it defines " +
							"the project duration span. When copying an existing project, it shifts the " +
							"original start and end dates by this many days. If omitted, calculated " +
							"automatically from the source project's date range.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCloneRequest projects.ProjectCloneRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "Get project.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Project",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the project to get.",
					},
					"fields": helpers.FieldsSchema[projects.Project]("project"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(projectGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectGetRequest projects.ProjectGetRequest

			// Always include project categories and custom fields in the response to
			// provide more context about the project. Categories are commonly used
			// for organizing projects and understanding their purpose, and custom
			// fields often contain important metadata relevant to the project.
			projectGetRequest.Filters.Include = []projects.ProjectRequestSideload{
				projects.ProjectRequestSideloadProjectCategories,
				projects.ProjectRequestSideloadCustomFields,
				projects.ProjectRequestSideloadCustomFieldValues,
			}

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.Project](&projectGetRequest.Fields.Project, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(projectGetRequest.Fields.Project) > 0 {
				// Drop the sideloads: they are not what the selection named, and
				// they would return the bulk it exists to avoid.
				projectGetRequest.Filters = projects.ProjectRequestFilters{}
				return helpers.NewRawToolResult(ctx, engine, projectGetRequest, "failed to get project",
					helpers.WebLinkerWithIDPathBuilder("/app/projects"),
				)
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
			Name: string(MethodProjectList),
			Description: "List projects with structured filters (membership, progress state, health, category, " +
				"tag, company, owner). For \"my projects\" pass user_ids with the ID from twprojects-get_user_me, " +
				"or only_starred. For projects that have slipped pass project_statuses=[\"late\"] — a project row " +
				"carries no late flag, so this filter is the only way to ask.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Projects",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_category_ids": {
						Description: "Filter projects by category.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"include_subcategories": {
						Description: "If true, project_category_ids also matches the categories nested under the " +
							"ones given.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"project_statuses": projectStatusVocabulary.arraySchema(
						"Filter projects by progress state, matching any of the values given. \"late\" is past its " +
							"end date and not yet completed, \"upcoming\" has not started yet, \"current\" is " +
							"running now, and \"active\" is every project that is neither completed nor archived. " +
							"Omit to keep the endpoint's own default set.",
					),
					"project_healths": projectHealthVocabulary.arraySchema(
						"Filter projects by the health rating set on them, matching any of the values given. " +
							"\"not_set\" matches the projects nobody has rated.",
					),
					"user_ids": {
						Description: "Filter projects by the users holding an explicit membership of them. For " +
							"\"my projects\", pass the ID returned by twprojects-get_user_me.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"team_ids": {
						Description: "Filter projects by team, matching the projects any member of those teams " +
							"belongs to.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_owner_ids": {
						Description: "Filter projects by the user who owns them.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"company_ids": {
						Description: "Filter projects by the company that owns them.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"only_starred": {
						Description: "If true, only return the projects the calling user has starred.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"only_admin_access": {
						Description: "If true, only return the projects the calling user administers.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"hide_observed": {
						Description: "If true, leave out the projects the calling user only observes, keeping the " +
							"ones they actually work on.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include_archived": {
						Description: "If true, return archived projects alongside the active ones; excluded by " +
							"default.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"only_archived": {
						Description: "If true, only return archived projects.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include_tentative": {
						Description: "If true, return tentative projects alongside the normal ones; excluded by " +
							"default.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"updated_after":            helpers.DateTimeFilterSchema("Filter projects updated after."),
					"search_term":              helpers.SearchTermSchema("projects", "name or description"),
					"tag_ids":                  helpers.TagIDsFilterSchema("projects"),
					"match_all_tags":           helpers.MatchAllTagsSchema(),
					"order_by":                 projectOrdering.orderBySchema(),
					"order_mode":               orderModeSchema(),
					"order_by_custom_field_id": orderByFieldIDSchema("projects", "customfield"),
					"page":                     helpers.PageSchema(),
					"page_size":                helpers.PageSizeSchema(),
					"verbose":                  helpers.VerboseSchema(),
					"count_only":               helpers.CountOnlySchema("projects"),
					"fields":                   helpers.FieldsSchema[projects.Project]("project"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(
				helpers.WithOptionalFields(withSuggestionsSchema(projectListOutputSchema)),
			),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectListRequest projects.ProjectListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericListParam(&projectListRequest.Filters.ProjectCategoryIDs, "project_category_ids"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.IncludeSubCategories, "include_subcategories"),
				projectStatusVocabulary.listParam(&projectListRequest.Filters.ProjectStatuses, "project_statuses"),
				projectHealthVocabulary.listParam(&projectListRequest.Filters.ProjectHealths, "project_healths"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.UserIDs, "user_ids"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.TeamIDs, "team_ids"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.ProjectOwnerIDs, "project_owner_ids"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.ProjectCompanyIDs, "company_ids"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.OnlyStarredProjects, "only_starred"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.OnlyProjectsWithAdminAccess,
					"only_admin_access"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.HideObservedProjects, "hide_observed"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.IncludeArchivedProjects, "include_archived"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.OnlyArchivedProjects, "only_archived"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.IncludeTentativeProjects,
					"include_tentative"),
				helpers.OptionalTimePointerParam(&projectListRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalParam(&projectListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.MatchAllTags, "match_all_tags"),
				projectOrdering.param(&projectListRequest.Filters.OrderBy, &projectListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&projectListRequest.Filters.OrderByCustomFieldID, "order_by_custom_field_id"),
				helpers.OptionalNumericParam(&projectListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&projectListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.Project](&projectListRequest.Filters.Fields.Projects, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			switch {
			case len(projectListRequest.Filters.Fields.Projects) > 0:
				// An explicit field selection overrides both defaults below: the
				// caller has already said what it wants, and the sideloads would
				// smuggle back the bulk the selection exists to avoid.

			case verbose:
				// Include project categories and custom fields in the response to
				// provide more context about the project, as categories are commonly
				// used for organizing projects and understanding their purpose, and
				// custom fields often contain important metadata relevant to the
				// project.
				projectListRequest.Filters.Include = []projects.ProjectRequestSideload{
					projects.ProjectRequestSideloadProjectCategories,
					projects.ProjectRequestSideloadCustomFields,
					projects.ProjectRequestSideloadCustomFieldValues,
				}

			default:
				projectListRequest.Filters.Fields.Projects = []projects.ProjectField{
					projects.ProjectFieldID,
					projects.ProjectFieldName,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, projectListRequest, "failed to count projects")
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, projectListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list projects")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list projects"), "failed to list projects")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/projects"))
			linked, err = withNearMissSuggestions(ctx, engine, linked, "projects", projectListRequest.Filters.SearchTerm)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to generate near-miss suggestions")
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
				},
			}
			var structured any
			if err := json.Unmarshal(linked, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
