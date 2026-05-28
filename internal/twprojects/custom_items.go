package twprojects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// Custom items are user-defined entity types that customers add to a project
// (Contracts, Leads, Deals, Monkeys — anything the workspace needs that isn't
// a built-in Teamwork concept). Each custom item type owns its own fields
// (columns, see CustomItemField*) and records (rows, see CustomItemRecord*).
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCustomItemCreate toolsets.Method = "twprojects-create_custom_item"
	MethodCustomItemUpdate toolsets.Method = "twprojects-update_custom_item"
	MethodCustomItemDelete toolsets.Method = "twprojects-delete_custom_item"
	MethodCustomItemGet    toolsets.Method = "twprojects-get_custom_item"
	MethodCustomItemList   toolsets.Method = "twprojects-list_custom_items"
)

// customItemRoutingHint is appended to every custom-item tool description so
// the LLM picks these tools when the user mentions an entity that isn't a
// built-in Teamwork concept. The same hint appears on field and record tools.
const customItemRoutingHint = " Custom items are user-defined entity types — Contracts, Leads, Deals, " +
	"or anything else a customer has set up on a project. Use these tools when the user refers to an " +
	"entity that is NOT a built-in Teamwork concept (Task, Tasklist, Project, Milestone, Comment, " +
	"Notebook, Company, Team, User, Tag). If you don't recognise an entity name in the user's request, " +
	"assume it is a custom item and call twprojects-list_custom_items on the relevant project to confirm."

var (
	customItemGetOutputSchema  *jsonschema.Schema
	customItemListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	customItemGetOutputSchema, err = jsonschema.For[projects.CustomItemGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomItemGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customItemGetOutputSchema)
	customItemListOutputSchema, err = jsonschema.For[projects.CustomItemListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomItemListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customItemListOutputSchema)
}

// CustomItemCreate creates a custom item type on a project in Teamwork.com.
func CustomItemCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemCreate),
			Description: "Create a new custom item type (e.g. Contracts, Leads, Deals) on a project." +
				customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Custom Item",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "Project ID that will own the new custom item type.",
					},
					"display_name": {
						Type:        "string",
						Description: "The display name of the custom item type (e.g. \"Contracts\").",
					},
					"description": {
						Description: "An optional human-readable description for the custom item type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"label_singular": {
						Description: "Singular label for one record (e.g. \"Contract\"). " +
							"Defaults to the display name when omitted.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"label_plural": {
						Description: "Plural label for many records (e.g. \"Contracts\"). " +
							"Defaults to the display name when omitted.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"project_id", "display_name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemCreateRequest projects.CustomItemCreateRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemCreateRequest.Path.ProjectID, "project_id"),
				helpers.RequiredParam(&customItemCreateRequest.DisplayName, "display_name"),
				helpers.OptionalPointerParam(&customItemCreateRequest.Description, "description"),
				helpers.OptionalPointerParam(&customItemCreateRequest.LabelSingular, "label_singular"),
				helpers.OptionalPointerParam(&customItemCreateRequest.LabelPlural, "label_plural"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			customItemResponse, err := projects.CustomItemCreate(ctx, engine, customItemCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create custom item")
			}
			return helpers.NewToolResultText(
				"Custom item created successfully with ID %d", customItemResponse.CustomItem.ID,
			), nil
		},
	}
}

// CustomItemUpdate updates a custom item type in Teamwork.com.
func CustomItemUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomItemUpdate),
			Description: "Update a custom item type's display name, description, or labels." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Custom Item",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "Custom item type ID to update.",
					},
					"display_name": {
						Description: "New display name for the custom item type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "New description for the custom item type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"label_singular": {
						Description: "New singular label.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"label_plural": {
						Description: "New plural label.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemUpdateRequest projects.CustomItemUpdateRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&customItemUpdateRequest.DisplayName, "display_name"),
				helpers.OptionalPointerParam(&customItemUpdateRequest.Description, "description"),
				helpers.OptionalPointerParam(&customItemUpdateRequest.LabelSingular, "label_singular"),
				helpers.OptionalPointerParam(&customItemUpdateRequest.LabelPlural, "label_plural"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CustomItemUpdate(ctx, engine, customItemUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update custom item")
			}
			return helpers.NewToolResultText("Custom item updated successfully"), nil
		},
	}
}

