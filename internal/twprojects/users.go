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
	MethodUserCreate        toolsets.Method = "twprojects-create_user"
	MethodUserUpdate        toolsets.Method = "twprojects-update_user"
	MethodUserDelete        toolsets.Method = "twprojects-delete_user"
	MethodUserGet           toolsets.Method = "twprojects-get_user"
	MethodUserGetMe         toolsets.Method = "twprojects-get_user_me"
	MethodUserList          toolsets.Method = "twprojects-list_users"
	MethodUserListByProject toolsets.Method = "twprojects-list_users_by_project"
)

const userDescription = "A user is an individual who has access to one or more projects within a Teamwork site, " +
	"typically as a team member, collaborator, or administrator. Users can be assigned tasks, participate in " +
	"discussions, log time, share files, and interact with other members depending on their permission levels. Each " +
	"user has a unique profile that defines their role, visibility, and access to features and project data. Users " +
	"can belong to clients/companies or teams within the system, and their permissions can be customized to control " +
	"what actions they can perform or what information they can see."

var (
	userGetOutputSchema   *jsonschema.Schema
	userGetMeOutputSchema *jsonschema.Schema
	userListOutputSchema  *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodUserCreate)
	toolsets.RegisterMethod(MethodUserUpdate)
	toolsets.RegisterMethod(MethodUserDelete)
	toolsets.RegisterMethod(MethodUserGet)
	toolsets.RegisterMethod(MethodUserGetMe)
	toolsets.RegisterMethod(MethodUserList)
	toolsets.RegisterMethod(MethodUserListByProject)

	var err error

	// generate the output schemas only once
	userGetOutputSchema, err = jsonschema.For[projects.UserGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserGetResponse: %v", err))
	}
	userGetMeOutputSchema, err = jsonschema.For[projects.UserGetMeResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserGetMeResponse: %v", err))
	}
	userListOutputSchema, err = jsonschema.For[projects.UserListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserListResponse: %v", err))
	}
}

// UserCreate creates a user in Teamwork.com.
func UserCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserCreate),
			Description: "Create a new user in Teamwork.com. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create User",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"first_name": {
						Type:        "string",
						Description: "The first name of the user.",
					},
					"last_name": {
						Type:        "string",
						Description: "The last name of the user.",
					},
					"title": {
						Type:        "string",
						Description: "The job title of the user, such as 'Project Manager' or 'Senior Software Developer'.",
					},
					"email": {
						Type:        "string",
						Description: "The email address of the user.",
					},
					"admin": {
						Type:        "boolean",
						Description: "Indicates whether the user is an administrator.",
					},
					"type": {
						Type:        "string",
						Description: "The type of user, such as 'account', 'collaborator', or 'contact'.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the client/company to which the user belongs.",
					},
				},
				Required: []string{"first_name", "last_name", "email"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userCreateRequest projects.UserCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&userCreateRequest.FirstName, "first_name"),
				helpers.RequiredParam(&userCreateRequest.LastName, "last_name"),
				helpers.OptionalPointerParam(&userCreateRequest.Title, "title"),
				helpers.RequiredParam(&userCreateRequest.Email, "email"),
				helpers.OptionalPointerParam(&userCreateRequest.Admin, "admin"),
				helpers.OptionalPointerParam(&userCreateRequest.Type, "type",
					helpers.RestrictValues("account", "collaborator", "contact"),
				),
				helpers.OptionalNumericPointerParam(&userCreateRequest.CompanyID, "company_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			user, err := projects.UserCreate(ctx, engine, userCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create user")
			}
			return helpers.NewToolResultText("User created successfully with ID %d", user.ID), nil
		},
	}
}

