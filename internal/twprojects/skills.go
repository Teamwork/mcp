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
	MethodSkillCreate toolsets.Method = "twprojects-create_skill"
	MethodSkillUpdate toolsets.Method = "twprojects-update_skill"
	MethodSkillDelete toolsets.Method = "twprojects-delete_skill"
	MethodSkillGet    toolsets.Method = "twprojects-get_skill"
	MethodSkillList   toolsets.Method = "twprojects-list_skills"
)

const skillDescription = "Skill represents a specific capability, area of expertise, or proficiency that can be " +
	"assigned to users to describe what they are good at or qualified to work on. Skills help teams understand the " +
	"strengths available across the organization and make it easier to match the right skills to the right work when " +
	"planning projects, assigning tasks, or managing resources. By associating skills with users and leveraging them " +
	"in planning and reporting, Teamwork enables more effective workload distribution, better project outcomes, and " +
	"clearer visibility into whether the team has the capabilities needed to deliver upcoming work."

var (
	skillGetOutputSchema  *jsonschema.Schema
	skillListOutputSchema *jsonschema.Schema
)

func init() {
	// register the toolset methods
	toolsets.RegisterMethod(MethodSkillCreate)
	toolsets.RegisterMethod(MethodSkillUpdate)
	toolsets.RegisterMethod(MethodSkillDelete)
	toolsets.RegisterMethod(MethodSkillGet)
	toolsets.RegisterMethod(MethodSkillList)

	var err error

	// generate the output schemas only once
	skillGetOutputSchema, err = jsonschema.For[projects.SkillGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for SkillGetResponse: %v", err))
	}
	skillListOutputSchema, err = jsonschema.For[projects.SkillListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for SkillListResponse: %v", err))
	}
}

// SkillCreate creates a skill in Teamwork.com.
func SkillCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodSkillCreate),
			Description: "Create a new skill in Teamwork.com. " + skillDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Skill",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the skill.",
					},
					"user_ids": {
						Type:        "array",
						Description: "The user IDs associated with the skill.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var skillCreateRequest projects.SkillCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&skillCreateRequest.Name, "name"),
				helpers.OptionalNumericListParam(&skillCreateRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			skillResponse, err := projects.SkillCreate(ctx, engine, skillCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create skill")
			}
			return helpers.NewToolResultText("Skill created successfully with ID %d", skillResponse.Skill.ID), nil
		},
	}
}

// SkillUpdate updates a skill in Teamwork.com.
func SkillUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodSkillUpdate),
			Description: "Update an existing skill in Teamwork.com. " + skillDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Skill",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the skill to update.",
					},
					"name": {
						Type:        "string",
						Description: "The name of the skill.",
					},
					"user_ids": {
						Type:        "array",
						Description: "The user IDs associated with the skill.",
						Items: &jsonschema.Schema{
							Type: "integer",
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var skillUpdateRequest projects.SkillUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&skillUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&skillUpdateRequest.Name, "name"),
				helpers.OptionalNumericListParam(&skillUpdateRequest.UserIDs, "user_ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.SkillUpdate(ctx, engine, skillUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update skill")
			}
			return helpers.NewToolResultText("Skill updated successfully"), nil
		},
	}
}

// SkillDelete deletes a skill in Teamwork.com.
func SkillDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodSkillDelete),
			Description: "Delete an existing skill in Teamwork.com. " + skillDescription,
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Skill",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the skill to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var skillDeleteRequest projects.SkillDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&skillDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			_, err = projects.SkillDelete(ctx, engine, skillDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete skill")
			}
			return helpers.NewToolResultText("Skill deleted successfully"), nil
		},
	}
}

// SkillGet retrieves a skill in Teamwork.com.
func SkillGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodSkillGet),
			Description: "Get an existing skill in Teamwork.com. " + skillDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Skill",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the skill to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: skillGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var skillGetRequest projects.SkillGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&skillGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			skill, err := projects.SkillGet(ctx, engine, skillGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get skill")
			}

			encoded, err := json.Marshal(skill)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(encoded),
					},
				},
				StructuredContent: skill,
			}, nil
		},
	}
}

// SkillList lists skills in Teamwork.com.
func SkillList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodSkillList),
			Description: "List skills in Teamwork.com. " + skillDescription,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Skills",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Type: "string",
						Description: "A search term to filter skills by name, or assigned users. " +
							"The skill will be selected if each word of the term matches the name, or assigned user first or last " +
							"name, not requiring that the word matches are in the same field.",
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
			OutputSchema: skillListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var skillListRequest projects.SkillListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("failed to decode request: %s", err.Error())), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&skillListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericParam(&skillListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&skillListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError(fmt.Sprintf("invalid parameters: %s", err.Error())), nil
			}

			skillList, err := projects.SkillList(ctx, engine, skillListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list skills")
			}

			encoded, err := json.Marshal(skillList)
			if err != nil {
				return nil, err
			}
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{
						Text: string(helpers.WebLinker(ctx, encoded,
							helpers.WebLinkerWithIDPathBuilder("/app/people"),
						)),
					},
				},
				StructuredContent: skillList,
			}, nil
		},
	}
}
