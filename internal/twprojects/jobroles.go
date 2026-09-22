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
	MethodJobRoleCreate    toolsets.Method = "twprojects-create_jobrole"
	MethodJobRoleUpdate    toolsets.Method = "twprojects-update_jobrole"
	MethodJobRoleDelete    toolsets.Method = "twprojects-delete_jobrole"
	MethodJobRoleGet       toolsets.Method = "twprojects-get_jobrole"
	MethodJobRoleList      toolsets.Method = "twprojects-list_jobroles"
	MethodJobRoleSetUser   toolsets.Method = "twprojects-set_user_jobrole"
	MethodJobRoleClearUser toolsets.Method = "twprojects-clear_user_jobrole"
)

var (
	jobRoleGetOutputSchema  *jsonschema.Schema
	jobRoleListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	jobRoleGetOutputSchema, err = jsonschema.For[projects.JobRoleGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for JobRoleGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(jobRoleGetOutputSchema)
	jobRoleListOutputSchema, err = jsonschema.For[projects.JobRoleListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for JobRoleListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(jobRoleListOutputSchema)
}

// jobRoleMembershipFields are the user attributes that turn a membership ID
// into a name. A full user record per member is bulk a reader of a job role
// never asked for.
var jobRoleMembershipFields = []projects.UserField{
	projects.UserFieldID,
	projects.UserFieldFirstName,
	projects.UserFieldLastName,
}

// jobRoleNeedsMembership reports whether a read has to request the users
// sideload. The endpoint leaves users and primaryUsers out of the job role
// payload entirely unless it is asked for, so membership is unreachable
// without it — including under a field selection that names either one, which
// otherwise comes back as an empty array. A selection naming neither does not
// need it.
func jobRoleNeedsMembership(fields []projects.JobRoleField) bool {
	if len(fields) == 0 {
		return true
	}
	for _, field := range fields {
		if field == projects.JobRoleFieldUsers || field == projects.JobRoleFieldPrimaryUsers {
			return true
		}
	}
	return false
}

// JobRoleCreate creates a job role in Teamwork.com.
func JobRoleCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleCreate),
			Description: "Create job role.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Job Role",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the job role.",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleCreateRequest projects.JobRoleCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&jobRoleCreateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			jobRoleResponse, err := projects.JobRoleCreate(ctx, engine, jobRoleCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create job role")
			}
			return helpers.NewToolResultText("Job role created successfully with ID %d", jobRoleResponse.JobRole.ID), nil
		},
	}
}

// JobRoleUpdate updates a job role in Teamwork.com.
func JobRoleUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleUpdate),
			Description: "Update job role.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Job Role",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the job role to update.",
					},
					"name": {
						Description: "The name of the job role.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleUpdateRequest projects.JobRoleUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&jobRoleUpdateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.JobRoleUpdate(ctx, engine, jobRoleUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update job role")
			}
			return helpers.NewToolResultText("Job role updated successfully"), nil
		},
	}
}

// JobRoleDelete deletes a job role in Teamwork.com.
func JobRoleDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleDelete),
			Description: "Delete job role.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Job Role",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the job role to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleDeleteRequest projects.JobRoleDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.JobRoleDelete(ctx, engine, jobRoleDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete job role")
			}
			return helpers.NewToolResultText("Job role deleted successfully"), nil
		},
	}
}

// JobRoleGet retrieves a job role in Teamwork.com.
func JobRoleGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodJobRoleGet),
			Description: "Get job role. The people assigned to the role come back under users, and those " +
				"holding it as their primary role under primaryUsers; both are references, resolved to " +
				"names under included.users.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Job Role",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the job role to get.",
					},
					"fields": helpers.FieldsSchema[projects.JobRole]("job role"),
				},
				Required: []string{"id"},
			},
			OutputSchema: helpers.WithOptionalFields(jobRoleGetOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleGetRequest projects.JobRoleGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleGetRequest.Path.ID, "id"),
				helpers.OptionalFieldsParam[projects.JobRole](&jobRoleGetRequest.Fields.JobRole, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if jobRoleNeedsMembership(jobRoleGetRequest.Fields.JobRole) {
				jobRoleGetRequest.Include = []projects.JobRoleRequestSideload{
					projects.JobRoleRequestSideloadUsers,
				}
				jobRoleGetRequest.Fields.Users = jobRoleMembershipFields
			}

			if len(jobRoleGetRequest.Fields.JobRole) > 0 {
				return helpers.NewRawToolResult(ctx, engine, jobRoleGetRequest, "failed to get job role", nil)
			}

			jobRole, err := projects.JobRoleGet(ctx, engine, jobRoleGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get job role")
			}

			encoded, err := json.Marshal(jobRole)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: jobRole,
			}, nil
		},
	}
}

