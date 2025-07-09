package twprojects

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork Projects MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCreateProject toolsets.Method = "twprojects-create_project"
)

const projectDescription = "The Project feature in Teamwork.com serves as the central workspace for organizing and " +
	"managing a specific piece of work or initiative. Each project provides a dedicated area where teams can plan " +
	"tasks, assign responsibilities, set deadlines, and track progress toward shared goals. Projects include tools " +
	"for communication, file sharing, milestones, and time tracking, allowing teams to stay aligned and informed " +
	"throughout the entire lifecycle of the work. Whether it's a product launch, client engagement, or internal " +
	"initiative, projects in Teamwork.com help teams structure their efforts, collaborate more effectively, and " +
	"deliver results with greater visibility and accountability."

// CreateProject creates a project in Teamwork Projects.
func CreateProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodCreateProject),
			mcp.WithDescription("Create a new project in Teamwork.com. "+projectDescription),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the project."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the project."),
			),
			mcp.WithString("start_at",
				mcp.Description("The start date of the project in the format YYYY-MM-DD."),
			),
			mcp.WithString("end_at",
				mcp.Description("The end date of the project in the format YYYY-MM-DD."),
			),
			mcp.WithNumber("company_id",
				mcp.Description("The ID of the company associated with the project."),
			),
			mcp.WithNumber("owned_id",
				mcp.Description("The ID of the user who owns the project."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the project."),
				mcp.Items(map[string]any{
					"type": "number",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var projectCreateRequest projects.ProjectCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredParam(&projectCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&projectCreateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&projectCreateRequest.StartAt, "start-at"),
				helpers.OptionalLegacyDatePointerParam(&projectCreateRequest.EndAt, "end-at"),
				helpers.OptionalNumericParam(&projectCreateRequest.CompanyID, "company-id"),
				helpers.OptionalNumericPointerParam(&projectCreateRequest.OwnerID, "owned-id"),
				helpers.OptionalNumericListParam(&projectCreateRequest.Tags, "tag-ids"),
			)
			if err != nil {
				return nil, fmt.Errorf("invalid parameters: %w", err)
			}

			project, err := projects.ProjectCreate(ctx, engine, projectCreateRequest)
			if err != nil {
				return nil, fmt.Errorf("failed to create project: %w", err)
			}

			return mcp.NewToolResultText(fmt.Sprintf("Project created successfully with ID %d", project.ID)), nil
		},
	}
}
