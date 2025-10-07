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

var (
	teamGetOutputSchema  *jsonschema.Schema
	teamListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTeamCreate)
	toolsets.RegisterMethod(MethodTeamUpdate)
	toolsets.RegisterMethod(MethodTeamDelete)
	toolsets.RegisterMethod(MethodTeamGet)
	toolsets.RegisterMethod(MethodTeamList)
	toolsets.RegisterMethod(MethodTeamListByCompany)
	toolsets.RegisterMethod(MethodTeamListByProject)

	var err error

	// generate the output schemas only once
	teamGetOutputSchema, err = jsonschema.For[projects.TeamGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TeamGetResponse: %v", err))
	}
	teamListOutputSchema, err = jsonschema.For[projects.TeamListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for TeamListResponse: %v", err))
	}
}

// TeamCreate creates a team in Teamwork.com.
func TeamCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamCreate),
			Description: "Create a new team in Teamwork.com. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Team",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the team.",
					},
					"handle": {
						Type: "string",
						Description: "The handle of the team. It is a unique identifier for the team. It must not have spaces " +
							"or special characters.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the team.",
					},
					"parent_team_id": {
						Type:        "integer",
						Description: "The ID of the parent team. This is used to create a hierarchy of teams.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the company. This is used to create a team scoped for a specific company.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project. This is used to create a team scoped for a specific project.",
					},
					"user_ids": {
						Type:        "array",
						Description: "A list of user IDs to add to the team.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamCreateRequest projects.TeamCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&teamCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&teamCreateRequest.Handle, "handle"),
				helpers.OptionalPointerParam(&teamCreateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&teamCreateRequest.ParentTeamID, "parent_team_id"),
				helpers.OptionalNumericPointerParam(&teamCreateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&teamCreateRequest.ProjectID, "project_id"),
				helpers.OptionalCustomNumericListParam(&teamCreateRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			team, err := projects.TeamCreate(ctx, engine, teamCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create team")
			}
			return helpers.NewToolResultText("Team created successfully with ID %d", team.ID), nil
		},
	}
}

// TeamUpdate updates a team in Teamwork.com.
func TeamUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamUpdate),
			Description: "Update an existing team in Teamwork.com. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Team",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the team to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the team.",
					},
					"handle": {
						Type: "string",
						Description: "The handle of the team. It is a unique identifier for the team. It must not have spaces " +
							"or special characters.",
					},
					"description": {
						Type:        "string",
						Description: "The description of the team.",
					},
					"parent_team_id": {
						Type:        "integer",
						Description: "The ID of the parent team. This is used to create a hierarchy of teams.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the company. This is used to create a team scoped for a specific company.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project. This is used to create a team scoped for a specific project.",
					},
					"user_ids": {
						Type:        "array",
						Description: "A list of user IDs to add to the team.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamUpdateRequest projects.TeamUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&teamUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&teamUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&teamUpdateRequest.Handle, "handle"),
				helpers.OptionalPointerParam(&teamUpdateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&teamUpdateRequest.CompanyID, "company_id"),
				helpers.OptionalNumericPointerParam(&teamUpdateRequest.ProjectID, "project_id"),
				helpers.OptionalCustomNumericListParam(&teamUpdateRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TeamUpdate(ctx, engine, teamUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update team")
			}
			return helpers.NewToolResultText("Team updated successfully"), nil
		},
	}
}

// TeamDelete deletes a team in Teamwork.com.
func TeamDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamDelete),
			Description: "Delete an existing team in Teamwork.com. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Team",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the team to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamDeleteRequest projects.TeamDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&teamDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.TeamDelete(ctx, engine, teamDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete team")
			}
			return helpers.NewToolResultText("Team deleted successfully"), nil
		},
	}
}

// TeamGet retrieves a team in Teamwork.com.
func TeamGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamGet),
			Description: "Get an existing team in Teamwork.com. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Team",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the team to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: teamGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamGetRequest projects.TeamGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&teamGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			team, err := projects.TeamGet(ctx, engine, teamGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get team")
			}

			encoded, err := json.Marshal(team)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/teams"),
						)),
					},
				},
				StructuredContent: team,
			}, nil
		},
	}
}

// TeamList lists teams in Teamwork.com.
func TeamList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamList),
			Description: "List teams in Teamwork.com. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Teams",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type:        "string",
						Description: "A search term to filter teams by name or handle.",
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
			OutputSchema: teamListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamListRequest projects.TeamListRequest

			// to simplify the teams logic for the LLM, always return all team types
			teamListRequest.Filters.IncludeCompanyTeams = true
			teamListRequest.Filters.IncludeProjectTeams = true
			teamListRequest.Filters.IncludeSubteams = true

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&teamListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			teamList, err := projects.TeamList(ctx, engine, teamListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list teams")
			}

			encoded, err := json.Marshal(teamList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/teams"),
						)),
					},
				},
				StructuredContent: teamList,
			}, nil
		},
	}
}

// TeamListByCompany lists teams in Teamwork.com by client/company.
func TeamListByCompany(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamListByCompany),
			Description: "List teams in Teamwork.com by client/company. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Teams By Company",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"company_id": {
						Type:        "integer",
						Description: "The ID of the company from which to retrieve teams.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter teams by name or handle.",
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
				Required: []string{"company_id"},
			},
			OutputSchema: teamListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamListRequest projects.TeamListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&teamListRequest.Path.CompanyID, "company_id"),
				helpers.OptionalParam(&teamListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			teamList, err := projects.TeamList(ctx, engine, teamListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list teams")
			}

			encoded, err := json.Marshal(teamList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/teams"),
						)),
					},
				},
				StructuredContent: teamList,
			}, nil
		},
	}
}

// TeamListByProject lists teams in Teamwork.com by project.
func TeamListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodTeamListByProject),
			Description: "List teams in Teamwork.com by project. " + teamDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Teams By Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project from which to retrieve teams.",
					},
					"search_term": {
						Type:        "string",
						Description: "A search term to filter teams by name or handle.",
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
			OutputSchema: teamListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var teamListRequest projects.TeamListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&teamListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&teamListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&teamListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			teamList, err := projects.TeamList(ctx, engine, teamListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list teams")
			}

			encoded, err := json.Marshal(teamList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/teams"),
						)),
					},
				},
				StructuredContent: teamList,
			}, nil
		},
	}
}
