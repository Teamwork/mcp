package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	MethodUserCreate      toolsets.Method = "twprojects-create_user"
	MethodUserCreateBatch toolsets.Method = "twprojects-create_users"
	MethodUserUpdate      toolsets.Method = "twprojects-update_user"
	MethodUserDelete      toolsets.Method = "twprojects-delete_user"
	MethodUserGet         toolsets.Method = "twprojects-get_user"
	MethodUserGetMe       toolsets.Method = "twprojects-get_user_me"
	MethodUserList        toolsets.Method = "twprojects-list_users"
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
				Title: "Create User",
				// The create emails an invite to the new user's address by default.
				DestructiveHint: new(false),
				OpenWorldHint:   new(true),
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

// userCreateBatchMaxUsers bounds create_users. The endpoint creates the users
// one after another inside the one request, so a long list risks outliving the
// tool call.
const userCreateBatchMaxUsers = 50

// UserCreateBatch creates many users in Teamwork.com with a single request to
// the bulk people endpoint, instead of one request per user.
func UserCreateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	itemSchema := batchItemSchema(UserCreate(engine).Tool.InputSchema)
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodUserCreateBatch),
			Description: fmt.Sprintf("Create many users in one call (at most %d per call), in one request. Use "+
				"instead of calling twprojects-create_user in a loop. No invitation email is sent unless "+
				"send_invite is true; invitations go to real people and cannot be recalled, so ask the user "+
				"before setting it. Items are validated first and nothing is created if one is invalid; after "+
				"that each user succeeds or fails on its own, and the result lists the new user ID of each "+
				"item and the reason for each failure, by item number counting from 1.", userCreateBatchMaxUsers),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Users",
				// Invitations are off by default, but the caller can turn them on,
				// and they email any address. Like create_user, adding people and
				// inviting them is not destructive.
				DestructiveHint: new(false),
				OpenWorldHint:   new(true),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"users": {
						Type:     "array",
						Items:    itemSchema,
						MinItems: new(1),
						MaxItems: new(userCreateBatchMaxUsers),
						Description: fmt.Sprintf("The users to create, at most %d, each with the parameters of "+
							"twprojects-create_user.", userCreateBatchMaxUsers),
					},
					"send_invite": {
						Type: "boolean",
						Description: "Email every created user an invitation to the site. Defaults to false. " +
							"Only set it to true after the user has confirmed that invitations should be sent.",
						Default: json.RawMessage("false"),
					},
				},
				Required: []string{"users"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			var sendInvite bool
			if err := helpers.ParamGroup(arguments, helpers.OptionalParam(&sendInvite, "send_invite")); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "users", userCreateBatchMaxUsers)
			if toolResult != nil {
				return toolResult, nil
			}

			users := make([]projects.UserBulkCreateRequestUser, len(items))
			labels := make([]string, len(items))
			var invalid []string
			for i, item := range items {
				user, err := parseUserBulkCreateItem(item)
				labels[i] = fmt.Sprintf("item %d %q", i+1, user.Email)
				if err != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, err.Error()))
					continue
				}
				user.SendInvite = sendInvite
				users[i] = user
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			response, err := projects.UserBulkCreate(ctx, engine, projects.NewUserBulkCreateRequest(users))
			if err != nil {
				if failures := projects.UserBulkCreateFailures(err); len(failures) > 0 {
					return userBulkCreateNothingCreated(labels, failures), nil
				}
				return helpers.HandleAPIError(err, "failed to create users")
			}
			return userBulkCreateReport(labels, response), nil
		},
	}
}

// parseUserBulkCreateItem binds one item of create_users with the same
// parameters and checks as create_user.
func parseUserBulkCreateItem(item map[string]any) (projects.UserBulkCreateRequestUser, error) {
	var user projects.UserBulkCreateRequestUser
	var userType *string
	err := helpers.ParamGroup(item,
		helpers.RequiredParam(&user.Email, "email"),
		helpers.RequiredParam(&user.FirstName, "first_name"),
		helpers.RequiredParam(&user.LastName, "last_name"),
		helpers.OptionalPointerParam(&user.Title, "title"),
		helpers.OptionalPointerParam(&user.Admin, "admin"),
		helpers.OptionalPointerParam(&userType, "type",
			helpers.RestrictValues("account", "collaborator", "contact"),
		),
		helpers.OptionalNumericPointerParam(&user.CompanyID, "company_id"),
	)
	if userType != nil {
		user.Type = new(projects.UserType(*userType))
	}
	return user, err
}

// userBulkCreateReport maps the created users back to the items they came from.
// The endpoint returns the created users in request order, leaving out the
// items it reports as failed.
func userBulkCreateReport(labels []string, response *projects.UserBulkCreateResponse) *mcp.CallToolResult {
	errs := make([]error, len(labels))
	for _, failure := range response.Failed {
		if failure.Index >= 0 && failure.Index < len(errs) {
			errs[failure.Index] = errors.New(userBulkCreateReasons(failure))
		}
	}

	details := make([]string, len(labels))
	next := 0
	for i := range errs {
		if errs[i] != nil {
			continue
		}
		if next >= len(response.Users) {
			errs[i] = errors.New("not reported as created or failed; check with " +
				string(MethodUserList) + " before sending it again")
			continue
		}
		details[i] = fmt.Sprintf("user %d", response.Users[next].ID)
		next++
	}
	return batchReport("Created", "users",
		"check with "+string(MethodUserList)+" before sending them again", labels, details, errs)
}

// userBulkCreateNothingCreated reports a request the endpoint rejected as a
// whole, naming each item that caused it.
func userBulkCreateNothingCreated(labels []string, failures []projects.UserBulkCreateFailure) *mcp.CallToolResult {
	var report strings.Builder
	fmt.Fprintf(&report, "Created 0 of %d users. Nothing was created.", len(labels))
	for _, failure := range failures {
		label := fmt.Sprintf("item %d", failure.Index+1)
		if failure.Index >= 0 && failure.Index < len(labels) {
			label = labels[failure.Index]
		}
		fmt.Fprintf(&report, "\nFailed: %s: %s.", label, userBulkCreateReasons(failure))
	}
	return helpers.NewToolResultTextError("%s", report.String())
}

// userBulkCreateReasons joins the reasons one item was not created.
func userBulkCreateReasons(failure projects.UserBulkCreateFailure) string {
	reasons := make([]string, 0, len(failure.Errors))
	for _, e := range failure.Errors {
		reason := e.Detail
		if reason == "" {
			reason = e.Title
		}
		reasons = append(reasons, reason)
	}
	if len(reasons) == 0 {
		return "no reason given"
	}
	return strings.Join(reasons, "; ")
}

// UserUpdate updates a user in Teamwork.com.
func UserUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodUserUpdate),
			Description: "Update user.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update User",
				DestructiveHint: new(true),
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
			// Optional fields: a restricted installation's result has no email or rates.
			OutputSchema: helpers.WithOptionalFields(userGetMeOutputSchema),
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
