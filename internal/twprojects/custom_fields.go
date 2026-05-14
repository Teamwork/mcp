package twprojects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/helpers"
	"github.com/teamwork/mcp/internal/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCustomFieldCreate toolsets.Method = "twprojects-create_custom_field"
	MethodCustomFieldUpdate toolsets.Method = "twprojects-update_custom_field"
	MethodCustomFieldDelete toolsets.Method = "twprojects-delete_custom_field"
	MethodCustomFieldGet    toolsets.Method = "twprojects-get_custom_field"
	MethodCustomFieldList   toolsets.Method = "twprojects-list_custom_fields"
)

var (
	customFieldGetOutputSchema  *jsonschema.Schema
	customFieldListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	customFieldGetOutputSchema, err = jsonschema.For[projects.CustomFieldGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomFieldGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customFieldGetOutputSchema)
	customFieldListOutputSchema, err = jsonschema.For[projects.CustomFieldListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomFieldListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customFieldListOutputSchema)
}

// CustomFieldCreate creates a custom field in Teamwork.com.
func CustomFieldCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldCreate),
			Description: "Create custom field.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Create Custom Field",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"name": {
						Type:        "string",
						Description: "The display name of the custom field.",
					},
					"type": {
						Type:        "string",
						Description: "The data type of the custom field.",
						Enum: []any{
							"text-short",
							"text-long",
							"number-decimal",
							"number-integer",
							"financial",
							"dropdown",
							"multiselect",
							"checkbox",
							"percentage",
							"url",
							"tw-url",
							"user",
							"rating",
							"date",
							"phone",
							"email",
							"status",
						},
					},
					"entity": {
						Type: "string",
						Description: "The type of entity this custom field can be applied to. " +
							"Use 'all' for installation-level custom fields that are available across the workspace.",
						Enum: []any{
							"all",
							"project",
							"task",
							"company",
						},
					},
					"description": {
						Description: "An optional description for the custom field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"required": {
						Description: "Whether the custom field must have a value when set on an entity.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"project_id": {
						Description: "The ID of the project to scope the custom field to. " +
							"When omitted, the custom field is created at the installation level.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"options": {
						Description: "Type-specific options for the custom field. " +
							`For 'dropdown' and 'multiselect' types, provide {"choices": [{"value": "...", "color": "#rrggbb"}, ...]}. ` +
							`For 'rating' type, provide {"icon": "star|heart|...", "color": "#rrggbb"}. ` +
							`For 'number-decimal' type, provide {"decimals": <int>}.`,
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"choices": {
										Type: "array",
										Items: &jsonschema.Schema{
											Type: "object",
											Properties: map[string]*jsonschema.Schema{
												"value": {
													Type: "string",
													Description: "The display value of the choice. For 'dropdown' fields, this is the value " +
														"that will be set on entities when this choice is selected.",
												},
												"color": {
													Type:        "string",
													Description: "The hex color code for the choice (e.g. #ff0000).",
												},
											},
										},
									},
								},
							},
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"icon": {
										Type:        "string",
										Description: "The icon to use for the rating, e.g. 'star', 'heart', etc.",
									},
									"color": {
										Type:        "string",
										Description: "The hex color code for the rating icons (e.g. #ff0000).",
									},
								},
							},
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"decimals": {
										Type:        "integer",
										Description: "The number of decimal places to allow for number-decimal custom fields.",
									},
								},
							},
							{Type: "null"},
						},
					},
					"formula": {
						Description: "The formula expression for 'formula' type custom fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"currency_code": {
						Description: "The ISO currency code for 'currency' or 'financial' type custom fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"unit": {
						Description: "The unit associated with the custom field, when applicable.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "string",
								Enum: []any{
									"currency",
									"duration",
									"date",
									"percent",
									"currency/duration",
									"duration/currency",
								},
							},
							{Type: "null"},
						},
					},
				},
				Required: []string{"name", "type", "entity"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var customFieldCreateRequest projects.CustomFieldCreateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&customFieldCreateRequest.Name, "name"),
				helpers.RequiredParam(&customFieldCreateRequest.Type, "type",
					helpers.RestrictValues(
						projects.CustomFieldTypeTextShort,
						projects.CustomFieldTypeTextLong,
						projects.CustomFieldTypeNumberDecimal,
						projects.CustomFieldTypeNumberInteger,
						projects.CustomFieldTypeFinancial,
						projects.CustomFieldTypeDropdown,
						projects.CustomFieldTypeMultiselect,
						projects.CustomFieldTypeCheckbox,
						projects.CustomFieldTypePercentage,
						projects.CustomFieldTypeURL,
						projects.CustomFieldTypeTeamworkURL,
						projects.CustomFieldTypeUser,
						projects.CustomFieldTypeRating,
						projects.CustomFieldTypeDate,
						projects.CustomFieldTypePhone,
						projects.CustomFieldTypeEmail,
						projects.CustomFieldTypeStatus,
					),
				),
				helpers.RequiredParam(&customFieldCreateRequest.Entity, "entity",
					helpers.RestrictValues(
						projects.CustomFieldEntityGlobal,
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityCompany,
						projects.CustomFieldEntityUser,
					),
				),
				helpers.OptionalPointerParam(&customFieldCreateRequest.Description, "description"),
				helpers.OptionalPointerParam(&customFieldCreateRequest.Required, "required"),
				helpers.OptionalNumericPointerParam(&customFieldCreateRequest.ProjectID, "project_id"),
				helpers.OptionalPointerParam(&customFieldCreateRequest.Formula, "formula"),
				helpers.OptionalPointerParam(&customFieldCreateRequest.CurrencyCode, "currency_code"),
				helpers.OptionalPointerParam(&customFieldCreateRequest.Unit, "unit",
					helpers.RestrictValues(
						projects.CustomFieldUnitCurrency,
						projects.CustomFieldUnitDuration,
						projects.CustomFieldUnitDate,
						projects.CustomFieldUnitPercent,
						projects.CustomFieldUnitCurrencyPerDuration,
						projects.CustomFieldUnitDurationPerCurrency,
					),
				),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			customFieldCreateRequest.Options, err = parseCustomFieldOptions(arguments)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			customFieldResponse, err := projects.CustomFieldCreate(ctx, engine, customFieldCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create custom field")
			}
			return helpers.NewToolResultText(
				"Custom field created successfully with ID %d", customFieldResponse.CustomField.ID,
			), nil
		},
	}
}

