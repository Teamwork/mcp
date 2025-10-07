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
	MethodMilestoneCreate        toolsets.Method = "twprojects-create_milestone"
	MethodMilestoneUpdate        toolsets.Method = "twprojects-update_milestone"
	MethodMilestoneDelete        toolsets.Method = "twprojects-delete_milestone"
	MethodMilestoneGet           toolsets.Method = "twprojects-get_milestone"
	MethodMilestoneList          toolsets.Method = "twprojects-list_milestones"
	MethodMilestoneListByProject toolsets.Method = "twprojects-list_milestones_by_project"
)

const milestoneDescription = "In the context of Teamwork.com, a milestone represents a significant point or goal " +
	"within a project that marks the completion of a major phase or a key deliverable. It acts as a high-level " +
	"indicator of progress, helping teams track whether work is advancing according to plan. Milestones are typically " +
	"used to coordinate efforts across different tasks and task lists, providing a clear deadline or objective that " +
	"multiple team members or departments can align around. They don't contain individual tasks themselves but serve " +
	"as checkpoints to ensure the project is moving in the right direction."

var (
	milestoneGetOutputSchema  *jsonschema.Schema
	milestoneListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodMilestoneCreate)
	toolsets.RegisterMethod(MethodMilestoneUpdate)
	toolsets.RegisterMethod(MethodMilestoneDelete)
	toolsets.RegisterMethod(MethodMilestoneGet)
	toolsets.RegisterMethod(MethodMilestoneList)
	toolsets.RegisterMethod(MethodMilestoneListByProject)

	var err error

	// generate the output schemas only once
	milestoneGetOutputSchema, err = jsonschema.For[projects.MilestoneGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MilestoneGetResponse: %v", err))
	}
	milestoneListOutputSchema, err = jsonschema.For[projects.MilestoneListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for MilestoneListResponse: %v", err))
	}
}

// MilestoneCreate creates a milestone in Teamwork.com.
func MilestoneCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneCreate),
			Description: "Create a new milestone in Teamwork.com. " + milestoneDescription,
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
						Type:        "string",
						Description: "A description of the milestone.",
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
								MinItems: twapi.Ptr(1),
							},
							"company_ids": {
								Type:        "array",
								Description: "List of company IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: twapi.Ptr(1),
							},
							"team_ids": {
								Type:        "array",
								Description: "List of team IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: twapi.Ptr(1),
							},
						},
						MinProperties: twapi.Ptr(1),
						MaxProperties: twapi.Ptr(3),
						AnyOf: []*jsonschema.Schema{
							{Required: []string{"user_ids"}},
							{Required: []string{"company_ids"}},
							{Required: []string{"team_ids"}},
						},
					},
					"tasklist_ids": {
						Type:        "array",
						Description: "A list of tasklist IDs to associate with the milestone.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the milestone.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name", "project_id", "due_date", "assignees"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneCreateRequest projects.MilestoneCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			assignees, ok := arguments["assignees"]
			if !ok {
				return helpers.NewToolResultTextError("missing required parameter: assignees"), nil
			}
			assigneesMap, ok := assignees.(map[string]any)
			if !ok {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid assignees: expected an object, got %T", assignees)), nil
			} else if assigneesMap == nil {
				return helpers.NewToolResultTextError("assignees cannot be null"), nil
			}
			err = helpers.ParamGroup(assigneesMap,
				helpers.OptionalNumericListParam(&milestoneCreateRequest.Assignees.UserIDs, "user_ids"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.Assignees.CompanyIDs, "company_ids"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.Assignees.TeamIDs, "team_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid assignees: %s", err.Error())), nil
			}
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
			Description: "Update an existing milestone in Teamwork.com. " + milestoneDescription,
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
						Type:        "string",
						Description: "The name of the milestone.",
					},
					"description": {
						Type:        "string",
						Description: "A description of the milestone.",
					},
					"due_date": {
						Type: "string",
						Description: "The due date of the milestone in the format YYYYMMDD. This date will be used in all tasks " +
							"without a due date related to this milestone.",
					},
					"assignees": {
						Type:        "object",
						Description: "An object containing assignees for the milestone.",
						Properties: map[string]*jsonschema.Schema{
							"user_ids": {
								Type:        "array",
								Description: "List of user IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: twapi.Ptr(1),
							},
							"company_ids": {
								Type:        "array",
								Description: "List of company IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: twapi.Ptr(1),
							},
							"team_ids": {
								Type:        "array",
								Description: "List of team IDs assigned to the milestone.",
								Items: &jsonschema.Schema{
									Type: "integer",
								},
								MinItems: twapi.Ptr(1),
							},
						},
						MinProperties: twapi.Ptr(1),
						MaxProperties: twapi.Ptr(3),
						AnyOf: []*jsonschema.Schema{
							{Required: []string{"user_ids"}},
							{Required: []string{"company_ids"}},
							{Required: []string{"team_ids"}},
						},
					},
					"tasklist_ids": {
						Type:        "array",
						Description: "A list of tasklist IDs to associate with the milestone.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to associate with the milestone.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneUpdateRequest projects.MilestoneUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
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
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			if assignees, ok := arguments["assignees"]; ok {
				assigneesMap, ok := assignees.(map[string]any)
				if !ok {
					return helpers.NewToolResultTextError("invalid assignees"), nil
				} else if assigneesMap != nil {
					milestoneUpdateRequest.Assignees = new(projects.LegacyUserGroups)
					err = helpers.ParamGroup(assigneesMap,
						helpers.OptionalNumericListParam(&milestoneUpdateRequest.Assignees.UserIDs, "user_ids"),
						helpers.OptionalNumericListParam(&milestoneUpdateRequest.Assignees.CompanyIDs, "company_ids"),
						helpers.OptionalNumericListParam(&milestoneUpdateRequest.Assignees.TeamIDs, "team_ids"),
					)
					if err != nil {
						return helpers.NewToolResultTextError(fmt.Sprintf("invalid assignees: %s", err.Error())), nil
					}
				}
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
			Description: "Delete an existing milestone in Teamwork.com. " + milestoneDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Milestone",
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
			Description: "Get an existing milestone in Teamwork.com. " + milestoneDescription,
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
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
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
				StructuredContent: milestone,
			}, nil
		},
	}
}