// CustomItemDelete deletes a custom item type in Teamwork.com.
func CustomItemDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemDelete),
			Description: "Delete a custom item type. This also deletes all records and fields under it." +
				customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Custom Item",
				DestructiveHint: new(true),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "Custom item type ID to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemDeleteRequest projects.CustomItemDeleteRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Invalidate any cached field schema for this type before the
			// delete so a subsequent operation on a different type with the
			// same id (extremely unlikely, but cheap) starts clean.
			invalidateFieldSchemaCache(customItemDeleteRequest.Path.ID)

			_, err = projects.CustomItemDelete(ctx, engine, customItemDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete custom item")
			}
			return helpers.NewToolResultText("Custom item deleted successfully"), nil
		},
	}
}

// CustomItemGet retrieves a custom item type in Teamwork.com. The response
// includes the type's fields and sections so the caller has the full schema
// in a single call.
func CustomItemGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemGet),
			Description: "Get a custom item type with its fields and sections inline, so you can see " +
				"its schema before creating or updating records." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Custom Item",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "Custom item type ID to retrieve.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: customItemGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			customItemGetRequest := projects.NewCustomItemGetRequest(0)
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Auto-include fields + sections — see plan §3 "Schema discovery
			// on the type itself". This gives the LLM everything it needs to
			// write a record in one round-trip.
			customItemGetRequest.Include = []string{"customItemFields", "customItemSections"}

			customItem, err := projects.CustomItemGet(ctx, engine, customItemGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get custom item")
			}
			return helpers.NewToolResultJSON(customItem)
		},
	}
}

// CustomItemList lists the custom item types on a project in Teamwork.com.
func CustomItemList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemList),
			Description: "List the custom item types defined on a project. Returns each type's id, " +
				"display name and labels — call get_custom_item to see a type's fields and sections." +
				customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Custom Items",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project_id": {
						Type:        "integer",
						Description: "Project ID to list custom item types for.",
					},
					"search_term": helpers.SearchTermSchema("custom items", "display name or labels"),
					"ids": {
						Description: "Restrict to these custom item type IDs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"show_deleted": {
						Description: "Include deleted custom item types.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"order_by": {
						Description: "Field to sort by.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"name"}},
							{Type: "null"},
						},
					},
					"order_mode": helpers.OrderDirectionSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
				},
				Required: []string{"project_id"},
			},
			OutputSchema: helpers.WithOptionalFields(customItemListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			customItemListRequest := projects.NewCustomItemListRequest(0)
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemListRequest.Path.ProjectID, "project_id"),
				helpers.OptionalParam(&customItemListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&customItemListRequest.Filters.IDs, "ids"),
				helpers.OptionalPointerParam(&customItemListRequest.Filters.ShowDeleted, "show_deleted"),
				helpers.OptionalParam(&customItemListRequest.Filters.OrderBy, "order_by",
					helpers.RestrictValues("name"),
				),
				helpers.OptionalParam(&customItemListRequest.Filters.OrderMode, "order_mode",
					helpers.RestrictValues(twapi.OrderModeAscending, twapi.OrderModeDescending),
				),
				helpers.OptionalNumericParam(&customItemListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&customItemListRequest.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Clamp page size to keep the list cheap — see plan §6 perf rules.
			if customItemListRequest.Filters.PageSize <= 0 {
				customItemListRequest.Filters.PageSize = 50
			} else if customItemListRequest.Filters.PageSize > 100 {
				customItemListRequest.Filters.PageSize = 100
			}

			customItems, err := projects.CustomItemList(ctx, engine, customItemListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list custom items")
			}
			return helpers.NewToolResultJSON(customItems)
		},
	}
}
