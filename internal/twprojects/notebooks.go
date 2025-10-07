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

var (
	notebookGetOutputSchema  *jsonschema.Schema
	notebookListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodNotebookCreate)
	toolsets.RegisterMethod(MethodNotebookUpdate)
	toolsets.RegisterMethod(MethodNotebookDelete)
	toolsets.RegisterMethod(MethodNotebookGet)
	toolsets.RegisterMethod(MethodNotebookList)

	var err error

	// generate the output schemas only once
	notebookGetOutputSchema, err = jsonschema.For[projects.NotebookGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for NotebookGetResponse: %v", err))
	}
	notebookListOutputSchema, err = jsonschema.For[projects.NotebookListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for NotebookListResponse: %v", err))
	}
}

// NotebookCreate creates a notebook in Teamwork.com.
func NotebookCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodNotebookCreate),
			Description: "Create a new notebook in Teamwork.com. " + notebookDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Notebook",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the notebook.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to create the notebook in.",
					},
					"description": {
						Type:        "string",
						Description: "A description of the notebook.",
					},
					"contents": {
						Type:        "string",
						Description: "The contents of the notebook.",
					},
					"type": {
						Type:        "string",
						Description: "The type of the notebook. Valid values are 'MARKDOWN' and 'HTML'.",
						Enum:        []any{"MARKDOWN", "HTML"},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the notebook.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name", "project_id", "contents", "type"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookCreateRequest projects.NotebookCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			notebookResponse, err := projects.NotebookCreate(ctx, engine, notebookCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create notebook")
			}
			return helpers.NewToolResultText("Notebook created successfully with ID %d", notebookResponse.Notebook.ID), nil
		},
	}
}

// NotebookUpdate updates a notebook in Teamwork.com.
func NotebookUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodNotebookUpdate),
			Description: "Update an existing notebook in Teamwork.com. " + notebookDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Notebook",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the notebook to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the notebook.",
					},
					"description": {
						Type:        "string",
						Description: "A description of the notebook.",
					},
					"contents": {
						Type:        "string",
						Description: "The contents of the notebook.",
					},
					"type": {
						Type:        "string",
						Description: "The type of the notebook. Valid values are 'MARKDOWN' and 'HTML'.",
						Enum:        []any{"MARKDOWN", "HTML"},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the notebook.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookUpdateRequest projects.NotebookUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.NotebookUpdate(ctx, engine, notebookUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update notebook")
			}
			return helpers.NewToolResultText("Notebook updated successfully"), nil
		},
	}
}

// NotebookDelete deletes a notebook in Teamwork.com.
func NotebookDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodNotebookDelete),
			Description: "Delete an existing notebook in Teamwork.com. " + notebookDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Notebook",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the notebook to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookDeleteRequest projects.NotebookDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&notebookDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.NotebookDelete(ctx, engine, notebookDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete notebook")
			}
			return helpers.NewToolResultText("Notebook deleted successfully"), nil
		},
	}
}

// NotebookGet retrieves a notebook in Teamwork.com.
func NotebookGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodNotebookGet),
			Description: "Get an existing notebook in Teamwork.com. " + notebookDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Notebook",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the notebook to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: notebookGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookGetRequest projects.NotebookGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&notebookGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			notebook, err := projects.NotebookGet(ctx, engine, notebookGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get notebook")
			}

			encoded, err := json.Marshal(notebook)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/notebooks"),
						)),
					},
				},
				StructuredContent: notebook,
			}, nil
		},
	}
}

// NotebookList lists notebooks in Teamwork.com.
func NotebookList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodNotebookList),
			Description: "List notebooks in Teamwork.com. " + notebookDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Notebooks",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_ids": {
						Type:        "array",
						Description: "A list of project IDs to filter notebooks by projects",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"search_term": {
						Type: "string",
						Description: "A search term to filter notebooks by name or description. " +
							"The notebook will be selected if each word of the term matches the notebook name or description, not " +
							"requiring that the word matches are in the same field.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter notebooks by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match notebooks that have all the specified tags. " +
							"If false, the search will match notebooks that have any of the specified tags. " +
							"Defaults to false.",
					},
					"include_contents": {
						Type: "boolean",
						Description: "If true, the contents of the notebook will be included in the response. " +
							"Defaults to true.",
						Default: json.RawMessage(`true`),
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
			},
			OutputSchema: notebookListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var notebookListRequest projects.NotebookListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericListParam(&notebookListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalParam(&notebookListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&notebookListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&notebookListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalPointerParam(&notebookListRequest.Filters.IncludeContents, "include_contents"),
				helpers.OptionalNumericParam(&notebookListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&notebookListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			notebookList, err := projects.NotebookList(ctx, engine, notebookListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list notebooks")
			}

			encoded, err := json.Marshal(notebookList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/notebooks"),
						)),
					},
				},
				StructuredContent: notebookList,
			}, nil
		},
	}
}
