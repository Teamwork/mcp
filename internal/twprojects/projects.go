package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodProjectCreate)
	toolsets.RegisterMethod(MethodProjectUpdate)
	toolsets.RegisterMethod(MethodProjectDelete)
	toolsets.RegisterMethod(MethodProjectGet)
	toolsets.RegisterMethod(MethodProjectList)
}

// ProjectCreate creates a project in Teamwork.com.
func ProjectCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodProjectCreate),
			mcp.WithDescription("Create a new project in Teamwork.com. "+projectDescription),
			mcp.WithTitleAnnotation("Create Project"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the project."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the project."),
			),
			mcp.WithString("start_at",
				mcp.Description("The start date of the project in the format YYYYMMDD."),
			),
			mcp.WithString("end_at",
				mcp.Description("The end date of the project in the format YYYYMMDD."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the company associated with the project."),
			),
			mcp.WithNumber("owned_id",
				mcp.Description("The ID of the user who owns the project."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the project."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCreateRequest projects.ProjectCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredParam(&projectCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectCreateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&projectCreateRequest.StartAt, "start_at"),
				helpers.OptionalLegacyDatePointerParam(&projectCreateRequest.EndAt, "end_at"),
				helpers.OptionalNumericParam(&projectCreateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&projectCreateRequest.OwnerID, "owned_id"),
				helpers.OptionalNumericListParam(&projectCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			project, err := projects.ProjectCreate(ctx, engine, projectCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create project")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Project created successfully with ID %d", project.ID)), nil
		},
	}
}

// ProjectUpdate updates a project in Teamwork.com.
func ProjectUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodProjectUpdate),
			mcp.WithDescription("Update an existing project in Teamwork.com. "+projectDescription),
			mcp.WithTitleAnnotation("Update Project"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the project."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the project."),
			),
			mcp.WithString("start_at",
				mcp.Description("The start date of the project in the format YYYYMMDD."),
			),
			mcp.WithString("end_at",
				mcp.Description("The end date of the project in the format YYYYMMDD."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the company associated with the project."),
			),
			mcp.WithNumber("owned_id",
				mcp.Description("The ID of the user who owns the project."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the project."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectUpdateRequest projects.ProjectUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&projectUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&projectUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectUpdateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&projectUpdateRequest.StartAt, "start_at"),
				helpers.OptionalLegacyDatePointerParam(&projectUpdateRequest.EndAt, "end_at"),
				helpers.OptionalNumericPointerParam(&projectUpdateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&projectUpdateRequest.OwnerID, "owned_id"),
				helpers.OptionalNumericListParam(&projectUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.ProjectUpdate(ctx, engine, projectUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update project")
			}

			return mcp.NewToolResultText("Project updated successfully"), nil
		},
	}
}

// ProjectDelete deletes a project in Teamwork.com.
func ProjectDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodProjectDelete),
			mcp.WithDescription("Delete an existing project in Teamwork.com. "+projectDescription),
			mcp.WithTitleAnnotation("Delete Project"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectDeleteRequest projects.ProjectDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&projectDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.ProjectDelete(ctx, engine, projectDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete project")
			}

			return mcp.NewToolResultText("Project deleted successfully"), nil
		},
	}
}

// ProjectGet retrieves a project in Teamwork.com.
func ProjectGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodProjectGet),
			mcp.WithDescription("Get an existing project in Teamwork.com. "+projectDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Project"),
			mcp.WithOutputSchema[projects.ProjectGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the project to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectGetRequest projects.ProjectGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&projectGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			project, err := projects.ProjectGet(ctx, engine, projectGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get project")
			}

			encoded, err := json.Marshal(project)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/projects"),
			))), nil
		},
	}
}

// ProjectList lists projects in Teamwork.com.
func ProjectList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodProjectList),
			mcp.WithDescription("List projects in Teamwork.com. "+projectDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Projects"),
			mcp.WithOutputSchema[projects.ProjectListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter projects by name or description."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter projects by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match projects that have all the specified tags. "+
					"If false, the search will match projects that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectListRequest projects.ProjectListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&projectListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&projectListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&projectListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&projectListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&projectListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			projectList, err := projects.ProjectList(ctx, engine, projectListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list projects")
			}

			encoded, err := json.Marshal(projectList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/projects"),
			))), nil
		},
	}
}