// JobRoleSetUser sets a job role as the role of one or more users in
// Teamwork.com.
func JobRoleSetUser(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodJobRoleSetUser),
			Description: "Set the job role of one or more users. Each user holds a single job role, so this " +
				"replaces whatever role a user held before — the previous role loses them — and makes this " +
				"one their primary role. It does not add a second role alongside an existing one. To take a " +
				"user's role away without giving them another, use clear_user_jobrole.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Set User Job Role",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"job_role_id": {
						Type:        "integer",
						Description: "The ID of the job role to set as the users' role.",
					},
					"user_ids": {
						Type:        "array",
						Items:       &jsonschema.Schema{Type: "integer"},
						Description: "The IDs of the users whose job role is being set.",
					},
				},
				Required: []string{"job_role_id", "user_ids"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var assignRequest projects.UserAssignJobRoleRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&assignRequest.Path.ID, "job_role_id"),
				helpers.OptionalNumericListParam(&assignRequest.IDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if len(assignRequest.IDs) == 0 {
				return helpers.NewToolResultTextError("invalid parameters: user_ids must contain at least one user"), nil
			}

			_, err = projects.UserAssignJobRole(ctx, engine, assignRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to set user job role")
			}
			return helpers.NewToolResultText("User job role set successfully"), nil
		},
	}
}

// JobRoleClearUser removes a job role from one or more users in Teamwork.com.
func JobRoleClearUser(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleClearUser),
			Description: "Remove a job role from one or more users, leaving them with no job role.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Clear User Job Role",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"job_role_id": {
						Type:        "integer",
						Description: "The ID of the job role to remove.",
					},
					"user_ids": {
						Type:        "array",
						Items:       &jsonschema.Schema{Type: "integer"},
						Description: "The IDs of the users to remove the job role from.",
					},
				},
				Required: []string{"job_role_id", "user_ids"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var unassignRequest projects.UserUnassignJobRoleRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&unassignRequest.Path.ID, "job_role_id"),
				helpers.OptionalNumericListParam(&unassignRequest.IDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if len(unassignRequest.IDs) == 0 {
				return helpers.NewToolResultTextError("invalid parameters: user_ids must contain at least one user"), nil
			}

			_, err = projects.UserUnassignJobRole(ctx, engine, unassignRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to clear user job role")
			}
			return helpers.NewToolResultText("User job role cleared successfully"), nil
		},
	}
}

// JobRoleList lists job roles in Teamwork.com.
func JobRoleList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodJobRoleList),
			Description: "List job roles. A verbose row carries the people assigned to the role under " +
				"users, and those holding it as their primary role under primaryUsers; both are " +
				"references, resolved to names under included.users.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Job Roles",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter job roles by name, or assigned users. " +
							"The job role will be selected if each word of the term matches the name, or assigned user first or " +
							"last name, not requiring that the word matches are in the same field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"order_mode": orderModeSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"count_only": helpers.CountOnlySchema("job roles"),
					"fields":     helpers.FieldsSchema[projects.JobRole]("job role"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(jobRoleListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleListRequest projects.JobRoleListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&jobRoleListRequest.Filters.SearchTerm, "search_term"),
				orderModeParam(&jobRoleListRequest.Filters.OrderMode),
				helpers.OptionalNumericParam(&jobRoleListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&jobRoleListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.JobRole](&jobRoleListRequest.Filters.Fields.JobRoles, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, jobRoleListRequest, "failed to count job roles")
			}

			switch {
			case len(jobRoleListRequest.Filters.Fields.JobRoles) > 0:
				// An explicit selection is answered as is, except that membership
				// still has to be asked for to be part of it.

			case verbose:
				// Membership is the question a job role is usually read for.

			default:
				jobRoleListRequest.Filters.Fields.JobRoles = []projects.JobRoleField{
					projects.JobRoleFieldID,
					projects.JobRoleFieldName,
				}
			}

			if jobRoleNeedsMembership(jobRoleListRequest.Filters.Fields.JobRoles) {
				jobRoleListRequest.Filters.Include = []projects.JobRoleRequestSideload{
					projects.JobRoleRequestSideloadUsers,
				}
				jobRoleListRequest.Filters.Fields.Users = jobRoleMembershipFields
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, jobRoleListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list job roles")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list job roles"), "failed to list job roles")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(body)},
				},
			}
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
