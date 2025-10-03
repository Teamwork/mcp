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
	MethodProjectMemberAdd toolsets.Method = "twprojects-add_project_member"
)

const projectMemberDescription = "In the context of Teamwork.com, a project member is a user who is assigned to a " +
	"specific project. Project members can have different roles and permissions within the project, allowing them to " +
	"collaborate on tasks, view project details, and contribute to the project's success. Managing project members " +
	"effectively is crucial for ensuring that the right people are involved in the right tasks, and it helps maintain " +
	"accountability and clarity throughout the project's lifecycle."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodProjectMemberAdd)
}

// ProjectMemberAdd adds a user to a project in Teamwork.com.
func ProjectMemberAdd(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodProjectMemberAdd),
			Description: "Add a user to a project in Teamwork.com. " + projectMemberDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Add Project Member",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to add the member to.",
					},
					"user_ids": {
						Type:        "array",
						Description: "A list of user IDs to add to the project.",
						Items: &jsonschema.Schema{
							Type: "integer",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&projectMemberAddRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericListParam(&projectMemberAddRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.ProjectMemberAdd(ctx, engine, projectMemberAddRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to add project member")
			}

			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: "Project member added successfully",
					},
				},
			}, nil
		},
	}
}
