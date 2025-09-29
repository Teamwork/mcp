package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

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
	MethodTasklistCreate        toolsets.Method = "twprojects-create_tasklist"
	MethodTasklistUpdate        toolsets.Method = "twprojects-update_tasklist"
	MethodTasklistDelete        toolsets.Method = "twprojects-delete_tasklist"
	MethodTasklistGet           toolsets.Method = "twprojects-get_tasklist"
	MethodTasklistList          toolsets.Method = "twprojects-list_tasklists"
	MethodTasklistListByProject toolsets.Method = "twprojects-list_tasklists_by_project"
)

const tasklistDescription = "In the context of Teamwork.com, a task list is a way to group related tasks within a " +
	"project, helping teams organize their work into meaningful sections such as phases, categories, or deliverables. " +
	"Each task list belongs to a specific project and can include multiple tasks that are typically aligned with a " +
	"common goal. Task lists can be associated with milestones, and they support privacy settings that control who " +
	"can view or interact with the tasks they contain. This structure helps teams manage progress, assign " +
	"responsibilities, and maintain clarity across complex projects."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTasklistCreate)
	toolsets.RegisterMethod(MethodTasklistUpdate)
	toolsets.RegisterMethod(MethodTasklistDelete)
	toolsets.RegisterMethod(MethodTasklistGet)
	toolsets.RegisterMethod(MethodTasklistList)
	toolsets.RegisterMethod(MethodTasklistListByProject)
}

// TasklistCreate creates a tasklist in Teamwork.com.
func TasklistCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTasklistCreate),
			mcp.WithDescription("Create a new tasklist in Teamwork.com. "+tasklistDescription),
			mcp.WithTitleAnnotation("Create Tasklist"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the tasklist."),
			),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the tasklist in."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the tasklist."),
			),
			mcp.WithNumber("milestone_id",
				mcp.Description("The ID of the milestone to associate with the tasklist."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistCreateRequest projects.TasklistCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredParam(&tasklistCreateRequest.Name, "name"),
				helpers.RequiredNumericParam(&tasklistCreateRequest.Path.ProjectID, "project_id"),
				helpers.OptionalPointerParam(&tasklistCreateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&tasklistCreateRequest.MilestoneID, "milestone_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tasklist, err := projects.TasklistCreate(ctx, engine, tasklistCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create tasklist")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Tasklist created successfully with ID %d", tasklist.ID)), nil
		},
	}
}

// TasklistUpdate updates a tasklist in Teamwork.com.
func TasklistUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTasklistUpdate),
			mcp.WithDescription("Update an existing tasklist in Teamwork.com. "+tasklistDescription),
			mcp.WithTitleAnnotation("Update Tasklist"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the tasklist to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the tasklist."),
			),
			mcp.WithString("description",
				mcp.Description("The description of the tasklist."),
			),
			mcp.WithNumber("milestone_id",
				mcp.Description("The ID of the milestone to associate with the tasklist."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistUpdateRequest projects.TasklistUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tasklistUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&tasklistUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&tasklistUpdateRequest.Description, "description"),
				helpers.OptionalNumericPointerParam(&tasklistUpdateRequest.MilestoneID, "milestone_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TasklistUpdate(ctx, engine, tasklistUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update tasklist")
			}

			return mcp.NewToolResultText("Tasklist updated successfully"), nil
		},
	}
}

// TasklistDelete deletes a tasklist in Teamwork.com.
func TasklistDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTasklistDelete),
			mcp.WithDescription("Delete an existing tasklist in Teamwork.com. "+tasklistDescription),
			mcp.WithTitleAnnotation("Delete Tasklist"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the tasklist to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistDeleteRequest projects.TasklistDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tasklistDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TasklistDelete(ctx, engine, tasklistDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete tasklist")
			}

			return mcp.NewToolResultText("Tasklist deleted successfully"), nil
		},
	}
}

// TasklistGet retrieves a tasklist in Teamwork.com.
func TasklistGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTasklistGet),
			mcp.WithDescription("Get an existing tasklist in Teamwork.com. "+tasklistDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Tasklist"),
			mcp.WithOutputSchema[projects.TasklistGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the tasklist to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistGetRequest projects.TasklistGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tasklistGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tasklist, err := projects.TasklistGet(ctx, engine, tasklistGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get tasklist")
			}

			encoded, err := json.Marshal(tasklist)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasklists"),
			))), nil
		},
	}
}

// TasklistList lists tasklists in Teamwork.com.
func TasklistList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTasklistList),
			mcp.WithDescription("List tasklists in Teamwork.com. "+tasklistDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Tasklists"),
			mcp.WithOutputSchema[projects.TasklistListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter tasklists by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistListRequest projects.TasklistListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&tasklistListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tasklistList, err := projects.TasklistList(ctx, engine, tasklistListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasklists")
			}

			encoded, err := json.Marshal(tasklistList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasklists"),
			))), nil
		},
	}
}

// TasklistListByProject lists tasklists in Teamwork.com by project.
func TasklistListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTasklistListByProject),
			mcp.WithDescription("List tasklists in Teamwork.com by project. "+tasklistDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Tasklists By Project"),
			mcp.WithOutputSchema[projects.TasklistListResponse](),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project from which to retrieve tasklists."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter tasklists by name."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tasklistListRequest projects.TasklistListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tasklistListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&tasklistListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tasklistListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tasklistList, err := projects.TasklistList(ctx, engine, tasklistListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tasklists")
			}

			encoded, err := json.Marshal(tasklistList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/tasklists"),
			))), nil
		},
	}
}
