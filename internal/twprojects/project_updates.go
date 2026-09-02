package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	MethodProjectStatusUpdateList toolsets.Method = "twprojects-list_project_updates"
)

var projectStatusUpdateListOutputSchema *jsonschema.Schema

// projectStatusUpdateOrdering is the order-by vocabulary of the project updates
// list endpoint.
var projectStatusUpdateOrdering = newOrdering("project updates",
	projects.ProjectStatusUpdateOrderByDate,
	projects.ProjectStatusUpdateOrderByColor,
	projects.ProjectStatusUpdateOrderByHealth,
	projects.ProjectStatusUpdateOrderByProject,
	projects.ProjectStatusUpdateOrderByUser,
	projects.ProjectStatusUpdateOrderByID,
)

func init() {
	var err error

	// generate the output schema only once
	projectStatusUpdateListOutputSchema, err = jsonschema.For[projects.ProjectStatusUpdateListResponse](
		helpers.WithDateTypeSchema(&jsonschema.ForOptions{}),
	)
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for ProjectStatusUpdateListResponse: %v", err))
	}
}

// ProjectStatusUpdateList lists project updates in Teamwork.com.
func ProjectStatusUpdateList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodProjectStatusUpdateList),
			Description: "List project updates: the Markdown text on a project's dashboard and the health it " +
				"reports. Nothing else returns either — a project row carries no health, and the activity log " +
				"holds a preview of the text, not the text. Newest first. Only each project's current update " +
				"is returned unless active_only is false, so an unfiltered call is not the whole history. " +
				"Rows carry text (Markdown, in full, emoji codes already converted to characters), health " +
				"(0 not set, 1 bad, 2 ok, 3 good), healthLabel (the name this installation gives that rating " +
				"— read it, never build one from health) and color (hex, empty when the health is not set). " +
				"The author and the project are bare IDs, sideloaded under included when verbose is true. " +
				"Keep a response small with fields and page_size; verbose=false drops the text and returns " +
				"the ratings.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Project Updates",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_ids": {
						Description: "Only return the updates of these projects. Naming any project makes the " +
							"endpoint ignore every other project filter, including project_healths and " +
							"include_archived.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"project_healths": projectHealthVocabulary.arraySchema(
						"Only return the updates reporting these health ratings, matching any of the values " +
							"given. \"not_set\" matches an update that rates nothing. Ignored when project_ids " +
							"is set.",
					),
					"active_only": {
						Description: "If true (the default), return only each project's current update. Set it " +
							"to false to read the update history, which returns every past update in full.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
						Default: []byte(`true`),
					},
					"show_deleted": {
						Description: "If true, return deleted updates alongside the live ones; excluded by default.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include_archived": {
						Description: "If true, return the updates of archived projects alongside those of the " +
							"active ones; excluded by default. Ignored when project_ids is set.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"created_after": helpers.DateTimeFilterSchema(
						"Only include updates posted at or after this moment; the boundary itself matches."),
					"updated_after": helpers.DateTimeFilterSchema(
						"Only include updates last edited strictly after this moment; the boundary itself does " +
							"not match."),
					"order_by":   projectStatusUpdateOrdering.orderBySchema(),
					"order_mode": orderModeSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"count_only": helpers.CountOnlySchema("project updates"),
					"fields":     helpers.FieldsSchema[projects.ProjectStatusUpdate]("project update"),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithCountOnlySchema(
				helpers.WithOptionalFields(projectStatusUpdateListOutputSchema),
			),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			projectStatusUpdateListRequest := projects.NewProjectStatusUpdateListRequest()

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			var countOnly bool
			filters := &projectStatusUpdateListRequest.Filters
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericListParam(&filters.ProjectIDs, "project_ids"),
				projectHealthVocabulary.listParam(&filters.ProjectHealths, "project_healths"),
				helpers.OptionalPointerParam(&filters.ActiveOnly, "active_only"),
				helpers.OptionalPointerParam(&filters.ShowDeleted, "show_deleted"),
				helpers.OptionalPointerParam(&filters.IncludeArchivedProjects, "include_archived"),
				helpers.OptionalTimePointerParam(&filters.CreatedAfter, "created_after"),
				helpers.OptionalTimePointerParam(&filters.UpdatedAfter, "updated_after"),
				projectStatusUpdateOrdering.param(&filters.OrderBy, &filters.OrderMode),
				helpers.OptionalNumericParam(&filters.Page, "page"),
				helpers.OptionalNumericParam(&filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.ProjectStatusUpdate](&filters.Fields.ProjectUpdates, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, projectStatusUpdateListRequest,
					"failed to count project updates")
			}

			switch {
			case len(filters.Fields.ProjectUpdates) > 0:
				// An explicit field selection overrides both defaults below: the
				// caller has already said what it wants, and the sideloads would
				// smuggle back the bulk the selection exists to avoid.

			case verbose:
				// The route reports the author and the project as bare identifiers,
				// so sideload both to spare the caller a follow-up lookup for a
				// name it almost always needs.
				filters.Include = []projects.ProjectStatusUpdateListRequestSideload{
					projects.ProjectStatusUpdateListRequestSideloadCreatedBy,
					projects.ProjectStatusUpdateListRequestSideloadProjects,
				}

			default:
				// The text is the bulk of a row, and a caller scanning many
				// projects wants the ratings. Ask for the text by name through
				// fields when both are needed.
				filters.Fields.ProjectUpdates = []projects.ProjectStatusUpdateField{
					projects.ProjectStatusUpdateFieldID,
					projects.ProjectStatusUpdateFieldProjectID,
					projects.ProjectStatusUpdateFieldHealth,
					projects.ProjectStatusUpdateFieldHealthLabel,
					projects.ProjectStatusUpdateFieldColor,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, projectStatusUpdateListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list project updates")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list project updates"),
					"failed to list project updates")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(body)},
				},
			}
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
