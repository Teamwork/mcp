package twprojects

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	MethodMilestoneCreate toolsets.Method = "twprojects-create_milestone"
	MethodMilestoneUpdate toolsets.Method = "twprojects-update_milestone"
	MethodMilestoneDelete toolsets.Method = "twprojects-delete_milestone"
	MethodMilestoneGet    toolsets.Method = "twprojects-get_milestone"
	MethodMilestoneList   toolsets.Method = "twprojects-list_milestones"
)

var (
	milestoneGetOutputSchema  *jsonschema.Schema
	milestoneListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	// generate the output schemas only once
	milestoneGetOutputSchema, err = jsonschema.For[projects.MilestoneGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MilestoneGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(milestoneGetOutputSchema)
	milestoneListOutputSchema, err = jsonschema.For[projects.MilestoneListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MilestoneListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(milestoneListOutputSchema)
}

// MilestoneCreate creates a milestone in Teamwork.com.
func MilestoneCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneCreate),
			Description: "Create milestone in a project.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Milestone",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the milestone.",
					},
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project to create the milestone in.",
					},
					"description": {
						Description: "A description of the milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"due_date": {
						Type: "string",
						Description: "The due date of the milestone in the format YYYYMMDD. This date will be used in all tasks " +
							"without a due date related to this milestone.",
					},
					"assignees": {
						Type: "object",
						Description: "An object containing assignees for the milestone. " +
							"MUST contain at least one of: user_ids, company_ids or team_ids with non-empty arrays.",
						Properties: map[string]*jsonschema.Schema{
							"user_ids": {
								Type:        "array",
								Description: "List of user IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: new(1),
							},
							"company_ids": {
								Type:        "array",
								Description: "List of company IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: new(1),
							},
							"team_ids": {
								Type:        "array",
								Description: "List of team IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: new(1),
							},
						},
						MinProperties: new(1),
						MaxProperties: new(3),
						AnyOf: []*jsonschema.Schema{
							{Required: []string{"user_ids"}},
							{Required: []string{"company_ids"}},
							{Required: []string{"team_ids"}},
						},
					},
					"tasklist_ids": {
						Description: "A list of tasklist IDs to associate with the milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("milestone"),
				},
				Required: []string{"name", "project_id", "due_date", "assignees"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneCreateRequest projects.MilestoneCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&milestoneCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&milestoneCreateRequest.Description, "description"),
				helpers.RequiredLegacyDateParam(&milestoneCreateRequest.DueAt, "due_date"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.TasklistIDs, "tasklist_ids"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if _, ok := arguments["assignees"]; !ok {
				return helpers.NewToolResultTextError("missing required parameter: assignees"), nil
			}
			assignees, toolResult := parseLegacyUserGroups(
				arguments,
				"assignees",
				"assignees",
			)
			if toolResult != nil {
				return toolResult, nil
			}
			if assignees == nil {
				return helpers.NewToolResultTextError("assignees cannot be null"), nil
			}
			milestoneCreateRequest.Assignees = *assignees
			if milestoneCreateRequest.Assignees.IsEmpty() {
				return helpers.NewToolResultTextError("at least one assignee must be provided"), nil
			}

			milestone, err := projects.MilestoneCreate(ctx, engine, milestoneCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create milestone")
			}
			return helpers.NewToolResultText("Milestone created successfully with ID %d", milestone.ID), nil
		},
	}
}

