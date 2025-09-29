package twprojects

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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
func ProjectMemberAdd(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodProjectMemberAdd),
			mcp.WithDescription("Add a user to a project in Teamwork.com. "+projectMemberDescription),
			mcp.WithTitleAnnotation("Add Project Member"),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to add the member to."),
			),
			mcp.WithArray("user_ids",
				mcp.Description("A list of user IDs to add to the project."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectMemberAddRequest projects.ProjectMemberAddRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&projectMemberAddRequest.Path.ProjectID, "project_id"),
				helpers.OptionalNumericListParam(&projectMemberAddRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.ProjectMemberAdd(ctx, engine, projectMemberAddRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to add project member")
			}

			return mcp.NewToolResultText("Project member added successfully"), nil
		},
	}
}
