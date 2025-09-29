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
	MethodTagCreate toolsets.Method = "twprojects-create_tag"
	MethodTagUpdate toolsets.Method = "twprojects-update_tag"
	MethodTagDelete toolsets.Method = "twprojects-delete_tag"
	MethodTagGet    toolsets.Method = "twprojects-get_tag"
	MethodTagList   toolsets.Method = "twprojects-list_tags"
)

const tagDescription = "In the context of Teamwork.com, a tag is a customizable label that can be applied to various " +
	"items such as tasks, projects, milestones, messages, and more, to help categorize and organize work efficiently. " +
	"Tags provide a flexible way to filter, search, and group related items across the platform, making it easier for " +
	"teams to manage complex workflows, highlight priorities, or track themes and statuses. Since tags are " +
	"user-defined, they adapt to each team’s specific needs and can be color-coded for better visual clarity."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodTagCreate)
	toolsets.RegisterMethod(MethodTagUpdate)
	toolsets.RegisterMethod(MethodTagDelete)
	toolsets.RegisterMethod(MethodTagGet)
	toolsets.RegisterMethod(MethodTagList)
}

// TagCreate creates a tag in Teamwork.com.
func TagCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagCreate),
			mcp.WithDescription("Create a new tag in Teamwork.com. "+tagDescription),
			mcp.WithTitleAnnotation("Create Tag"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the tag. It must have less than 50 characters."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project to associate the tag with. This is for project-scoped tags."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagCreateRequest projects.TagCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredParam(&tagCreateRequest.Name, "name"),
				helpers.OptionalNumericPointerParam(&tagCreateRequest.ProjectID, "project_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tagResponse, err := projects.TagCreate(ctx, engine, tagCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create tag")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Tag created successfully with ID %d", tagResponse.Tag.ID)), nil
		},
	}
}

// TagUpdate updates a tag in Teamwork.com.
func TagUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagUpdate),
			mcp.WithDescription("Update an existing tag in Teamwork.com. "+tagDescription),
			mcp.WithTitleAnnotation("Update Tag"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the tag to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the tag. It must have less than 50 characters."),
			),
			mcp.WithNumber("project_id",
				mcp.Description("The ID of the project to associate the tag with. This is for project-scoped tags."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagUpdateRequest projects.TagUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tagUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&tagUpdateRequest.Name, "name"),
				helpers.OptionalNumericPointerParam(&tagUpdateRequest.ProjectID, "project_id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TagUpdate(ctx, engine, tagUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update tag")
			}

			return mcp.NewToolResultText("Tag updated successfully"), nil
		},
	}
}

// TagDelete deletes a tag in Teamwork.com.
func TagDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagDelete),
			mcp.WithDescription("Delete an existing tag in Teamwork.com. "+tagDescription),
			mcp.WithTitleAnnotation("Delete Tag"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the tag to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagDeleteRequest projects.TagDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tagDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.TagDelete(ctx, engine, tagDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete tag")
			}

			return mcp.NewToolResultText("Tag deleted successfully"), nil
		},
	}
}

// TagGet retrieves a tag in Teamwork.com.
func TagGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagGet),
			mcp.WithDescription("Get an existing tag in Teamwork.com. "+tagDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Tag"),
			mcp.WithOutputSchema[projects.TagGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the tag to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagGetRequest projects.TagGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&tagGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tag, err := projects.TagGet(ctx, engine, tagGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get tag")
			}

			encoded, err := json.Marshal(tag)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}

// TagList lists tags in Teamwork.com.
func TagList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodTagList),
			mcp.WithDescription("List tags in Teamwork.com. "+tagDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Tags"),
			mcp.WithOutputSchema[projects.TagListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter tags by name. "+
					"Each word from the search term is used to match against the tag name."),
			),
			mcp.WithString("item_type",
				mcp.Description("The type of item to filter tags by. Valid values are 'project', 'task', 'tasklist', "+
					"'milestone', 'message', 'timelog', 'notebook', 'file', 'company' and 'link'."),
			),
			mcp.WithArray("project_ids",
				mcp.Description("A list of project IDs to filter tags by projects"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var tagListRequest projects.TagListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&tagListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalParam(&tagListRequest.Filters.ItemType, "item_type",
					helpers.RestrictValues("project", "task", "tasklist", "milestone", "message", "timelog", "notebook",
						"file", "company", "link"),
				),
				helpers.OptionalNumericListParam(&tagListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalNumericParam(&tagListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&tagListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			tagList, err := projects.TagList(ctx, engine, tagListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list tags")
			}

			encoded, err := json.Marshal(tagList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(encoded)), nil
		},
	}
}
