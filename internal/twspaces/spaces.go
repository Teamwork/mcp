package twspaces

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	spacesmodels "github.com/teamwork/spacessdkgo/models"
)

// List of methods available in the Teamwork Spaces MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodSpaceCreate        toolsets.Method = "twspaces-create_space"
	MethodSpaceUpdate        toolsets.Method = "twspaces-update_space"
	MethodSpaceDelete        toolsets.Method = "twspaces-delete_space"
	MethodSpaceGet           toolsets.Method = "twspaces-get_space"
	MethodSpaceList          toolsets.Method = "twspaces-list_spaces"
	MethodSpaceCollaborators toolsets.Method = "twspaces-list_space_collaborators"
)

// SpaceGet retrieves a single space by ID.
func SpaceGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSpaceGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Space",
				ReadOnlyHint: true,
			},
			Description: "Retrieve detailed information about a specific space in Teamwork Spaces by its ID. " +
				"Useful for inspecting space configuration, metadata, or linking spaces to projects.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the space to retrieve.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			space, err := client.Spaces.Get(ctx, int64(arguments.GetInt("id", 0)))
			if err != nil {
				return nil, fmt.Errorf("failed to get space: %w", err)
			}
			return helpers.NewToolResultJSON(space)
		},
	}
}

// SpaceList returns a list of spaces.
func SpaceList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSpaceList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Spaces",
				ReadOnlyHint: true,
			},
			Description: "List all spaces in Teamwork Spaces. Enables users to discover, audit, or synchronize " +
				"space data for documentation management, reporting, or integration scenarios.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
		},
		Handler: func(ctx context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)

			spaces, err := client.Spaces.List(ctx, url.Values{})
			if err != nil {
				return nil, fmt.Errorf("failed to list spaces: %w", err)
			}
			return helpers.NewToolResultJSON(spaces)
		},
	}
}

// SpaceCreate creates a new space.
func SpaceCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSpaceCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Space",
			},
			Description: "Create a new space in Teamwork Spaces. Useful for setting up knowledge bases, " +
				"team wikis, or project documentation areas.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"title": {
						Type:        "string",
						Description: "The title of the space.",
					},
					"code": {
						Type:        "string",
						Description: "A short unique code/identifier for the space (e.g. \"ENG\", \"DOCS\").",
					},
					"purpose": {
						Type:        "string",
						Description: "A brief description of the space's purpose.",
					},
					"spaceColor": {
						Type:        "string",
						Description: "A hex color code for the space (e.g. \"#FF5733\").",
					},
					"icon": {
						Type:        "string",
						Description: "An icon identifier for the space.",
					},
					"projectId": {
						Type:        "integer",
						Description: "The ID of a Teamwork project to link to this space.",
					},
					"categoryId": {
						Type: "integer",
						Description: "The ID of the category to assign to this space. " +
							"Use the 'twspaces-list_categories' tool to find valid IDs.",
					},
				},
				Required: []string{"title", "code"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.SpaceCreate{
				Title: arguments.GetString("title", ""),
				Code:  arguments.GetString("code", ""),
			}

			if purpose := arguments.GetString("purpose", ""); purpose != "" {
				req.Purpose = &purpose
			}
			if color := arguments.GetString("spaceColor", ""); color != "" {
				req.SpaceColor = color
			}
			if icon := arguments.GetString("icon", ""); icon != "" {
				req.Icon = icon
			}
			if projectID := int64(arguments.GetInt("projectId", 0)); projectID > 0 {
				req.LinkedProjectID = &projectID
			}
			if categoryID := int64(arguments.GetInt("categoryId", 0)); categoryID > 0 {
				req.CategoryID = &categoryID
			}

			space, err := client.Spaces.Create(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("failed to create space: %w", err)
			}
			return helpers.NewToolResultJSON(space)
		},
	}
}

// SpaceUpdate updates an existing space.
func SpaceUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSpaceUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Space",
			},
			Description: "Update an existing space in Teamwork Spaces by ID. Supports changes to title, code, " +
				"purpose, color, icon, state, linked project, and category.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the space to update.",
					},
					"title": {
						Type:        "string",
						Description: "The new title of the space.",
					},
					"code": {
						Type:        "string",
						Description: "A new short unique code/identifier for the space.",
					},
					"purpose": {
						Type:        "string",
						Description: "A new brief description of the space's purpose.",
					},
					"spaceColor": {
						Type:        "string",
						Description: "A new hex color code for the space.",
					},
					"icon": {
						Type:        "string",
						Description: "A new icon identifier for the space.",
					},
					"state": {
						Type:        "string",
						Description: "The state of the space (e.g. \"active\", \"archived\").",
					},
					"projectId": {
						Type:        "integer",
						Description: "The ID of a Teamwork project to link to this space.",
					},
					"categoryId": {
						Type: "integer",
						Description: "The ID of the category to assign to this space. " +
							"Use the 'twspaces-list_categories' tool to find valid IDs.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.SpaceUpdate{}

			if title := arguments.GetString("title", ""); title != "" {
				req.Title = &title
			}
			if code := arguments.GetString("code", ""); code != "" {
				req.Code = &code
			}
			if purpose := arguments.GetString("purpose", ""); purpose != "" {
				req.Purpose = &purpose
			}
			if color := arguments.GetString("spaceColor", ""); color != "" {
				req.SpaceColor = &color
			}
			if icon := arguments.GetString("icon", ""); icon != "" {
				req.Icon = &icon
			}
			if state := arguments.GetString("state", ""); state != "" {
				req.State = &state
			}
			if projectID := int64(arguments.GetInt("projectId", 0)); projectID > 0 {
				req.LinkedProjectID = &projectID
			}
			if categoryID := int64(arguments.GetInt("categoryId", 0)); categoryID > 0 {
				req.CategoryID = &categoryID
			}

			space, err := client.Spaces.Update(ctx, int64(arguments.GetInt("id", 0)), req)
			if err != nil {
				return nil, fmt.Errorf("failed to update space: %w", err)
			}
			return helpers.NewToolResultJSON(space)
		},
	}
}

// SpaceDelete deletes a space by ID.
func SpaceDelete(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSpaceDelete),
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Space",
			},
			Description: "Delete a space in Teamwork Spaces by its ID. This action is irreversible and will " +
				"remove the space and all its content.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the space to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			if err := client.Spaces.Delete(ctx, int64(arguments.GetInt("id", 0))); err != nil {
				return nil, fmt.Errorf("failed to delete space: %w", err)
			}
			return helpers.NewToolResultText("Space deleted successfully"), nil
		},
	}
}

// SpaceCollaborators lists the collaborators for a space.
func SpaceCollaborators(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodSpaceCollaborators),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Space Collaborators",
				ReadOnlyHint: true,
			},
			Description: "List all collaborators (users and teams) for a specific space in Teamwork Spaces. " +
				"Useful for auditing access, reviewing permissions, or understanding who contributes to a space.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the space to retrieve collaborators for.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := clientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			collaborators, err := client.Spaces.Collaborators(ctx, int64(arguments.GetInt("id", 0)))
			if err != nil {
				return nil, fmt.Errorf("failed to get space collaborators: %w", err)
			}
			return helpers.NewToolResultJSON(collaborators)
		},
	}
}
