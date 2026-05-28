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

const (
	MethodCustomItemFieldCreate toolsets.Method = "twprojects-create_custom_item_field"
	MethodCustomItemFieldUpdate toolsets.Method = "twprojects-update_custom_item_field"
	MethodCustomItemFieldDelete toolsets.Method = "twprojects-delete_custom_item_field"
	MethodCustomItemFieldGet    toolsets.Method = "twprojects-get_custom_item_field"
	MethodCustomItemFieldList   toolsets.Method = "twprojects-list_custom_item_fields"
)

var (
	customItemFieldGetOutputSchema  *jsonschema.Schema
	customItemFieldListOutputSchema *jsonschema.Schema
)

func init() {
	var err error
	customItemFieldGetOutputSchema, err = jsonschema.For[projects.CustomItemFieldGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomItemFieldGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customItemFieldGetOutputSchema)
	customItemFieldListOutputSchema, err = jsonschema.For[projects.CustomItemFieldListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomItemFieldListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customItemFieldListOutputSchema)
}

// CustomItemFieldCreate adds a field (column) to a custom item type.
func CustomItemFieldCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemFieldCreate),
			Description: "Add a field (column) to a custom item type. " +
				"Field types include text, number, dropdown, multiselect, checkbox, url, user, date, time and datetime." +
				customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Custom Item Field",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID that will own the new field.",
					},
					"display_name": {
						Type:        "string",
						Description: "Human-readable name of the field (e.g. \"Status\").",
					},
					"type": {
						Type:        "string",
						Description: "Field data type.",
						Enum: []any{
							"text-short", "text-long",
							"number-decimal", "number-integer",
							"dropdown", "multiselect",
							"checkbox", "url",
							"user",
							"date", "time", "datetime",
						},
					},
					"tw_type": {
						Description: "Optional sub-classification for dropdown fields. Use \"status\" for a Status field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"status"}},
							{Type: "null"},
						},
					},
					"definition": {
						Description: "Optional type-specific configuration as a JSON object. Examples: " +
							"for number-decimal fields {\"precision\": \"2\", \"unit\": {\"type\": \"currency\"}}; " +
							"for user fields {\"limit\": 1, \"source\": \"workspace\"}. " +
							"See the API docs for the exact shape per type.",
						AnyOf: []*jsonschema.Schema{
							{Type: "object"},
							{Type: "null"},
						},
					},
					"options": {
						Description: "Choices for dropdown/multiselect fields. Each option is " +
							"{label: \"Active\", color: \"#22c55e\"}.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "object",
									Properties: map[string]*jsonschema.Schema{
										"label": {Type: "string"},
										"color": {Type: "string"},
									},
									Required: []string{"label"},
								},
							},
							{Type: "null"},
						},
					},
					"position_after_id": {
						Description: "Place this field after the given field ID. Null appends to the end.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"custom_item_id", "display_name", "type"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var req projects.CustomItemFieldCreateRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.RequiredParam(&req.DisplayName, "display_name"),
				helpers.RequiredParam(&req.Type, "type",
					helpers.RestrictValues(
						projects.CustomItemFieldTypeTextShort,
						projects.CustomItemFieldTypeTextLong,
						projects.CustomItemFieldTypeNumberDecimal,
						projects.CustomItemFieldTypeNumberInteger,
						projects.CustomItemFieldTypeDropdown,
						projects.CustomItemFieldTypeMultiselect,
						projects.CustomItemFieldTypeCheckbox,
						projects.CustomItemFieldTypeURL,
						projects.CustomItemFieldTypeUser,
						projects.CustomItemFieldTypeDate,
						projects.CustomItemFieldTypeTime,
						projects.CustomItemFieldTypeDateTime,
					),
				),
				helpers.OptionalPointerParam(&req.TwType, "tw_type",
					helpers.RestrictValues(projects.CustomItemFieldTwTypeStatus),
				),
				helpers.OptionalNumericPointerParam(&req.PositionAfterID, "position_after_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if rawDef, ok := arguments["definition"]; ok && rawDef != nil {
				def, ok := rawDef.(map[string]any)
				if !ok {
					return helpers.NewToolResultTextError(
						"invalid parameters: definition must be an object"), nil
				}
				req.Definition = def
			}

			if rawOpts, ok := arguments["options"]; ok && rawOpts != nil {
				options, perr := parseCustomItemFieldOptionInputs(rawOpts)
				if perr != nil {
					return helpers.NewToolResultTextError("invalid parameters: %s", perr.Error()), nil
				}
				req.Options = options
			}

			resp, err := projects.CustomItemFieldCreate(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create custom item field")
			}
			invalidateCustomItemFieldCache(ctx, req.Path.CustomItemID)
			return helpers.NewToolResultText(
				"Custom item field created successfully with ID %d", resp.CustomItemField.ID,
			), nil
		},
	}
}

