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
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodUserCreate toolsets.Method = "twprojects-create_user"
	MethodUserUpdate toolsets.Method = "twprojects-update_user"
	MethodUserDelete toolsets.Method = "twprojects-delete_user"
	MethodUserGet    toolsets.Method = "twprojects-get_user"
	MethodUserGetMe  toolsets.Method = "twprojects-get_user_me"
	MethodUserList   toolsets.Method = "twprojects-list_users"
)

var (
	userGetOutputSchema   *jsonschema.Schema
	userGetMeOutputSchema *jsonschema.Schema
	userListOutputSchema  *jsonschema.Schema
)

// userOrdering is the order-by vocabulary of the users list endpoint.
var userOrdering = newOrdering("users",
	projects.UserOrderByName,
	projects.UserOrderByNameCaseInsensitive,
	projects.UserOrderByCompany,
	projects.UserOrderByID,
)

func init() {
	var err error

	// generate the output schemas only once
	userGetOutputSchema, err = jsonschema.For[projects.UserGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(userGetOutputSchema)
	userGetMeOutputSchema, err = jsonschema.For[projects.UserGetMeResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserGetMeResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(userGetMeOutputSchema)
	userListOutputSchema, err = jsonschema.For[projects.UserListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for UserListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(userListOutputSchema)
}

// UserCreate creates a user in Teamwork.com.
func UserCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserCreate),
			Description: "Create user.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create User",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Description: "The job title of the user, such as 'Project Manager' or 'Senior Software Developer'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email": {
						Type:        "string",
						Description: "The email address of the user.",
					},
					"admin": {
						Description: "Indicates whether the user is an administrator.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"type": {
						Description: "The type of user, such as 'account', 'collaborator', or 'contact'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"company_id": {
						Description: "The ID of the client/company to which the user belongs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"first_name", "last_name", "email"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userCreateRequest projects.UserCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "Update user.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update User",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the user to update.",
					},
					"first_name": {
						Description: "The first name of the user.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"last_name": {
						Description: "The last name of the user.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"title": {
						Description: "The job title of the user, such as 'Project Manager' or 'Senior Software Developer'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"email": {
						Description: "The email address of the user.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"admin": {
						Description: "Indicates whether the user is an administrator.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"type": {
						Description: "The type of user, such as 'account', 'collaborator', or 'contact'.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"company_id": {
						Description: "The ID of the client/company to which the user belongs.",
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
			var userUpdateRequest projects.UserUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "Delete user.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete User",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
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
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&userDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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
			Description: "Get user.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get User",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the user to get.",
					},
					"fields": helpers.FieldsSchema[projects.User]("user"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(userGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userGetRequest projects.UserGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&userGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.User](&userGetRequest.Fields.User, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(userGetRequest.Fields.User) > 0 {
				return helpers.NewRawToolResult(ctx, engine, userGetRequest, "failed to get user",
					helpers.WebLinkerWithIDPathBuilder("/app/people"),
				)
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
				StructuredContent: helpers.StructuredWebLinker(ctx, user,
					helpers.WebLinkerWithIDPathBuilder("/app/people"),
				),
			}, nil
		},
	}
}

// UserGetMe retrieves the logged user in Teamwork.com.
func UserGetMe(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserGetMe),
			Description: "Get the currently authenticated user.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Logged User",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
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
				StructuredContent: helpers.StructuredWebLinker(ctx, user,
					helpers.WebLinkerWithIDPathBuilder("/app/people"),
				),
			}, nil
		},
	}
}

// UserList lists users in Teamwork.com.
func UserList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserList),
			Description: "List users. Scope by project_id or filter by type (account/collaborator/contact).",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Users",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Description: "The ID of the project from which to retrieve users. Omit to list users across all projects.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"search_term": {
						Description: "A search term to filter users by first or last names, or e-mail. " +
							"The user will be selected if each word of the term matches the first or last name, or e-mail, not " +
							"requiring that the word matches are in the same field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"type": {
						Description: "Type of user to filter by. The available options are account, collaborator or contact.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"order_by":   userOrdering.orderBySchema(),
					"order_mode": orderModeSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"count_only": helpers.CountOnlySchema("users"),
					"fields":     helpers.FieldsSchema[projects.User]("user"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(userListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var userListRequest projects.UserListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&userListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&userListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&userListRequest.Filters.Type, "type",
					helpers.RestrictValues(
						projects.UserTypeAccount,
						projects.UserTypeCollaborator,
						projects.UserTypeContact,
					),
				),
				userOrdering.param(&userListRequest.Filters.OrderBy, &userListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&userListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&userListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.User](&userListRequest.Filters.Fields.Users, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose && len(userListRequest.Filters.Fields.Users) == 0 {
				userListRequest.Filters.Fields.Users = []projects.UserField{
					projects.UserFieldID,
					projects.UserFieldFirstName,
					projects.UserFieldLastName,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, userListRequest, "failed to count users")
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, userListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list users")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list users"), "failed to list users")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/people"))
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