// CustomFieldUpdate updates a custom field in Teamwork.com.
func CustomFieldUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldUpdate),
			Description: "Update custom field.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Custom Field",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the custom field to update.",
					},
					"name": {
						Description: "The display name of the custom field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"description": {
						Description: "An optional description for the custom field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"required": {
						Description: "Whether the custom field must have a value when set on an entity.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"options": {
						Description: "Type-specific options for the custom field. " +
							`For 'dropdown' and 'multiselect' types, provide {"choices": [{"value": "...", "color": "#rrggbb"}, ...]}. ` +
							`For 'rating' type, provide {"icon": "star|heart|...", "color": "#rrggbb"}. ` +
							`For 'number-decimal' type, provide {"decimals": <int>}.`,
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"choices": {
										Type: "array",
										Items: &jsonschema.Schema{
											Type: "object",
											Properties: map[string]*jsonschema.Schema{
												"value": {
													Type: "string",
													Description: "The display value of the choice. For 'dropdown' fields, this is the value " +
														"that will be set on entities when this choice is selected.",
												},
												"color": {
													Type:        "string",
													Description: "The hex color code for the choice (e.g. #ff0000).",
												},
											},
										},
									},
								},
							},
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"icon": {
										Type:        "string",
										Description: "The icon to use for the rating, e.g. 'star', 'heart', etc.",
									},
									"color": {
										Type:        "string",
										Description: "The hex color code for the rating icons (e.g. #ff0000).",
									},
								},
							},
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"decimals": {
										Type:        "integer",
										Description: "The number of decimal places to allow for number-decimal custom fields.",
									},
								},
							},
							{Type: "null"},
						},
					},
					"formula": {
						Description: "The formula expression for 'formula' type custom fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"currency_code": {
						Description: "The ISO currency code for 'currency' or 'financial' type custom fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"unit": {
						Description: "The unit associated with the custom field, when applicable.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "string",
								Enum: []any{
									"currency",
									"duration",
									"date",
									"percent",
									"currency/duration",
									"duration/currency",
								},
							},
							{Type: "null"},
						},
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var customFieldUpdateRequest projects.CustomFieldUpdateRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customFieldUpdateRequest.Path.ID, "id"),
				helpers.OptionalPointerParam(&customFieldUpdateRequest.Name, "name"),
				helpers.OptionalPointerParam(&customFieldUpdateRequest.Description, "description"),
				helpers.OptionalPointerParam(&customFieldUpdateRequest.Required, "required"),
				helpers.OptionalPointerParam(&customFieldUpdateRequest.Formula, "formula"),
				helpers.OptionalPointerParam(&customFieldUpdateRequest.CurrencyCode, "currency_code"),
				helpers.OptionalPointerParam(&customFieldUpdateRequest.Unit, "unit",
					helpers.RestrictValues(
						projects.CustomFieldUnitCurrency,
						projects.CustomFieldUnitDuration,
						projects.CustomFieldUnitDate,
						projects.CustomFieldUnitPercent,
						projects.CustomFieldUnitCurrencyPerDuration,
						projects.CustomFieldUnitDurationPerCurrency,
					),
				),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			customFieldUpdateRequest.Options, err = parseCustomFieldOptions(arguments)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CustomFieldUpdate(ctx, engine, customFieldUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update custom field")
			}
			return helpers.NewToolResultText("Custom field updated successfully"), nil
		},
	}
}

// CustomFieldDelete deletes a custom field in Teamwork.com.
func CustomFieldDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldDelete),
			Description: "Delete custom field.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Custom Field",
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the custom field to delete.",
					},
				},
				Required: []string{"id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var customFieldDeleteRequest projects.CustomFieldDeleteRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customFieldDeleteRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CustomFieldDelete(ctx, engine, customFieldDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete custom field")
			}
			return helpers.NewToolResultText("Custom field deleted successfully"), nil
		},
	}
}