// CustomItemFieldUpdate updates a custom item field's display name,
// definition or position.
func CustomItemFieldUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomItemFieldUpdate),
			Description: "Update a field on a custom item type." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Custom Item Field",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the field belongs to.",
					},
					"id": {
						Type:        "integer",
						Description: "Field ID to update.",
					},
					"display_name": {
						Description: "New display name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"definition": {
						Description: "Replacement type-specific configuration as a JSON object.",
						AnyOf: []*jsonschema.Schema{
							{Type: "object"},
							{Type: "null"},
						},
					},
					"position_after_id": {
						Description: "Move this field after the given field ID.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"custom_item_id", "id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var req projects.CustomItemFieldUpdateRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.RequiredNumericParam(&req.Path.ID, "id"),
				helpers.OptionalPointerParam(&req.DisplayName, "display_name"),
				helpers.OptionalNumericPointerParam(&req.PositionAfterID, "position_after_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if rawDef, ok := arguments["definition"]; ok && rawDef != nil {
				def, ok := rawDef.(map[string]any)
				if !ok {
					return helpers.NewToolResultTextError(
						"invalid parameters: definition must be an object"), nil
				}
				req.Definition = def
			}

			_, err = projects.CustomItemFieldUpdate(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update custom item field")
			}
			invalidateCustomItemFieldCache(ctx, req.Path.CustomItemID)
			return helpers.NewToolResultText("Custom item field updated successfully"), nil
		},
	}
}

// CustomItemFieldDelete deletes a field from a custom item type.
func CustomItemFieldDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemFieldDelete),
			Description: "Delete a field from a custom item type. " +
				"Existing records on the type lose their value for this field." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Custom Item Field",
				DestructiveHint: new(true),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the field belongs to.",
					},
					"id": {
						Type:        "integer",
						Description: "Field ID to delete.",
					},
				},
				Required: []string{"custom_item_id", "id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var req projects.CustomItemFieldDeleteRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.RequiredNumericParam(&req.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CustomItemFieldDelete(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete custom item field")
			}
			invalidateCustomItemFieldCache(ctx, req.Path.CustomItemID)
			return helpers.NewToolResultText("Custom item field deleted successfully"), nil
		},
	}
}

// CustomItemFieldGet retrieves a single field on a custom item type.
func CustomItemFieldGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomItemFieldGet),
			Description: "Get a single field on a custom item type." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Custom Item Field",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the field belongs to.",
					},
					"id": {
						Type:        "integer",
						Description: "Field ID.",
					},
				},
				Required: []string{"custom_item_id", "id"},
			},
			OutputSchema: customItemFieldGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var req projects.CustomItemFieldGetRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.RequiredNumericParam(&req.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			resp, err := projects.CustomItemFieldGet(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get custom item field")
			}
			return helpers.NewToolResultJSON(resp)
		},
	}
}

// CustomItemFieldList lists the fields on a custom item type.
func CustomItemFieldList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemFieldList),
			Description: "List fields on a custom item type. " +
				"Each entry includes the twId you need when writing record values." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Custom Item Fields",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID to list fields for.",
					},
					"search_term": helpers.SearchTermSchema("custom item fields", "display name"),
					"ids": {
						Description: "Restrict to these field IDs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"show_deleted": {
						Description: "Include deleted fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"order_mode": helpers.OrderDirectionSchema(),
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
				},
				Required: []string{"custom_item_id"},
			},
			OutputSchema: helpers.WithOptionalFields(customItemFieldListOutputSchema),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			req := projects.NewCustomItemFieldListRequest(0)
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.OptionalParam(&req.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&req.Filters.IDs, "ids"),
				helpers.OptionalPointerParam(&req.Filters.ShowDeleted, "show_deleted"),
				helpers.OptionalParam(&req.Filters.OrderMode, "order_mode",
					helpers.RestrictValues(twapi.OrderModeAscending, twapi.OrderModeDescending),
				),
				helpers.OptionalNumericParam(&req.Filters.Page, "page"),
				helpers.OptionalNumericParam(&req.Filters.PageSize, "page_size"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if req.Filters.PageSize <= 0 {
				req.Filters.PageSize = 50
			} else if req.Filters.PageSize > 100 {
				req.Filters.PageSize = 100
			}

			resp, err := projects.CustomItemFieldList(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list custom item fields")
			}
			return helpers.NewToolResultJSON(resp)
		},
	}
}

// parseCustomItemFieldOptionInputs turns the raw "options" argument into the
// SDK's []CustomItemFieldOptionInput. Each item must be an object with at
// least a "label" string; optional "color" is hex without the leading hash.
func parseCustomItemFieldOptionInputs(raw any) ([]projects.CustomItemFieldOptionInput, error) {
	array, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("options must be an array")
	}
	result := make([]projects.CustomItemFieldOptionInput, 0, len(array))
	for i, item := range array {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("options[%d] must be an object", i)
		}
		var entry projects.CustomItemFieldOptionInput
		if rawLabel, ok := obj["label"]; ok {
			label, ok := rawLabel.(string)
			if !ok {
				return nil, fmt.Errorf("options[%d].label must be a string", i)
			}
			entry.Label = &label
		} else {
			return nil, fmt.Errorf("options[%d].label is required", i)
		}
		if rawColor, ok := obj["color"]; ok && rawColor != nil {
			color, ok := rawColor.(string)
			if !ok {
				return nil, fmt.Errorf("options[%d].color must be a string", i)
			}
			entry.Color = &color
		}
		result = append(result, entry)
	}
	return result, nil
}
