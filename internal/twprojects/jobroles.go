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
	MethodJobRoleCreate toolsets.Method = "twprojects-create_jobrole"
	MethodJobRoleUpdate toolsets.Method = "twprojects-update_jobrole"
	MethodJobRoleDelete toolsets.Method = "twprojects-delete_jobrole"
	MethodJobRoleGet    toolsets.Method = "twprojects-get_jobrole"
	MethodJobRoleList   toolsets.Method = "twprojects-list_jobroles"
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

// JobRoleCreate creates a job role in Teamwork.com.
func JobRoleCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleCreate),
			Description: "Create job role.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Job Role",
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
				Title: "Update Job Role",
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
			Name:        string(MethodJobRoleGet),
			Description: "Get job role.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Job Role",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the job role to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: jobRoleGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleGetRequest projects.JobRoleGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
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

// JobRoleList lists job roles in Teamwork.com.
func JobRoleList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleList),
			Description: "List job roles.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Job Roles",
				ReadOnlyHint: true,
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
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(jobRoleListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleListRequest projects.JobRoleListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&jobRoleListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&jobRoleListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&jobRoleListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				jobRoleListRequest.Filters.Fields.JobRoles = []projects.JobRoleField{
					projects.JobRoleFieldID,
					projects.JobRoleFieldName,
				}
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
