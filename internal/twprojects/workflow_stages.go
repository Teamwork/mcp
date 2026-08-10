package twprojects

import (
	"bytes"
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
	MethodWorkflowStageCreate   toolsets.Method = "twprojects-create_workflow_stage"
	MethodWorkflowStageUpdate   toolsets.Method = "twprojects-update_workflow_stage"
	MethodWorkflowStageDelete   toolsets.Method = "twprojects-delete_workflow_stage"
	MethodWorkflowStageTaskMove toolsets.Method = "twprojects-move_task_to_workflow_stage"
	MethodWorkflowStageGet      toolsets.Method = "twprojects-get_workflow_stage"
	MethodWorkflowStageList     toolsets.Method = "twprojects-list_workflow_stages"
)

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
			Description: "Create workflow stage.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Workflow Stage",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
			Description: "Update workflow stage.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Workflow Stage",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Description: "The new name of the workflow stage.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
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
			Description: "Delete workflow stage.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Workflow Stage",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
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
						Description: "The ID of another stage to which tasks in the deleted stage will be moved. " +
							"If not provided, tasks will be moved back to the backlog.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
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

// workflowStageTasksMoveRequest moves one or more tasks into a workflow stage.
//
// twapi-go-sdk only models the single-task PATCH
// /tasks/{taskId}/workflows/{workflowId}.json, which carries the task in the
// path and so costs one round trip per task. The stage's own tasks endpoint
// takes a taskIds array and moves the whole set in one call, so it is built
// here rather than looped over there.
//
// https://apidocs.teamwork.com/guides/teamwork/workflows-api-getting-started-guide
type workflowStageTasksMoveRequest struct {
	WorkflowID int64
	StageID    int64
	TaskIDs    []int64
}

// HTTPRequest builds the POST
// /projects/api/v3/workflows/{workflowId}/stages/{stageId}/tasks.json request.
func (w workflowStageTasksMoveRequest) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	uri := fmt.Sprintf("%s/projects/api/v3/workflows/%d/stages/%d/tasks.json", server, w.WorkflowID, w.StageID)

	payload := struct {
		TaskIDs []int64 `json:"taskIds"`
	}{TaskIDs: w.TaskIDs}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return nil, fmt.Errorf("failed to encode move tasks to workflow stage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// workflowStageTasksMoveResponse carries no payload; the endpoint answers with
// the outcome in its status code alone.
type workflowStageTasksMoveResponse struct{}

// HandleHTTPResponse accepts any 2xx rather than pinning one code. The endpoint
// is documented without a response body, and its single-task sibling answers
// 204 while a create-shaped POST may answer 200 or 201.
func (*workflowStageTasksMoveResponse) HandleHTTPResponse(resp *http.Response) error {
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return twapi.NewHTTPError(resp, "failed to move tasks to workflow stage")
	}
	return nil
}

// WorkflowStageTaskMove moves tasks to a specific stage within a workflow in
// Teamwork.com.
func WorkflowStageTaskMove(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageTaskMove),
			Description: "Move one or more tasks to a workflow stage.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Move Tasks to Workflow Stage",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
						Description: "The ID of the workflow stage to move the tasks to.",
					},
					"task_ids": {
						Type:  "array",
						Items: &jsonschema.Schema{Type: "integer"},
						Description: "The IDs of the tasks to move. At least one is needed; all of them " +
							"move in a single call.",
					},
				},
				// task_ids is deliberately absent from Required. The SDK validates
				// the schema before the handler runs, so requiring it would reject
				// clients still sending the scalar task_id this tool advertised
				// before it could move a set, and the handler could never see them.
				Required: []string{"workflow_id", "stage_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var moveRequest workflowStageTasksMoveRequest
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&moveRequest.WorkflowID, "workflow_id"),
				helpers.RequiredNumericParam(&moveRequest.StageID, "stage_id"),
				helpers.OptionalNumericListParam(&moveRequest.TaskIDs, "task_ids"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// This tool advertised a scalar task_id before it could move a set.
			// Clients holding a cached tool list still send it, so it is accepted
			// but no longer advertised, to keep one way of saying this in the
			// schema.
			if len(moveRequest.TaskIDs) == 0 {
				var legacyTaskID int64
				if err := helpers.ParamGroup(arguments,
					helpers.OptionalNumericParam(&legacyTaskID, "task_id"),
				); err != nil {
					return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
				}
				if legacyTaskID > 0 {
					moveRequest.TaskIDs = []int64{legacyTaskID}
				}
			}
			if len(moveRequest.TaskIDs) == 0 {
				return helpers.NewToolResultTextError("task_ids must contain at least one task ID"), nil
			}

			_, err := twapi.Execute[workflowStageTasksMoveRequest, *workflowStageTasksMoveResponse](
				ctx, engine, moveRequest,
			)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to move tasks to workflow stage")
			}
			if len(moveRequest.TaskIDs) == 1 {
				return helpers.NewToolResultText("Task moved to workflow stage successfully"), nil
			}
			return helpers.NewToolResultText("%d tasks moved to workflow stage successfully",
				len(moveRequest.TaskIDs)), nil
		},
	}
}

// WorkflowStageGet retrieves a workflow stage in Teamwork.com.
func WorkflowStageGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodWorkflowStageGet),
			Description: "Get workflow stage.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Workflow Stage",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
					"fields": helpers.FieldsSchema("workflow stage"),
				},
				Required: []string{"workflow_id", "id"},
			},
			OutputSchema: helpers.WithOptionalFields(workflowStageGetOutputSchema),
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
				helpers.OptionalFieldsParam[projects.WorkflowStage](&workflowStageGetRequest.Fields.Stage, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if len(workflowStageGetRequest.Fields.Stage) > 0 {
				return helpers.NewRawToolResult(ctx, engine, workflowStageGetRequest, "failed to get workflow stage", nil)
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
			Description: "List workflow stages.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Workflow Stages",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"workflow_id": {
						Type:        "integer",
						Description: "The ID of the workflow whose stages to list.",
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
					"fields":    helpers.FieldsSchema("workflow stage"),
				},
				Required: []string{"workflow_id"},
			},
			OutputSchema: helpers.WithOptionalFields(workflowStageListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var workflowStageListRequest projects.WorkflowStageListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&workflowStageListRequest.Path.WorkflowID, "workflow_id"),
				helpers.OptionalNumericParam(&workflowStageListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&workflowStageListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalFieldsParam[projects.WorkflowStage](&workflowStageListRequest.Filters.Fields.Stages, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose && len(workflowStageListRequest.Filters.Fields.Stages) == 0 {
				workflowStageListRequest.Filters.Fields.Stages = []projects.WorkflowStageField{
					projects.WorkflowStageFieldID,
					projects.WorkflowStageFieldName,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, workflowStageListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list workflow stages")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list workflow stages"),
					"failed to list workflow stages",
				)
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
