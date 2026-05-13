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
	MethodSearch toolsets.Method = "twprojects-search"
)

var (
	searchOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	searchOutputSchema, err = jsonschema.For[projects.SearchResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for SearchResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(searchOutputSchema)
}

// Search lists searches in Teamwork.com.
func Search(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodSearch),
			Description: "Cross-entity keyword search across projects, tasks, files, messages, and more.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Search",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to to look for items.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to limit the search to.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"include_completed_items": {
						Description: "Whether to include completed items in the search results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"updated_after": {
						Description: "Only include items updated after this date. Must be follow RFC3339 format.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
						Format:   "date-time",
						Examples: []any{"2023-01-01T00:00:00Z"},
					},
					"extended_search": {
						Description: "Whether to perform an extended search, which includes items updated more than 5 years ago.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"cursor": {
						Description: "Cursor for pagination of results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"limit": {
						Description: "Number of results per page for pagination.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"search_term"},
			},
			OutputSchema: searchOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var searchRequest projects.SearchRequest
			searchRequest.Filters.Include = []projects.SearchRequestSideload{
				projects.SearchRequestSideloadComments,
				projects.SearchRequestSideloadCompanies,
				projects.SearchRequestSideloadLinks,
				projects.SearchRequestSideloadMessages,
				projects.SearchRequestSideloadMilestones,
				projects.SearchRequestSideloadNotebooks,
				projects.SearchRequestSideloadProjects,
				projects.SearchRequestSideloadTasklists,
				projects.SearchRequestSideloadTasks,
				projects.SearchRequestSideloadTeams,
				projects.SearchRequestSideloadTimelogs,
				projects.SearchRequestSideloadUsers,
			}

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&searchRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&searchRequest.Filters.ProjectID, "project_id"),
				helpers.OptionalPointerParam(&searchRequest.Filters.IncludeCompletedItems, "include_completed_items"),
				helpers.OptionalTimeParam(&searchRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalPointerParam(&searchRequest.Filters.ExtendedSearch, "extended_search"),
				helpers.OptionalParam(&searchRequest.Filters.Cursor, "cursor"),
				helpers.OptionalNumericParam(&searchRequest.Filters.Limit, "limit"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			search, err := projects.Search(ctx, engine, searchRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list searches")
			}

			encoded, err := json.Marshal(search)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: search,
			}, nil
		},
	}
}
