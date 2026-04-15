package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

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
	MethodWorkflowStageCreate   toolsets.Method = "twprojects-create_workflow_stage"
	MethodWorkflowStageUpdate   toolsets.Method = "twprojects-update_workflow_stage"
	MethodWorkflowStageDelete   toolsets.Method = "twprojects-delete_workflow_stage"
	MethodWorkflowStageTaskMove toolsets.Method = "twprojects-move_task_to_workflow_stage"
	MethodWorkflowStageGet      toolsets.Method = "twprojects-get_workflow_stage"
	MethodWorkflowStageList     toolsets.Method = "twprojects-list_workflow_stages"
)

const workflowStageDescription = "A workflow stage is a single step within a workflow in Teamwork.com. " +
	"Stages are ordered and define the progression path for tasks as they move through the workflow " +
	"from start to completion. Each stage belongs to a parent workflow."

var (
	workflowStageGetOutputSchema  *jsonschema.Schema
	workflowStageListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	workflowStageGetOutputSchema, err = jsonschema.For[projects.WorkflowStageGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for WorkflowStageGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(workflowStageGetOutputSchema)
	workflowStageListOutputSchema, err = jsonschema.For[projects.WorkflowStageListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for WorkflowStageListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(workflowStageListOutputSchema)
}

// WorkflowStageCreate creates a workflow stage in Teamwork.com.
func WorkflowStageCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageCreate),
			Description: "Create a new stage within a workflow in Teamwork.com. " + workflowStageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Workflow Stage",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow to add the stage to.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the workflow stage.",
					},
				},
				Required: []string{"workflow_id", "name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageCreateRequest projects.WorkflowStageCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageCreateRequest.Path.WorkflowID, "workflow_id"),
				helpers.RequiredParam(&workflowStageCreateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			stage, err := projects.WorkflowStageCreate(ctx, engine, workflowStageCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create workflow stage")
			}
			return helpers.NewToolResultText("Workflow stage created successfully with ID %d", stage.Stage.ID), nil
		},
	}
}

// WorkflowStageUpdate updates a workflow stage in Teamwork.com.
func WorkflowStageUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageUpdate),
			Description: "Update an existing stage within a workflow in Teamwork.com. " + workflowStageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Workflow Stage",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow that owns the stage.",
					},
					"id": {
						Type:        "integer",
						Description: "The ID of the workflow stage to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name of the workflow stage.",
					},
				},
				Required: []string{"workflow_id", "id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageUpdateRequest projects.WorkflowStageUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageUpdateRequest.Path.WorkflowID, "workflow_id"),
				helpers.RequiredNumericParam(&workflowStageUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&workflowStageUpdateRequest.Name, "name"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.WorkflowStageUpdate(ctx, engine, workflowStageUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update workflow stage")
			}
			return helpers.NewToolResultText("Workflow stage updated successfully"), nil
		},
	}
}

// WorkflowStageDelete deletes a workflow stage in Teamwork.com.
func WorkflowStageDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageDelete),
			Description: "Delete an existing stage within a workflow in Teamwork.com. " + workflowStageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Workflow Stage",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow that owns the stage.",
					},
					"id": {
						Type:        "integer",
						Description: "The ID of the workflow stage to delete.",
					},
					"map_tasks_to_stage_id": {
						Type: "integer",
						Description: "The ID of another stage to which tasks in the deleted stage will be moved. " +
							"If not provided, tasks will be moved back to the backlog.",
					},
				},
				Required: []string{"workflow_id", "id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageDeleteRequest projects.WorkflowStageDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageDeleteRequest.Path.WorkflowID, "workflow_id"),
				helpers.RequiredNumericParam(&workflowStageDeleteRequest.Path.ID, "id"),
				helpers.OptionalNumericParam(&workflowStageDeleteRequest.MapTasksToStageID, "map_tasks_to_stage_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.WorkflowStageDelete(ctx, engine, workflowStageDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete workflow stage")
			}
			return helpers.NewToolResultText("Workflow stage deleted successfully"), nil
		},
	}
}

// WorkflowStageTaskMove moves a task to a specific stage within a workflow in
// Teamwork.com.
func WorkflowStageTaskMove(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodWorkflowStageTaskMove),
			Description: "Move a task to a specific stage within a workflow in Teamwork.com. " +
				workflowStageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Move Task to Workflow Stage",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow that contains the target stage.",
					},
					"stage_id": {
						Type:        "integer",
						Description: "The ID of the workflow stage to move the task to.",
					},
					"task_id": {
						Type:        "integer",
						Description: "The ID of the task to move.",
					},
				},
				Required: []string{"workflow_id", "stage_id", "task_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageTaskMoveRequest projects.WorkflowStageTaskMoveRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageTaskMoveRequest.Path.WorkflowID, "workflow_id"),
				helpers.RequiredNumericParam(&workflowStageTaskMoveRequest.StageID, "stage_id"),
				helpers.RequiredNumericParam(&workflowStageTaskMoveRequest.Path.TaskID, "task_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.WorkflowStageTaskMove(ctx, engine, workflowStageTaskMoveRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to move task to workflow stage")
			}
			return helpers.NewToolResultText("Task moved to workflow stage successfully"), nil
		},
	}
}

// WorkflowStageGet retrieves a workflow stage in Teamwork.com.
func WorkflowStageGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageGet),
			Description: "Get an existing stage within a workflow in Teamwork.com. " + workflowStageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Workflow Stage",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow that owns the stage.",
					},
					"id": {
						Type:        "integer",
						Description: "The ID of the workflow stage to get.",
					},
				},
				Required: []string{"workflow_id", "id"},
			},
			OutputSchema: workflowStageGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageGetRequest projects.WorkflowStageGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageGetRequest.Path.WorkflowID, "workflow_id"),
				helpers.RequiredNumericParam(&workflowStageGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			stage, err := projects.WorkflowStageGet(ctx, engine, workflowStageGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get workflow stage")
			}

			encoded, err := json.Marshal(stage)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: stage,
			}, nil
		},
	}
}

// WorkflowStageList lists workflow stages in Teamwork.com.
func WorkflowStageList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageList),
			Description: "List stages within a workflow in Teamwork.com. " + workflowStageDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Workflow Stages",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow whose stages to list.",
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
				Required: []string{"workflow_id"},
			},
			OutputSchema: workflowStageListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageListRequest projects.WorkflowStageListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageListRequest.Path.WorkflowID, "workflow_id"),
				helpers.OptionalNumericParam(&workflowStageListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&workflowStageListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			stageList, err := projects.WorkflowStageList(ctx, engine, workflowStageListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list workflow stages")
			}

			encoded, err := json.Marshal(stageList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: stageList,
			}, nil
		},
	}
}