// UserUpdate updates a user in Teamwork.com.
func UserUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserUpdate),
			Description: "Update an existing user in Teamwork.com. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update User",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the user to update.",
					},
					"first_name": {
						Type:        "string",
						Description: "The first name of the user.",
					},
					"last_name": {
						Type:        "string",
						Description: "The last name of the user.",
					},
					"title": {
						Type:        "string",
						Description: "The job title of the user, such as 'Project Manager' or 'Senior Software Developer'.",
					},
					"email": {
						Type:        "string",
						Description: "The email address of the user.",
					},
					"admin": {
						Type:        "boolean",
						Description: "Indicates whether the user is an administrator.",
					},
					"type": {
						Type:        "string",
						Description: "The type of user, such as 'account', 'collaborator', or 'contact'.",
					},
					"company_id": {
						Type:        "integer",
						Description: "The ID of the client/company to which the user belongs.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userUpdateRequest projects.UserUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&userUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&userUpdateRequest.FirstName, "first_name"),
				helpers.OptionalPointerParam(&userUpdateRequest.LastName, "last_name"),
				helpers.OptionalPointerParam(&userUpdateRequest.Title, "title"),
				helpers.OptionalPointerParam(&userUpdateRequest.Email, "email"),
				helpers.OptionalPointerParam(&userUpdateRequest.Admin, "admin"),
				helpers.OptionalPointerParam(&userUpdateRequest.Type, "type",
					helpers.RestrictValues("account", "collaborator", "contact"),
				),
				helpers.OptionalNumericPointerParam(&userUpdateRequest.CompanyID, "company_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.UserUpdate(ctx, engine, userUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update user")
			}
			return helpers.NewToolResultText("User updated successfully"), nil
		},
	}
}

// UserDelete deletes a user in Teamwork.com.
func UserDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserDelete),
			Description: "Delete an existing user in Teamwork.com. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete User",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the user to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userDeleteRequest projects.UserDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&userDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.UserDelete(ctx, engine, userDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete user")
			}
			return helpers.NewToolResultText("User deleted successfully"), nil
		},
	}
}

// UserGet retrieves a user in Teamwork.com.
func UserGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserGet),
			Description: "Get an existing user in Teamwork.com. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get User",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the user to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: userGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userGetRequest projects.UserGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&userGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			user, err := projects.UserGet(ctx, engine, userGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get user")
			}

			encoded, err := json.Marshal(user)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/people"),
						)),
					},
				},
				StructuredContent: user,
			}, nil
		},
	}
}

// UserGetMe retrieves the logged user in Teamwork.com.
func UserGetMe(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserGetMe),
			Description: "Get the logged user in Teamwork.com. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Logged User",
				ReadOnlyHint: true,
			},
			InputSchema:  &jsonschema.Schema{Type: "object"},
			OutputSchema: userGetMeOutputSchema,
		},
		Handler: func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userGetMeRequest projects.UserGetMeRequest
			user, err := projects.UserGetMe(ctx, engine, userGetMeRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get user")
			}

			encoded, err := json.Marshal(user)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/people"),
						)),
					},
				},
				StructuredContent: user,
			}, nil
		},
	}
}

// UserList lists users in Teamwork.com.
func UserList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserList),
			Description: "List users in Teamwork.com. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Users",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter users by first or last names, or e-mail. " +
							"The user will be selected if each word of the term matches the first or last name, or e-mail, not " +
							"requiring that the word matches are in the same field.",
					},
					"type": {
						Type:        "string",
						Description: "Type of user to filter by. The available options are account, collaborator or contact.",
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
			OutputSchema: userListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userListRequest projects.UserListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&userListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&userListRequest.Filters.Type, "type",
					helpers.RestrictValues("account", "collaborator", "contact"),
				),
				helpers.OptionalNumericParam(&userListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&userListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			userList, err := projects.UserList(ctx, engine, userListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list users")
			}

			encoded, err := json.Marshal(userList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/people"),
						)),
					},
				},
				StructuredContent: userList,
			}, nil
		},
	}
}

// UserListByProject lists users in Teamwork.com by project.
func UserListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserListByProject),
			Description: "List users in Teamwork.com by project. " + userDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Users By Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project from which to retrieve users.",
					},
					"search_term": {
						Type: "string",
						Description: "A search term to filter users by first or last names, or e-mail. " +
							"The user will be selected if each word of the term matches the first or last name, or e-mail, not " +
							"requiring that the word matches are in the same field.",
					},
					"type": {
						Type:        "string",
						Description: "Type of user to filter by. The available options are account, collaborator or contact.",
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
			OutputSchema: userListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userListRequest projects.UserListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&userListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&userListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&userListRequest.Filters.Type, "type",
					helpers.RestrictValues("account", "collaborator", "contact"),
				),
				helpers.OptionalNumericParam(&userListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&userListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			userList, err := projects.UserList(ctx, engine, userListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list users")
			}

			encoded, err := json.Marshal(userList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/people"),
						)),
					},
				},
				StructuredContent: userList,
			}, nil
		},
	}
}
