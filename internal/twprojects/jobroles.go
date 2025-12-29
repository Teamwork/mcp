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
	MethodJobRoleCreate toolsets.Method = "twprojects-create_jobrole"
	MethodJobRoleUpdate toolsets.Method = "twprojects-update_jobrole"
	MethodJobRoleDelete toolsets.Method = "twprojects-delete_jobrole"
	MethodJobRoleGet    toolsets.Method = "twprojects-get_jobrole"
	MethodJobRoleList   toolsets.Method = "twprojects-list_jobroles"
)

const jobRoleDescription = "Job role defines a user's primary function or position within the organization, such as " +
	"developer, designer, project manager, or account manager. It provides high-level context about what a person is " +
	"generally responsible for, helping teams understand who does what across projects and departments. Job roles are " +
	"commonly used in resource planning, capacity forecasting, and reporting, allowing managers to group work by role, " +
	"plan future demand more accurately, and ensure the right mix of roles is available to deliver projects efficiently."

var (
	jobRoleGetOutputSchema  *jsonschema.Schema
	jobRoleListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodJobRoleCreate)
	toolsets.RegisterMethod(MethodJobRoleUpdate)
	toolsets.RegisterMethod(MethodJobRoleDelete)
	toolsets.RegisterMethod(MethodJobRoleGet)
	toolsets.RegisterMethod(MethodJobRoleList)

	var err error

	// generate the output schemas only once
	jobRoleGetOutputSchema, err = jsonschema.For[projects.JobRoleGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for JobRoleGetResponse: %v", err))
	}
	jobRoleListOutputSchema, err = jsonschema.For[projects.JobRoleListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for JobRoleListResponse: %v", err))
	}
}

// JobRoleCreate creates a job role in Teamwork.com.
func JobRoleCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodJobRoleCreate),
			Description: "Create a new job role in Teamwork.com. " + jobRoleDescription,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&jobRoleCreateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Update an existing job role in Teamwork.com. " + jobRoleDescription,
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
						Type:        "string",
						Description: "The name of the job role.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleUpdateRequest projects.JobRoleUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&jobRoleUpdateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Delete an existing job role in Teamwork.com. " + jobRoleDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Job Role",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Get an existing job role in Teamwork.com. " + jobRoleDescription,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&jobRoleGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "List job roles in Teamwork.com. " + jobRoleDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Job Roles",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter job roles by name, or assigned users. " +
							"The job role will be selected if each word of the term matches the name, or assigned user first or " +
							"last name, not requiring that the word matches are in the same field.",
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
			OutputSchema: jobRoleListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var jobRoleListRequest projects.JobRoleListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&jobRoleListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&jobRoleListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&jobRoleListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			jobRoleList, err := projects.JobRoleList(ctx, engine, jobRoleListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list job roles")
			}

			encoded, err := json.Marshal(jobRoleList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: jobRoleList,
			}, nil
		},
	}
}
