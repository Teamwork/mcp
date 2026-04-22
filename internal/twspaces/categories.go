package twspaces

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	spacesmodels "github.com/teamwork/spacessdkgo/models"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
)

// List of category methods available in the Teamwork Spaces MCP service.
const (
	MethodCategoryCreate toolsets.Method = "twspaces-create_category"
	MethodCategoryUpdate toolsets.Method = "twspaces-update_category"
	MethodCategoryDelete toolsets.Method = "twspaces-delete_category"
	MethodCategoryGet    toolsets.Method = "twspaces-get_category"
	MethodCategoryList   toolsets.Method = "twspaces-list_categories"
)

// CategoryGet retrieves a single category by ID.
func CategoryGet(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCategoryGet),
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Category",
				ReadOnlyHint: true,
			},
			Description: "Retrieve a specific space category in Teamwork Spaces by its ID.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the category to retrieve.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			category, err := client.Categories.Get(ctx, int64(arguments.GetInt("id", 0)))
			if err != nil {
				return nil, fmt.Errorf("failed to get category: %w", err)
			}
			return helpers.NewToolResultJSON(category)
		},
	}
}

// CategoryList lists all categories.
func CategoryList(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCategoryList),
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Categories",
				ReadOnlyHint: true,
			},
			Description: "List all space categories in Teamwork Spaces. Categories are used to organize spaces " +
				"into logical groups for easier navigation and management.",
			InputSchema: &jsonschema.Schema{
				Type:       "object",
				Properties: map[string]*jsonschema.Schema{},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)

			categories, err := client.Categories.List(ctx, url.Values{})
			if err != nil {
				return nil, fmt.Errorf("failed to list categories: %w", err)
			}
			return helpers.NewToolResultJSON(categories)
		},
	}
}

// CategoryCreate creates a new category.
func CategoryCreate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCategoryCreate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Category",
			},
			Description: "Create a new space category in Teamwork Spaces. Categories help organize spaces into " +
				"logical groups for easier navigation and management.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The name of the category.",
					},
					"color": {
						Type:        "string",
						Description: "A hex color code for the category (e.g. \"#FF5733\").",
					},
				},
				Required: []string{"name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.CategoryCreate{
				Name: arguments.GetString("name", ""),
			}
			if color := arguments.GetString("color", ""); color != "" {
				req.Color = &color
			}

			category, err := client.Categories.Create(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("failed to create category: %w", err)
			}
			return helpers.NewToolResultJSON(category)
		},
	}
}

// CategoryUpdate updates an existing category.
func CategoryUpdate(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCategoryUpdate),
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Category",
			},
			Description: "Update an existing space category in Teamwork Spaces by ID.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the category to update.",
					},
					"name": {
						Type:        "string",
						Description: "The new name for the category.",
					},
					"color": {
						Type:        "string",
						Description: "A new hex color code for the category.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			req := &spacesmodels.CategoryUpdate{}

			if name := arguments.GetString("name", ""); name != "" {
				req.Name = &name
			}
			if color := arguments.GetString("color", ""); color != "" {
				req.Color = &color
			}

			category, err := client.Categories.Update(ctx, int64(arguments.GetInt("id", 0)), req)
			if err != nil {
				return nil, fmt.Errorf("failed to update category: %w", err)
			}
			return helpers.NewToolResultJSON(category)
		},
	}
}

// CategoryDelete deletes a category by ID.
func CategoryDelete(httpClient *http.Client) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCategoryDelete),
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Category",
			},
			Description: "Delete a space category in Teamwork Spaces by its ID.",
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the category to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			client := ClientFromContext(ctx, httpClient)
			arguments, err := helpers.NewToolArguments(request)
			if err != nil {
				return helpers.NewToolResultTextError("%v", err), nil
			}

			if err := client.Categories.Delete(ctx, int64(arguments.GetInt("id", 0))); err != nil {
				return nil, fmt.Errorf("failed to delete category: %w", err)
			}
			return helpers.NewToolResultText("Category deleted successfully"), nil
		},
	}
}
