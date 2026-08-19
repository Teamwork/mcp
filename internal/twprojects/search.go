package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

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
	MethodSearch toolsets.Method = "twprojects-search"
)

// searchSideloads is every sideload the SDK models, and the default when the
// caller does not narrow `include`. The API also accepts files and
// fileversions, but the SDK does not model those sideloads yet.
var searchSideloads = []projects.SearchRequestSideload{
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

// searchTruncatedFields maps each sideload section to its free-form content
// fields and the tool that returns the full record. Note the comment sideload
// serializes the comment's content under "title".
var searchTruncatedFields = map[string]struct {
	fields []string
	method toolsets.Method
}{
	"comments":   {fields: []string{"title"}, method: MethodCommentGet},
	"links":      {fields: []string{"description"}, method: MethodLinkGet},
	"messages":   {fields: []string{"body"}, method: MethodMessageGet},
	"milestones": {fields: []string{"description"}, method: MethodMilestoneGet},
	"notebooks":  {fields: []string{"description", "contents"}, method: MethodNotebookGet},
	"projects":   {fields: []string{"description"}, method: MethodProjectGet},
	"tasklists":  {fields: []string{"description"}, method: MethodTasklistGet},
	"tasks":      {fields: []string{"description"}, method: MethodTaskGet},
	"teams":      {fields: []string{"description"}, method: MethodTeamGet},
	"timelogs":   {fields: []string{"description"}, method: MethodTimelogGet},
}

// searchMinimalFields is what each sideloaded record carries when
// verbose=false: id plus its identifying label. A comment's content doubles as
// its label ("title") and is capped by truncation like any other field.
var searchMinimalFields = projects.SearchFields{
	Comments:   []projects.CommentSideloadField{projects.CommentSideloadFieldID, projects.CommentSideloadFieldBody},
	Companies:  []projects.CompanyField{projects.CompanyFieldID, projects.CompanyFieldName},
	Links:      []projects.LinkField{projects.LinkFieldID, projects.LinkFieldTitle},
	Messages:   []projects.MessageField{projects.MessageFieldID, projects.MessageFieldTitle},
	Milestones: []projects.MilestoneField{projects.MilestoneFieldID, projects.MilestoneFieldName},
	Notebooks:  []projects.NotebookField{projects.NotebookFieldID, projects.NotebookFieldName},
	Projects:   []projects.ProjectField{projects.ProjectFieldID, projects.ProjectFieldName},
	Tasklists:  []projects.TasklistField{projects.TasklistFieldID, projects.TasklistFieldName},
	Tasks:      []projects.TaskField{projects.TaskFieldID, projects.TaskFieldName},
	Teams:      []projects.TeamField{projects.TeamFieldID, projects.TeamFieldName},
	Timelogs:   []projects.TimelogField{projects.TimelogFieldID, projects.TimelogFieldDescription},
	Users:      []projects.UserField{projects.UserFieldID, projects.UserFieldFirstName, projects.UserFieldLastName},
}

var (
	searchOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	searchOutputSchema, err = jsonschema.For[projects.SearchResponse](helpers.WithDateTypeSchema(&jsonschema.ForOptions{}))
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for SearchResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(searchOutputSchema)
	withSearchHighlightsSchema(searchOutputSchema)
}

// withSearchHighlightsSchema documents the highlights on a hit, whose meta the
// SDK models as a free-form map. The shape goes in the description because
// helpers.WithOptionalFields strips additionalProperties.
func withSearchHighlightsSchema(schema *jsonschema.Schema) {
	items := schema.Properties["search"]
	if items == nil || items.Items == nil {
		panic("search output schema has no hit list")
	}
	meta := items.Items.Properties["meta"]
	if meta == nil || meta.Properties == nil {
		panic("search hit schema has no meta object")
	}
	meta.Properties["highlights"] = &jsonschema.Schema{
		Type:        "object",
		Description: "Fragments that matched, keyed by field, with the matches wrapped in <em> tags.",
	}
}

// Search lists searches in Teamwork.com.
func Search(engine *twapi.Engine) toolsets.ToolWrapper {
	searchIncludeEnum := make([]any, len(searchSideloads))
	for i, sideload := range searchSideloads {
		searchIncludeEnum[i] = string(sideload)
	}

	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSearch),
			Description: "Cross-entity keyword search across projects, tasks, files, messages, and more. " +
				"Long content fields in the sideloaded records are truncated at " +
				strconv.Itoa(contentTruncationLimit) + " characters and marked where they are cut; " +
				"the marker names the tool that returns the full record.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Search",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
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
					"updated_after": helpers.DateTimeFilterSchema("Only include items updated after this date."),
					"extended_search": {
						Description: "Whether to perform an extended search, which includes items updated more than 5 years ago.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include_highlights": {
						Description: "Whether to return why each result matched, as fragments under its meta.highlights. " +
							"Unavailable on an extended search.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include": {
						Description: "Entity types to return as full records in the response sideloads. " +
							"Defaults to all supported types; narrow it to the types of interest to keep the response small.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "string",
									Enum: searchIncludeEnum,
								},
							},
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
					"verbose": helpers.VerboseSchema(),
				},
				Required: []string{"search_term"},
			},
			OutputSchema: helpers.WithOptionalFields(searchOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var searchRequest projects.SearchRequest
			searchRequest.Filters.Include = searchSideloads

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&searchRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&searchRequest.Filters.ProjectID, "project_id"),
				helpers.OptionalPointerParam(&searchRequest.Filters.IncludeCompletedItems, "include_completed_items"),
				helpers.OptionalTimeParam(&searchRequest.Filters.UpdatedAfter, "updated_after"),
				helpers.OptionalPointerParam(&searchRequest.Filters.ExtendedSearch, "extended_search"),
				helpers.OptionalPointerParam(&searchRequest.Filters.IncludeHighlights, "include_highlights"),
				helpers.OptionalListParam(&searchRequest.Filters.Include, "include",
					helpers.RestrictValues(searchSideloads...)),
				helpers.OptionalParam(&searchRequest.Filters.Cursor, "cursor"),
				helpers.OptionalNumericParam(&searchRequest.Filters.Limit, "limit"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				searchRequest.Filters.Fields = searchMinimalFields
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, searchRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to search")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to search"), "failed to search")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			truncated := truncateSearchSideloads(body)
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(truncated)},
				},
			}
			var structured any
			if err := json.Unmarshal(truncated, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}

// truncateSearchSideloads caps the content fields named in
// searchTruncatedFields across a raw search response. Anything unexpected
// leaves the payload untouched: returning the response whole beats failing a
// read the API already answered.
func truncateSearchSideloads(data []byte) []byte {
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return data
	}
	included, ok := decoded["included"].(map[string]any)
	if !ok {
		return data
	}

	var truncated bool
	for section, truncation := range searchTruncatedFields {
		records, ok := included[section].(map[string]any)
		if !ok {
			continue
		}
		for key, item := range records {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id := record["id"]
			if id == nil {
				// sideload sections are keyed by the record's ID
				id = key
			}
			for _, field := range truncation.fields {
				content, ok := record[field].(string)
				if !ok {
					continue
				}
				short, ok := truncateContent(content, truncation.method, id)
				if !ok {
					continue
				}
				record[field] = short
				truncated = true
			}
		}
	}
	if !truncated {
		return data
	}

	encoded, err := json.Marshal(decoded)
	if err != nil {
		return data
	}
	return encoded
}
