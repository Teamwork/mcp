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

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodMilestoneCreate)
	toolsets.RegisterMethod(MethodMilestoneUpdate)
	toolsets.RegisterMethod(MethodMilestoneDelete)
	toolsets.RegisterMethod(MethodMilestoneGet)
	toolsets.RegisterMethod(MethodMilestoneList)
	toolsets.RegisterMethod(MethodMilestoneListByProject)
}

// MilestoneCreate creates a milestone in Teamwork.com.
func MilestoneCreate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMilestoneCreate),
			mcp.WithDescription("Create a new milestone in Teamwork.com. "+milestoneDescription),
			mcp.WithTitleAnnotation("Create Milestone"),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description("The name of the milestone."),
			),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project to create the milestone in."),
			),
			mcp.WithString("description",
				mcp.Description("A description of the milestone."),
			),
			mcp.WithString("due_date",
				mcp.Required(),
				mcp.Description("The due date of the milestone in the format YYYYMMDD. This date will be used in all tasks "+
					"without a due date related to this milestone."),
			),
			mcp.WithObject("assignees",
				mcp.Required(),
				mcp.Description("An object containing assignees for the milestone. "+
					"MUST contain at least one of: user_ids, company_ids or team_ids with non-empty arrays."),
				mcp.Properties(map[string]any{
					"user_ids": map[string]any{
						"type":        "array",
						"description": "List of user IDs assigned to the milestone.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"company_ids": map[string]any{
						"type":        "array",
						"description": "List of company IDs assigned to the milestone.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"team_ids": map[string]any{
						"type":        "array",
						"description": "List of team IDs assigned to the milestone.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
				}),
				mcp.AdditionalProperties(false),
				func(schemaMap map[string]any) {
					schemaMap["minProperties"] = 1
					schemaMap["maxProperties"] = 3
					schemaMap["anyOf"] = []map[string]any{
						{"required": []string{"user_ids"}},
						{"required": []string{"company_ids"}},
						{"required": []string{"team_ids"}},
					}
				},
			),
			mcp.WithArray("tasklist_ids",
				mcp.Description("A list of tasklist IDs to associate with the milestone."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the milestone."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneCreateRequest projects.MilestoneCreateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&milestoneCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&milestoneCreateRequest.Name, "name"),
				helpers.OptionalPointerParam(&milestoneCreateRequest.Description, "description"),
				helpers.RequiredLegacyDateParam(&milestoneCreateRequest.DueAt, "due_date"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.TasklistIDs, "tasklist_ids"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			assignees, ok := request.GetArguments()["assignees"]
			if !ok {
				return nil, fmt.Errorf("missing required parameter: assignees")
			}
			assigneesMap, ok := assignees.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid assignees: expected an object, got %T", assignees)
			} else if assigneesMap == nil {
				return nil, fmt.Errorf("assignees cannot be null")
			}
			err = helpers.ParamGroup(assigneesMap,
				helpers.OptionalNumericListParam(&milestoneCreateRequest.Assignees.UserIDs, "user_ids"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.Assignees.CompanyIDs, "company_ids"),
				helpers.OptionalNumericListParam(&milestoneCreateRequest.Assignees.TeamIDs, "team_ids"),
			)
			if err != nil {
				return nil, fmt.Errorf("invalid assignees: %w", err)
			}
			if milestoneCreateRequest.Assignees.IsEmpty() {
				return nil, fmt.Errorf("at least one assignee must be provided")
			}

			milestone, err := projects.MilestoneCreate(ctx, engine, milestoneCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create milestone")
			}

			return mcp.NewToolResultText(fmt.Sprintf("Milestone created successfully with ID %d", milestone.ID)), nil
		},
	}
}

// MilestoneUpdate updates a milestone in Teamwork.com.
func MilestoneUpdate(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMilestoneUpdate),
			mcp.WithDescription("Update an existing milestone in Teamwork.com. "+milestoneDescription),
			mcp.WithTitleAnnotation("Update Milestone"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the milestone to update."),
			),
			mcp.WithString("name",
				mcp.Description("The name of the milestone."),
			),
			mcp.WithString("description",
				mcp.Description("A description of the milestone."),
			),
			mcp.WithString("due_date",
				mcp.Description("The due date of the milestone in the format YYYYMMDD. This date will be used in all tasks "+
					"without a due date related to this milestone."),
			),
			mcp.WithObject("assignees",
				mcp.Description("An object containing assignees for the milestone."),
				mcp.Properties(map[string]any{
					"user_ids": map[string]any{
						"type":        "array",
						"description": "List of user IDs assigned to the milestone.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"company_ids": map[string]any{
						"type":        "array",
						"description": "List of company IDs assigned to the milestone.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
					"team_ids": map[string]any{
						"type":        "array",
						"description": "List of team IDs assigned to the milestone.",
						"items":       map[string]any{"type": "integer"},
						"minItems":    1,
					},
				}),
				mcp.AdditionalProperties(false),
				func(schemaMap map[string]any) {
					schemaMap["minProperties"] = 1
					schemaMap["maxProperties"] = 3
					schemaMap["anyOf"] = []map[string]any{
						{"required": []string{"user_ids"}},
						{"required": []string{"company_ids"}},
						{"required": []string{"team_ids"}},
					}
				},
			),
			mcp.WithArray("tasklist_ids",
				mcp.Description("A list of tasklist IDs to associate with the milestone."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to associate with the milestone."),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneUpdateRequest projects.MilestoneUpdateRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&milestoneUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&milestoneUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&milestoneUpdateRequest.Description, "description"),
				helpers.OptionalLegacyDatePointerParam(&milestoneUpdateRequest.DueAt, "due_date"),
				helpers.OptionalNumericListParam(&milestoneUpdateRequest.TasklistIDs, "tasklist_ids"),
				helpers.OptionalNumericListParam(&milestoneUpdateRequest.TagIDs, "tag_ids"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			if assignees, ok := request.GetArguments()["assignees"]; ok {
				assigneesMap, ok := assignees.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid assignees")
				} else if assigneesMap != nil {
					milestoneUpdateRequest.Assignees = new(projects.LegacyUserGroups)
					err = helpers.ParamGroup(assigneesMap,
						helpers.OptionalNumericListParam(&milestoneUpdateRequest.Assignees.UserIDs, "user_ids"),
						helpers.OptionalNumericListParam(&milestoneUpdateRequest.Assignees.CompanyIDs, "company_ids"),
						helpers.OptionalNumericListParam(&milestoneUpdateRequest.Assignees.TeamIDs, "team_ids"),
					)
					if err != nil {
						return nil, fmt.Errorf("invalid assignees: %w", err)
					}
				}
			}

			_, err = projects.MilestoneUpdate(ctx, engine, milestoneUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update milestone")
			}

			return mcp.NewToolResultText("Milestone updated successfully"), nil
		},
	}
}

