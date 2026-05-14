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
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodWorkflowCreate      toolsets.Method = "twprojects-create_workflow"
	MethodWorkflowUpdate      toolsets.Method = "twprojects-update_workflow"
	MethodWorkflowDelete      toolsets.Method = "twprojects-delete_workflow"
	MethodWorkflowProjectLink toolsets.Method = "twprojects-link_project_to_workflow"
	MethodWorkflowGet         toolsets.Method = "twprojects-get_workflow"
	MethodWorkflowList        toolsets.Method = "twprojects-list_workflows"
)

var (
	workflowGetOutputSchema  *jsonschema.Schema
	workflowListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	workflowGetOutputSchema, err = jsonschema.For[projects.WorkflowGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for WorkflowGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(workflowGetOutputSchema)
	workflowListOutputSchema, err = jsonschema.For[projects.WorkflowListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for WorkflowListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(workflowListOutputSchema)
}

// WorkflowCreate creates a workflow in Teamwork.com.
func WorkflowCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowCreate),
			Description: "Create workflow.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Workflow",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the workflow.",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowCreateRequest projects.WorkflowCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&workflowCreateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			workflow, err := projects.WorkflowCreate(ctx, engine, workflowCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create workflow")
			}
			return helpers.NewToolResultText("Workflow created successfully with ID %d", workflow.Workflow.ID), nil
		},
	}
}

// WorkflowUpdate updates a workflow in Teamwork.com.
func WorkflowUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowUpdate),
			Description: "Update workflow.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Workflow",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the workflow to update.",
					},
					"name": {
						Description: "The new name of the workflow.",
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
			var workflowUpdateRequest projects.WorkflowUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&workflowUpdateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.WorkflowUpdate(ctx, engine, workflowUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update workflow")
			}
			return helpers.NewToolResultText("Workflow updated successfully"), nil
		},
	}
}

// WorkflowDelete deletes a workflow in Teamwork.com.
func WorkflowDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowDelete),
			Description: "Delete workflow.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Workflow",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the workflow to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowDeleteRequest projects.WorkflowDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.WorkflowDelete(ctx, engine, workflowDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete workflow")
			}
			return helpers.NewToolResultText("Workflow deleted successfully"), nil
		},
	}
}

// WorkflowProjectLink links a project to a workflow in Teamwork.com.
func WorkflowProjectLink(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowProjectLink),
			Description: "Link a project to a workflow so its tasks track through workflow stages.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Link Project to Workflow",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to link to the workflow.",
					},
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow to link the project to.",
					},
				},
				Required: []string{"project_id", "workflow_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowProjectLinkRequest projects.WorkflowProjectLinkRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowProjectLinkRequest.Path.ProjectID, "project_id"),
				helpers.RequiredNumericParam(&workflowProjectLinkRequest.WorkflowID, "workflow_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.WorkflowProjectLink(ctx, engine, workflowProjectLinkRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to link project to workflow")
			}
			return helpers.NewToolResultText("Project linked to workflow successfully"), nil
		},
	}
}

// WorkflowGet retrieves a workflow in Teamwork.com.
func WorkflowGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowGet),
			Description: "Get workflow.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Workflow",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the workflow to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: workflowGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowGetRequest projects.WorkflowGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			workflow, err := projects.WorkflowGet(ctx, engine, workflowGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get workflow")
			}

			encoded, err := json.Marshal(workflow)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: workflow,
			}, nil
		},
	}
}

// WorkflowList lists workflows in Teamwork.com.
func WorkflowList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowList),
			Description: "List workflows.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Workflows",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": helpers.SearchTermSchema("workflows", "name"),
					"page":        helpers.PageSchema(),
					"page_size":   helpers.PageSizeSchema(),
					"verbose":     helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(workflowListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowListRequest projects.WorkflowListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&workflowListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&workflowListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&workflowListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				workflowListRequest.Filters.Fields.Workflows = []projects.WorkflowField{
					projects.WorkflowFieldID,
					projects.WorkflowFieldName,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, workflowListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list workflows")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list workflows"), "failed to list workflows")
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