// MilestoneList lists milestones in Teamwork.com.
func MilestoneList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneList),
			Description: "List milestones in Teamwork.com. " + milestoneDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Milestones",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter milestones by name. " +
							"Each word from the search term is used to match against the milestone name and description. " +
							"The milestone will be selected if each word of the term matches the milestone name or description, not " +
							"requiring that the word matches are in the same field.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter milestones by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match milestones that have all the specified tags. " +
							"If false, the search will match milestones that have any of the specified tags. " +
							"Defaults to false.",
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
			OutputSchema: milestoneListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneListRequest projects.MilestoneListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&milestoneListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&milestoneListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&milestoneListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			milestoneList, err := projects.MilestoneList(ctx, engine, milestoneListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list milestones")
			}

			encoded, err := json.Marshal(milestoneList)
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
				StructuredContent: milestoneList,
			}, nil
		},
	}
}

// MilestoneListByProject lists milestones in Teamwork.com by project.
func MilestoneListByProject(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodMilestoneListByProject),
			Description: "List milestones in Teamwork.com by project. " + milestoneDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Milestones by Project",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "The ID of the project from which to retrieve milestones.",
					},
					"search_term": {
						Type: "string",
						Description: "A search term to filter milestones by name. " +
							"Each word from the search term is used to match against the milestone name and description. " +
							"The milestone will be selected if each word of the term matches the milestone name or description, not " +
							"requiring that the word matches are in the same field.",
					},
					"tag_ids": {
						Type:        "array",
						Description: "A list of tag IDs to filter milestones by tags",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
					"match_all_tags": {
						Type: "boolean",
						Description: "If true, the search will match milestones that have all the specified tags. " +
							"If false, the search will match milestones that have any of the specified tags. " +
							"Defaults to false.",
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
				Required: []string{"project_id"},
			},
			OutputSchema: milestoneListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneListRequest projects.MilestoneListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&milestoneListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&milestoneListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&milestoneListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&milestoneListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			milestoneList, err := projects.MilestoneList(ctx, engine, milestoneListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list milestones")
			}

			encoded, err := json.Marshal(milestoneList)
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
				StructuredContent: milestoneList,
			}, nil
		},
	}
}
