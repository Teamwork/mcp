package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

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
	MethodProjectMemberAdd      toolsets.Method = "twprojects-add_project_member"
	MethodProjectMemberAddBatch toolsets.Method = "twprojects-add_project_members"
)

// ProjectMemberAdd adds a user to a project in Teamwork.com.
func ProjectMemberAdd(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectMemberAdd),
			Description: "Add users to a project. For several projects, use twprojects-add_project_members.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Add Project Member",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to add the member to.",
					},
					"user_ids": {
						Description: "Users to add.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
				},
				Required: []string{"project_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectMemberAddRequest projects.ProjectMemberAddRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectMemberAddRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericListParam(&projectMemberAddRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.ProjectMemberAdd(ctx, engine, projectMemberAddRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to add project member")
			}
			return helpers.NewToolResultText("Project member added successfully"), nil
		},
	}
}

// ProjectMemberAddBatch adds users to many projects in one call, one request
// per project.
func ProjectMemberAddBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodProjectMemberAddBatch),
			Description: "Add users to many projects in one call, each project with its own users. Use instead " +
				"of calling twprojects-add_project_member in a loop. Items are validated first and nothing is " +
				"added if one is invalid; after that each project succeeds or fails on its own, and the result " +
				"lists the error of each failure. Adding people needs project admin rights on that project.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Add Project Members",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"projects": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"project_id": {
									Type:        "integer",
									Description: "The ID of the project to add the users to.",
								},
								"user_ids": {
									Type:        "array",
									Items:       &jsonschema.Schema{Type: "integer"},
									MinItems:    new(1),
									Description: "The users to add.",
								},
							},
							Required: []string{"project_id", "user_ids"},
						},
						MinItems: new(1),
						MaxItems: new(batchMaxItems),
						Description: "One item per project. A project may appear only once; list all of its " +
							"users in that item.",
					},
				},
				Required: []string{"projects"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "projects", batchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			requests := make([]projects.ProjectMemberAddRequest, len(items))
			labels := make([]string, len(items))
			seen := make(map[int64]bool, len(items))
			var invalid []string
			for i, item := range items {
				var req projects.ProjectMemberAddRequest
				err := helpers.ParamGroup(item,
					helpers.RequiredNumericParam(&req.Path.ProjectID, "project_id"),
					helpers.OptionalNumericListParam(&req.UserIDs, "user_ids"),
				)
				switch {
				case err != nil:
					invalid = append(invalid, fmt.Sprintf("item %d: invalid parameters: %s", i+1, err.Error()))
					continue
				case len(req.UserIDs) == 0:
					invalid = append(invalid, fmt.Sprintf("item %d: user_ids must name at least one user", i+1))
					continue
				case seen[req.Path.ProjectID]:
					invalid = append(invalid, fmt.Sprintf("item %d: project %d is listed more than once; "+
						"merge its users into one item", i+1, req.Path.ProjectID))
					continue
				}
				seen[req.Path.ProjectID] = true
				requests[i] = req
				labels[i] = fmt.Sprintf("project %d", req.Path.ProjectID)
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			errs := runBatch(ctx, len(requests), batchConcurrency, func(ctx context.Context, i int) error {
				return retryIdempotent(ctx, func() error {
					_, err := projects.ProjectMemberAdd(ctx, engine, requests[i])
					return err
				})
			})
			return batchReport("Added users to", "projects",
				"sending the same item again is safe", labels, nil, errs), nil
		},
	}
}
