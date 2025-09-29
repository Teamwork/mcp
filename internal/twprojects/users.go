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

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodUserCreate)
	toolsets.RegisterMethod(MethodUserUpdate)
	toolsets.RegisterMethod(MethodUserDelete)
	toolsets.RegisterMethod(MethodUserGet)
	toolsets.RegisterMethod(MethodUserGetMe)
	toolsets.RegisterMethod(MethodUserList)
	toolsets.RegisterMethod(MethodUserListByProject)
}

// UserCreate creates a user in Teamwork.com.
func UserCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserCreate),
			mcp.WithDescription("Create a new user in Teamwork.com. "+userDescription),
			mcp.WithTitleAnnotation("Create User"),
			mcp.WithString("first_name",
				mcp.Required(),
				mcp.Description("The first name of the user."),
			),
			mcp.WithString("last_name",
				mcp.Required(),
				mcp.Description("The last name of the user."),
			),
			mcp.WithString("title",
				mcp.Description("The job title of the user, such as 'Project Manager' or 'Senior Software Developer'."),
			),
			mcp.WithString("email",
				mcp.Required(),
				mcp.Description("The email address of the user."),
			),
			mcp.WithBoolean("admin",
				mcp.Description("Indicates whether the user is an administrator."),
			),
			mcp.WithString("type",
				mcp.Description("The type of user, such as 'account', 'collaborator', or 'contact'."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the client/company to which the user belongs."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userCreateRequest projects.UserCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
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
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			user, err := projects.UserCreate(ctx, engine, userCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create user")
			}

			return mcp.NewToolResultText(fmt.Sprintf("User created successfully with ID %d", user.ID)), nil
		},
	}
}

// UserUpdate updates a user in Teamwork.com.
func UserUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserUpdate),
			mcp.WithDescription("Update an existing user in Teamwork.com. "+userDescription),
			mcp.WithTitleAnnotation("Update User"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the user to update."),
			),
			mcp.WithString("first_name",
				mcp.Description("The first name of the user."),
			),
			mcp.WithString("last_name",
				mcp.Description("The last name of the user."),
			),
			mcp.WithString("title",
				mcp.Description("The job title of the user, such as 'Project Manager' or 'Senior Software Developer'."),
			),
			mcp.WithString("email",
				mcp.Description("The email address of the user."),
			),
			mcp.WithBoolean("admin",
				mcp.Description("Indicates whether the user is an administrator."),
			),
			mcp.WithString("type",
				mcp.Description("The type of user, such as 'account', 'collaborator', or 'contact'."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the client/company to which the user belongs."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userUpdateRequest projects.UserUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
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
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.UserUpdate(ctx, engine, userUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update user")
			}

			return mcp.NewToolResultText("User updated successfully"), nil
		},
	}
}

// UserDelete deletes a user in Teamwork.com.
func UserDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserDelete),
			mcp.WithDescription("Delete an existing user in Teamwork.com. "+userDescription),
			mcp.WithTitleAnnotation("Delete User"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the user to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userDeleteRequest projects.UserDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&userDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.UserDelete(ctx, engine, userDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete user")
			}

			return mcp.NewToolResultText("User deleted successfully"), nil
		},
	}
}

// UserGet retrieves a user in Teamwork.com.
func UserGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserGet),
			mcp.WithDescription("Get an existing user in Teamwork.com. "+userDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get User"),
			mcp.WithOutputSchema[projects.UserGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the user to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userGetRequest projects.UserGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&userGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			user, err := projects.UserGet(ctx, engine, userGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get user")
			}

			encoded, err := json.Marshal(user)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}

// UserGetMe retrieves the logged user in Teamwork.com.
func UserGetMe(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserGetMe),
			mcp.WithDescription("Get the logged user in Teamwork.com. "+userDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Logged User"),
			mcp.WithOutputSchema[projects.UserGetMeResponse](),
		),
		Handler: func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userGetMeRequest projects.UserGetMeRequest
			user, err := projects.UserGetMe(ctx, engine, userGetMeRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get user")
			}

			encoded, err := json.Marshal(user)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}

// UserList lists users in Teamwork.com.
func UserList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserList),
			mcp.WithDescription("List users in Teamwork.com. "+userDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Users"),
			mcp.WithOutputSchema[projects.UserListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter users by first or last names, or e-mail. "+
					"The user will be selected if each word of the term matches the first or last name, or e-mail, not "+
					"requiring that the word matches are in the same field."),
			),
			mcp.WithNumber("type",
				mcp.Description("Type of user to filter by. The available options are account, collaborator or contact."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userListRequest projects.UserListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&userListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&userListRequest.Filters.Type, "type",
					helpers.RestrictValues("account", "collaborator", "contact"),
				),
				helpers.OptionalNumericParam(&userListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&userListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			userList, err := projects.UserList(ctx, engine, userListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list users")
			}

			encoded, err := json.Marshal(userList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}

// UserListByProject lists users in Teamwork.com by project.
func UserListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodUserListByProject),
			mcp.WithDescription("List users in Teamwork.com by project. "+userDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Users By Project"),
			mcp.WithOutputSchema[projects.UserListResponse](),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project from which to retrieve users."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter users by first or last names, or e-mail. "+
					"The user will be selected if each word of the term matches the first or last name, or e-mail, not "+
					"requiring that the word matches are in the same field."),
			),
			mcp.WithNumber("type",
				mcp.Description("Type of user to filter by. The available options are account, collaborator or contact."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userListRequest projects.UserListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&userListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&userListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&userListRequest.Filters.Type, "type",
					helpers.RestrictValues("account", "collaborator", "contact"),
				),
				helpers.OptionalNumericParam(&userListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&userListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			userList, err := projects.UserList(ctx, engine, userListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list users")
			}

			encoded, err := json.Marshal(userList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/people"),
			))), nil
		},
	}
}