// MilestoneUpdate updates a milestone in Teamwork.com.
func MilestoneUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneUpdate),
			Description: "Update milestone.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Milestone",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the milestone to update.",
					},
					"name": {
						Description: "The name of the milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "A description of the milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"due_date": {
						Description: "The due date of the milestone in the format YYYYMMDD. This date will be used in all tasks " +
							"without a due date related to this milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"assignees": {
						Description: "An object containing assignees for the milestone.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										Type:        "array",
										Description: "List of user IDs assigned to the milestone.",
										Items: &jsonschema.Schema{
											Type: "integer",
										},
										MinItems: new(1),
									},
									"company_ids": {
										Type:        "array",
										Description: "List of company IDs assigned to the milestone.",
										Items: &jsonschema.Schema{
											Type: "integer",
										},
										MinItems: new(1),
									},
									"team_ids": {
										Type:        "array",
										Description: "List of team IDs assigned to the milestone.",
										Items: &jsonschema.Schema{
											Type: "integer",
										},
										MinItems: new(1),
									},
								},
								MinProperties: new(1),
								MaxProperties: new(3),
								AnyOf: []*jsonschema.Schema{
									{Required: []string{"user_ids"}},
									{Required: []string{"company_ids"}},
									{Required: []string{"team_ids"}},
								},
							},
							{Type: "null"},
						},
					},
					"tasklist_ids": {
						Description: "A list of tasklist IDs to associate with the milestone.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"tag_ids": helpers.TagIDsAssociateSchema("milestone"),
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneUpdateRequest projects.MilestoneUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&milestoneUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&milestoneUpdateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&milestoneUpdateRequest.DueAt, "due_date"),
				helpers.OptionalNumericListParam(&milestoneUpdateRequest.TasklistIDs, "tasklist_ids"),
				helpers.OptionalNumericListParam(&milestoneUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if assignees, toolResult := parseLegacyUserGroups(
				arguments,
				"assignees",
				"assignees",
			); toolResult != nil {
				return toolResult, nil
			} else if assignees != nil {
				milestoneUpdateRequest.Assignees = assignees
			}

			_, err = projects.MilestoneUpdate(ctx, engine, milestoneUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update milestone")
			}
			return helpers.NewToolResultText("Milestone updated successfully"), nil
		},
	}
}

// MilestoneDelete deletes a milestone in Teamwork.com.
func MilestoneDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneDelete),
			Description: "Delete milestone.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Milestone",
				DestructiveHint: new(true),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the milestone to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneDeleteRequest projects.MilestoneDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.MilestoneDelete(ctx, engine, milestoneDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete milestone")
			}
			return helpers.NewToolResultText("Milestone deleted successfully"), nil
		},
	}
}

// MilestoneGet retrieves a milestone in Teamwork.com.
func MilestoneGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneGet),
			Description: "Get milestone.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Milestone",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the milestone to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: milestoneGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneGetRequest projects.MilestoneGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			milestone, err := projects.MilestoneGet(ctx, engine, milestoneGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get milestone")
			}

			encoded, err := json.Marshal(milestone)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/milestones"),
						)),
					},
				},
				StructuredContent: helpers.StructuredWebLinker(ctx, milestone,
					helpers.WebLinkerWithIDPathBuilder("/app/milestones"),
				),
			}, nil
		},
	}
}

// MilestoneList lists milestones in Teamwork.com.
func MilestoneList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneList),
			Description: "List milestones. Scope by project_id or omit for site-wide.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Milestones",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Description: "The ID of the project from which to retrieve milestones. Omit to list milestones across " +
							"all projects.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"search_term": {
						Description: "A search term to filter milestones by name. " +
							"Each word from the search term is used to match against the milestone name and description. " +
							"The milestone will be selected if each word of the term matches the milestone name or description, not " +
							"requiring that the word matches are in the same field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"tag_ids":        helpers.TagIDsFilterSchema("milestones"),
					"match_all_tags": helpers.MatchAllTagsSchema("milestones"),
					"page":           helpers.PageSchema(),
					"page_size":      helpers.PageSizeSchema(),
					"verbose":        helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: helpers.WithOptionalFields(milestoneListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneListRequest projects.MilestoneListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalNumericParam(&milestoneListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&milestoneListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&milestoneListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&milestoneListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				milestoneListRequest.Filters.Fields.Milestones = []projects.MilestoneField{
					projects.MilestoneFieldID,
					projects.MilestoneFieldName,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, milestoneListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list milestones")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(twapi.NewHTTPError(resp, "failed to list milestones"), "failed to list milestones")
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			linked := helpers.WebLinker(ctx, body, helpers.WebLinkerWithIDPathBuilder("/app/milestones"))
			result := &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: string(linked)},
				},
			}
			var structured any
			if err := json.Unmarshal(linked, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}