// CustomFieldGet retrieves a custom field in Teamwork.com.
func CustomFieldGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldGet),
			Description: "Get custom field.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "Get Custom Field",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"id": {
						Type:        "integer",
						Description: "The ID of the custom field to get.",
					},
				},
				Required: []string{"id"},
			},
			OutputSchema: customFieldGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var customFieldGetRequest projects.CustomFieldGetRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customFieldGetRequest.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			customField, err := projects.CustomFieldGet(ctx, engine, customFieldGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get custom field")
			}
			return helpers.NewToolResultJSON(customField)
		},
	}
}

// CustomFieldList lists custom fields in Teamwork.com.
func CustomFieldList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldList),
			Description: "List custom fields.",
			Annotations: &mcp.ToolAnnotations{
				Title:        "List Custom Fields",
				ReadOnlyHint: true,
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"search_term": {
						Description: "A search term to filter custom fields by name.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"ids": {
						Description: "A list of custom field IDs to retrieve.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"entities": {
						Description: "A list of entity types to filter custom fields by.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "string",
									Enum: []any{
										"project",
										"task",
										"company",
									},
								},
							},
							{Type: "null"},
						},
					},
					"project_ids": {
						Description: "A list of project IDs to filter custom fields by.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"only_site_level": {
						Description: "Whether to return only installation-level custom fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"only_project_level": {
						Description: "Whether to return only project-level custom fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"include_site_level": {
						Description: "Whether to also include installation-level custom fields when filtering by project.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"show_deleted": {
						Description: "Whether to include deleted custom fields in the results.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"order_by": {
						Description: "The field to sort the results by.",
						AnyOf: []*jsonschema.Schema{
							{
								Type: "string",
								Enum: []any{"name", "project", "dateCreated", "dateUpdated"},
							},
							{Type: "null"},
						},
					},
					"order_mode": {
						Description: "The direction to sort the results in.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Enum: []any{"asc", "desc"}},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{},
			},
			OutputSchema: customFieldListOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var customFieldListRequest projects.CustomFieldListRequest

			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			verbose := true
			err := helpers.ParamGroup(arguments,
				helpers.OptionalParam(&customFieldListRequest.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&customFieldListRequest.Filters.IDs, "ids"),
				helpers.OptionalListParam(&customFieldListRequest.Filters.Entities, "entities",
					helpers.RestrictValues(
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityCompany,
					),
				),
				helpers.OptionalNumericListParam(&customFieldListRequest.Filters.ProjectIDs, "project_ids"),
				helpers.OptionalPointerParam(&customFieldListRequest.Filters.OnlySiteLevel, "only_site_level"),
				helpers.OptionalPointerParam(&customFieldListRequest.Filters.OnlyProjectLevel, "only_project_level"),
				helpers.OptionalPointerParam(&customFieldListRequest.Filters.IncludeSiteLevel, "include_site_level"),
				helpers.OptionalPointerParam(&customFieldListRequest.Filters.ShowDeleted, "show_deleted"),
				helpers.OptionalParam(&customFieldListRequest.Filters.OrderBy, "order_by",
					helpers.RestrictValues(
						"name",
						"project",
						"dateCreated",
						"dateUpdated",
					),
				),
				helpers.OptionalParam(&customFieldListRequest.Filters.OrderMode, "order_mode",
					helpers.RestrictValues(
						twapi.OrderModeAscending,
						twapi.OrderModeDescending,
					),
				),
				helpers.OptionalNumericParam(&customFieldListRequest.Filters.Page, "page"),
				helpers.OptionalNumericParam(&customFieldListRequest.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			if !verbose {
				customFieldListRequest.Filters.Fields.CustomFields = []projects.CustomFieldField{
					projects.CustomFieldFieldID,
					projects.CustomFieldFieldName,
				}
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, customFieldListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list custom fields")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list custom fields"),
					"failed to list custom fields",
				)
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
			if verbose {
				var structured any
				if err := json.Unmarshal(body, &structured); err != nil {
					return nil, fmt.Errorf("failed to decode response: %w", err)
				}
				result.StructuredContent = structured
			}
			return result, nil
		},
	}
}

// parseCustomFieldOptions converts the raw options argument into a typed
// CustomFieldOptions value.
func parseCustomFieldOptions(arguments map[string]any) (projects.CustomFieldOptions, error) {
	raw, ok := arguments["options"]
	if !ok || raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	options := []projects.CustomFieldOptions{
		&projects.CustomFieldOptionsDropdown{},
		&projects.CustomFieldOptionsRating{},
		&projects.CustomFieldOptionsNumberDecimal{},
	}
	for _, option := range options {
		decoder := json.NewDecoder(bytes.NewBuffer(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(option); err == nil {
			// Successfully decoded into this options struct, return it
			return option, nil
		}
	}

	return nil, errors.New("options are not supported")
}
