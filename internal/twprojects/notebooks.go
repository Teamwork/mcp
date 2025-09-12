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
	MethodNotebookCreate toolsets.Method = "twprojects-create_notebook"
	MethodNotebookUpdate toolsets.Method = "twprojects-update_notebook"
	MethodNotebookDelete toolsets.Method = "twprojects-delete_notebook"
	MethodNotebookGet    toolsets.Method = "twprojects-get_notebook"
	MethodNotebookList   toolsets.Method = "twprojects-list_notebooks"
)

const notebookDescription = "Notebook is a space where teams can create, share, and organize written content in a " +
	"structured way. It’s commonly used for documenting processes, storing meeting notes, capturing research, or " +
	"drafting ideas that need to be revisited and refined over time. Unlike quick messages or task comments, " +
	"notebooks provide a more permanent and organized format that can be easily searched and referenced, helping " +
	"teams maintain a centralized source of knowledge and ensuring important information remains accessible to " +
	"everyone who needs it."

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodNotebookCreate)
	toolsets.RegisterMethod(MethodNotebookUpdate)
	toolsets.RegisterMethod(MethodNotebookDelete)
	toolsets.RegisterMethod(MethodNotebookGet)
	toolsets.RegisterMethod(MethodNotebookList)
}

// NotebookCreate creates a notebook in Teamwork.com.
func NotebookCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodNotebookCreate),
			mcp.WithDescription("Create a new notebook in Teamwork.com. "+notebookDescription),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the notebook."),
			),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the notebook in."),
			),
			mcp.WithString("description",
				mcp.Description("A description of the notebook."),
			),
			mcp.WithString("contents",
				mcp.Required(),
				mcp.Description("The contents of the notebook."),
			),
			mcp.WithString("type",
				mcp.Required(),
				mcp.Description("The type of the notebook. Valid values are 'MARKDOWN' and 'HTML'."),
				mcp.Enum("MARKDOWN", "HTML"),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the notebook."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookCreateRequest projects.NotebookCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&notebookCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&notebookCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&notebookCreateRequest.Description, "description"),
				helpers.RequiredParam(&notebookCreateRequest.Contents, "contents"),
				helpers.RequiredParam(&notebookCreateRequest.Type, "type",
					helpers.RestrictValues(
						projects.NotebookTypeMarkdown,
						projects.NotebookTypeHTML,
					),
				),
				helpers.OptionalNumericListParam(&notebookCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			notebookResponse, err := projects.NotebookCreate(ctx, engine, notebookCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create notebook")
			}
			id := notebookResponse.Notebook.ID

			return mcp.NewToolResultText(fmt.Sprintf("Notebook created successfully with ID %d", id)), nil
		},
	}
}

// NotebookUpdate updates a notebook in Teamwork.com.
func NotebookUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodNotebookUpdate),
			mcp.WithDescription("Update an existing notebook in Teamwork.com. "+notebookDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the notebook to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the notebook."),
			),
			mcp.WithString("description",
				mcp.Description("A description of the notebook."),
			),
			mcp.WithString("contents",
				mcp.Description("The contents of the notebook."),
			),
			mcp.WithString("type",
				mcp.Description("The type of the notebook. Valid values are 'MARKDOWN' and 'HTML'."),
				mcp.Enum("MARKDOWN", "HTML"),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the notebook."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookUpdateRequest projects.NotebookUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&notebookUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&notebookUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&notebookUpdateRequest.Description, "description"),
				helpers.OptionalPointerParam(&notebookUpdateRequest.Contents, "contents"),
				helpers.OptionalPointerParam(&notebookUpdateRequest.Type, "type",
					helpers.RestrictValues(
						projects.NotebookTypeMarkdown,
						projects.NotebookTypeHTML,
					),
				),
				helpers.OptionalNumericListParam(&notebookUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.NotebookUpdate(ctx, engine, notebookUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update notebook")
			}

			return mcp.NewToolResultText("Notebook updated successfully"), nil
		},
	}
}

// NotebookDelete deletes a notebook in Teamwork.com.
func NotebookDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodNotebookDelete),
			mcp.WithDescription("Delete an existing notebook in Teamwork.com. "+notebookDescription),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the notebook to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookDeleteRequest projects.NotebookDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&notebookDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.NotebookDelete(ctx, engine, notebookDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete notebook")
			}

			return mcp.NewToolResultText("Notebook deleted successfully"), nil
		},
	}
}

// NotebookGet retrieves a notebook in Teamwork.com.
func NotebookGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodNotebookGet),
			mcp.WithDescription("Get an existing notebook in Teamwork.com. "+notebookDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the notebook to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookGetRequest projects.NotebookGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&notebookGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			notebook, err := projects.NotebookGet(ctx, engine, notebookGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get notebook")
			}

			encoded, err := json.Marshal(notebook)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/notebooks"),
			))), nil
		},
	}
}

// NotebookList lists notebooks in Teamwork.com.
func NotebookList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodNotebookList),
			mcp.WithDescription("List notebooks in Teamwork.com. "+notebookDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithArray("project_ids",
				mcp.Description("A list of project IDs to filter notebooks by projects"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter notebooks by name or description. "+
					"The notebook will be selected if each word of the term matches the notebook name or description, not "+
					"requiring that the word matches are in the same field."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter notebooks by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match notebooks that have all the specified tags. "+
					"If false, the search will match notebooks that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithBoolean("include_contents",
				mcp.Description("If true, the contents of the notebook will be included in the response. "+
					"Defaults to true."),
				mcp.DefaultBool(true),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookListRequest projects.NotebookListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalNumericListParam(&notebookListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalParam(&notebookListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&notebookListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&notebookListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalPointerParam(&notebookListRequest.Filters.IncludeContents, "include_contents"),
				helpers.OptionalNumericParam(&notebookListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&notebookListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			notebookList, err := projects.NotebookList(ctx, engine, notebookListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list notebooks")
			}

			encoded, err := json.Marshal(notebookList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/notebooks"),
			))), nil
		},
	}
}