// MilestoneDelete deletes a milestone in Teamwork.com.
func MilestoneDelete(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMilestoneDelete),
			mcp.WithDescription("Delete an existing milestone in Teamwork.com. "+milestoneDescription),
			mcp.WithTitleAnnotation("Delete Milestone"),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the milestone to delete."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneDeleteRequest projects.MilestoneDeleteRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&milestoneDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			_, err = projects.MilestoneDelete(ctx, engine, milestoneDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete milestone")
			}

			return mcp.NewToolResultText("Milestone deleted successfully"), nil
		},
	}
}

// MilestoneGet retrieves a milestone in Teamwork.com.
func MilestoneGet(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMilestoneGet),
			mcp.WithDescription("Get an existing milestone in Teamwork.com. "+milestoneDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("Get Milestone"),
			mcp.WithOutputSchema[projects.MilestoneGetResponse](),
			mcp.WithNumber("id",
				mcp.Required(),
				mcp.Description("The ID of the milestone to get."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneGetRequest projects.MilestoneGetRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&milestoneGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			milestone, err := projects.MilestoneGet(ctx, engine, milestoneGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get milestone")
			}

			encoded, err := json.Marshal(milestone)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/milestones"),
			))), nil
		},
	}
}

// MilestoneList lists milestones in Teamwork.com.
func MilestoneList(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMilestoneList),
			mcp.WithDescription("List milestones in Teamwork.com. "+milestoneDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Milestones"),
			mcp.WithOutputSchema[projects.MilestoneListResponse](),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter milestones by name. "+
					"Each word from the search term is used to match against the milestone name and description. "+
					"The milestone will be selected if each word of the term matches the milestone name or description, not "+
					"requiring that the word matches are in the same field."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter milestones by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match milestones that have all the specified tags. "+
					"If false, the search will match milestones that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneListRequest projects.MilestoneListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.OptionalParam(&milestoneListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&milestoneListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&milestoneListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			milestoneList, err := projects.MilestoneList(ctx, engine, milestoneListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list milestones")
			}

			encoded, err := json.Marshal(milestoneList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/milestones"),
			))), nil
		},
	}
}

// MilestoneListByProject lists milestones in Teamwork.com by project.
func MilestoneListByProject(engine *twapi.Engine) server.ServerTool {
	return server.ServerTool{
		Tool: mcp.NewTool(string(MethodMilestoneListByProject),
			mcp.WithDescription("List milestones in Teamwork.com by project. "+milestoneDescription),
			mcp.WithToolAnnotation(mcp.ToolAnnotation{
				ReadOnlyHint: twapi.Ptr(true),
			}),
			mcp.WithTitleAnnotation("List Milestones by Project"),
			mcp.WithOutputSchema[projects.MilestoneListResponse](),
			mcp.WithNumber("project_id",
				mcp.Required(),
				mcp.Description("The ID of the project from which to retrieve milestones."),
			),
			mcp.WithString("search_term",
				mcp.Description("A search term to filter milestones by name. "+
					"Each word from the search term is used to match against the milestone name and description. "+
					"The milestone will be selected if each word of the term matches the milestone name or description, not "+
					"requiring that the word matches are in the same field."),
			),
			mcp.WithArray("tag_ids",
				mcp.Description("A list of tag IDs to filter milestones by tags"),
				mcp.Items(map[string]any{
					"type": "integer",
				}),
			),
			mcp.WithBoolean("match_all_tags",
				mcp.Description("If true, the search will match milestones that have all the specified tags. "+
					"If false, the search will match milestones that have any of the specified tags. "+
					"Defaults to false."),
			),
			mcp.WithNumber("page",
				mcp.Description("Page number for pagination of results."),
			),
			mcp.WithNumber("page_size",
				mcp.Description("Number of results per page for pagination."),
			),
		),
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var milestoneListRequest projects.MilestoneListRequest

			err := helpers.ParamGroup(request.GetArguments(),
				helpers.RequiredNumericParam(&milestoneListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&milestoneListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&milestoneListRequest.Filters.TagIDs, "tag_ids"),
				helpers.OptionalPointerParam(&milestoneListRequest.Filters.MatchAllTags, "match_all_tags"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&milestoneListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return mcp.NewToolResultErrorFromErr("invalid parameters", err), nil
			}

			milestoneList, err := projects.MilestoneList(ctx, engine, milestoneListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list milestones")
			}

			encoded, err := json.Marshal(milestoneList)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(string(helpers.WebLinker(ctx, encoded,
				helpers.WebLinkerWithIDPathBuilder("/app/milestones"),
			))), nil
		},
	}
}
