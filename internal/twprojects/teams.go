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
	MethodTeamCreate        toolsets.Method = "twprojects-create_team"
	MethodTeamUpdate        toolsets.Method = "twprojects-update_team"
	MethodTeamDelete        toolsets.Method = "twprojects-delete_team"
	MethodTeamGet           toolsets.Method = "twprojects-get_team"
	MethodTeamList          toolsets.Method = "twprojects-list_teams"
	MethodTeamListByCompany toolsets.Method = "twprojects-list_teams_by_company"
	MethodTeamListByProject toolsets.Method = "twprojects-list_teams_by_project"
)

const teamDescription = "In the context of Teamwork.com, a team is a group of users who are organized together to " +
	"collaborate more efficiently on projects and tasks. Teams help structure work by grouping individuals with " +
	"similar roles, responsibilities, or departmental functions, making it easier to assign work, track progress, " +
	"and manage communication. By using teams, organizations can streamline project planning and ensure the right " +
	"people are involved in the right parts of a project, enhancing clarity and accountability across the platform."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTeamCreate)
	toolsets.RegisterMethod(MethodTeamUpdate)
	toolsets.RegisterMethod(MethodTeamDelete)
	toolsets.RegisterMethod(MethodTeamGet)
	toolsets.RegisterMethod(MethodTeamList)
	toolsets.RegisterMethod(MethodTeamListByCompany)
	toolsets.RegisterMethod(MethodTeamListByProject)
}

// TeamCreate creates a team in Teamwork.com.
func TeamCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamCreate),
			mcp.WithDescription("Create a new team in Teamwork.com. "+teamDescription),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the team."),
			),
			mcp.WithString("handle",
				mcp.Description("The handle of the team. It is a unique identifier for the team. It must not have spaces "+
					"or special characters."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the team."),
			),
			mcp.WithNumber("parent_team_id",
				mcp.Description("The ID of the parent team. This is used to create a hierarchy of teams."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the company. This is used to create a team scoped for a specific company."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project. This is used to create a team scoped for a specific project."),
			),
			mcp.WithArray("user_ids",
				mcp.Description("A list of user IDs to add to the team."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamCreateRequest projects.TeamCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredParam(&teamCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&teamCreateRequest.Handle, "handle"),
				helpers.OptionalPointerParam(&teamCreateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&teamCreateRequest.ParentTeamID, "parent_team_id"),
				helpers.OptionalNumericPointerParam(&teamCreateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&teamCreateRequest.ProjectID, "project_id"),
				helpers.OptionalCustomNumericListParam(&teamCreateRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			team, err := projects.TeamCreate(ctx, engine, teamCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create team")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Team created successfully with ID %d", team.ID)), nil
		},
	}
}

// TeamUpdate updates a team in Teamwork.com.
func TeamUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamUpdate),
			mcp.WithDescription("Update an existing team in Teamwork.com. "+teamDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the team to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the team."),
			),
			mcp.WithString("handle",
				mcp.Description("The handle of the team. It is a unique identifier for the team. It must not have spaces "+
					"or special characters."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the team."),
			),
			mcp.WithNumber("parent_team_id",
				mcp.Description("The ID of the parent team. This is used to create a hierarchy of teams."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the company. This is used to create a team scoped for a specific company."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project. This is used to create a team scoped for a specific project."),
			),
			mcp.WithArray("user_ids",
				mcp.Description("A list of user IDs to add to the team."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamUpdateRequest projects.TeamUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&teamUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&teamUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&teamUpdateRequest.Handle, "handle"),
				helpers.OptionalPointerParam(&teamUpdateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&teamUpdateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&teamUpdateRequest.ProjectID, "project_id"),
				helpers.OptionalCustomNumericListParam(&teamUpdateRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TeamUpdate(ctx, engine, teamUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update team")
			}

			return mcp.NewToolResultText("Team updated successfully"), nil
		},
	}
}

// TeamDelete deletes a team in Teamwork.com.
func TeamDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamDelete),
			mcp.WithDescription("Delete an existing team in Teamwork.com. "+teamDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the team to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamDeleteRequest projects.TeamDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&teamDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TeamDelete(ctx, engine, teamDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete team")
			}

			return mcp.NewToolResultText("Team deleted successfully"), nil
		},
	}
}

// TeamGet retrieves a team in Teamwork.com.
func TeamGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamGet),
			mcp.WithDescription("Get an existing team in Teamwork.com. "+teamDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the team to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamGetRequest projects.TeamGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&teamGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			team, err := projects.TeamGet(ctx, engine, teamGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get team")
			}

			encoded, err := json.Marshal(team)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/teams"),
			))), nil
		},
	}
}

// TeamList lists teams in Teamwork.com.
func TeamList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamList),
			mcp.WithDescription("List teams in Teamwork.com. "+teamDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter teams by name or handle."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamListRequest projects.TeamListRequest

			// to simplify the teams logic for the LLM, always return all team types
			teamListRequest.Filters.IncludeCompanyTeams = true
			teamListRequest.Filters.IncludeProjectTeams = true
			teamListRequest.Filters.IncludeSubteams = true

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&teamListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			teamList, err := projects.TeamList(ctx, engine, teamListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list teams")
			}

			encoded, err := json.Marshal(teamList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/teams"),
			))), nil
		},
	}
}

// TeamListByCompany lists teams in Teamwork.com by client/company.
func TeamListByCompany(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamListByCompany),
			mcp.WithDescription("List teams in Teamwork.com by client/company. "+teamDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("company_id",
				mcp.Required(),
				mcp.Description("The ID of the company from which to retrieve teams."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter teams by name or handle."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamListRequest projects.TeamListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&teamListRequest.Path.CompanyID, "company_id"),
				helpers.OptionalParam(&teamListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			teamList, err := projects.TeamList(ctx, engine, teamListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list teams")
			}

			encoded, err := json.Marshal(teamList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/teams"),
			))), nil
		},
	}
}

// TeamListByProject lists teams in Teamwork.com by project.
func TeamListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTeamListByProject),
			mcp.WithDescription("List teams in Teamwork.com by project. "+teamDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project from which to retrieve teams."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter teams by name or handle."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamListRequest projects.TeamListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&teamListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&teamListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			teamList, err := projects.TeamList(ctx, engine, teamListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list teams")
			}

			encoded, err := json.Marshal(teamList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/teams"),
			))), nil
		},
	}
}
