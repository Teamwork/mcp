package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCustomFieldValueCreate      toolsets.Method = "twprojects-create_custom_field_value"
	MethodCustomFieldValueCreateBatch toolsets.Method = "twprojects-create_custom_field_values"
	MethodCustomFieldValueUpdate      toolsets.Method = "twprojects-update_custom_field_value"
	MethodCustomFieldValueUpdateBatch toolsets.Method = "twprojects-update_custom_field_values"
	MethodCustomFieldValueDelete      toolsets.Method = "twprojects-delete_custom_field_value"
	MethodCustomFieldValueGet         toolsets.Method = "twprojects-get_custom_field_value"
	MethodCustomFieldValueList        toolsets.Method = "twprojects-list_custom_field_values"
)

var (
	customFieldValueGetOutputSchema  *jsonschema.Schema
	customFieldValueListOutputSchema *jsonschema.Schema
)

func init() {
	var err error

	customFieldValueGetOutputSchema, err = jsonschema.For[projects.CustomFieldValueGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomFieldValueGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customFieldValueGetOutputSchema)
	customFieldValueListOutputSchema, err = jsonschema.For[projects.CustomFieldValueListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomFieldValueListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customFieldValueListOutputSchema)
}

// CustomFieldValueCreate creates a custom field value on a task, project or
// company in Teamwork.com.
func CustomFieldValueCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomFieldValueCreate),
			Description: "Set a custom field value on a task, project or company. " +
				"The custom field must already exist and be applicable to the target entity.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Custom Field Value",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of entity the custom field value is attached to.",
						Enum: []any{
							"task",
							"project",
							"company",
						},
					},
					"entity_id": {
						Type:        "integer",
						Description: "The ID of the task, project or company the custom field value is attached to.",
					},
					"custom_field_id": {
						Type:        "integer",
						Description: "The ID of the custom field the value belongs to.",
					},
					"value": {
						Description: "The value to assign, typed per the field: string (text), " +
							"number (number), boolean (checkbox), choice value string " +
							"(dropdown/status; array for multiselect), ISO-8601 string (date).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "number"},
							{Type: "boolean"},
							{
								Type: "array",
								Items: &jsonschema.Schema{
									AnyOf: []*jsonschema.Schema{
										{Type: "string"},
										{Type: "number"},
									},
								},
							},
							{Type: "null"},
						},
					},
					"currency_code": {
						Description: "The ISO currency code for currency-type custom field values.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"country_code": {
						Description: "The country code for currency-type custom field values.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"entity", "entity_id", "custom_field_id", "value"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			customFieldValueCreateRequest, toolResult := parseCustomFieldValueCreateArguments(arguments)
			if toolResult != nil {
				return toolResult, nil
			}
			customFieldValueCreateRequest.Value = coerceCustomFieldValue(ctx, engine,
				customFieldValueCreateRequest.CustomFieldID, customFieldValueCreateRequest.Value)

			response, err := projects.CustomFieldValueCreate(ctx, engine, customFieldValueCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create custom field value")
			}
			return helpers.NewToolResultText("Custom field value created successfully with ID %d",
				response.CustomFieldValue.ID), nil
		},
	}
}

// parseCustomFieldValueCreateArguments binds the arguments of a single value
// create, shared by create_custom_field_value and each item of
// create_custom_field_values. The value is left uncoerced.
func parseCustomFieldValueCreateArguments(
	arguments map[string]any,
) (projects.CustomFieldValueCreateRequest, *mcp.CallToolResult) {
	var entity projects.CustomFieldEntity
	var entityID, customFieldID int64
	var currencyCode, countryCode *string
	err := helpers.ParamGroup(arguments,
		helpers.RequiredParam(&entity, "entity",
			helpers.RestrictValues(
				projects.CustomFieldEntityTask,
				projects.CustomFieldEntityProject,
				projects.CustomFieldEntityCompany,
			),
		),
		helpers.RequiredNumericParam(&entityID, "entity_id"),
		helpers.RequiredNumericParam(&customFieldID, "custom_field_id"),
		helpers.OptionalPointerParam(&currencyCode, "currency_code"),
		helpers.OptionalPointerParam(&countryCode, "country_code"),
	)
	if err != nil {
		return projects.CustomFieldValueCreateRequest{},
			helpers.NewToolResultTextError("invalid parameters: %s", err.Error())
	}

	value, ok := arguments["value"]
	if !ok {
		return projects.CustomFieldValueCreateRequest{},
			helpers.NewToolResultTextError("invalid parameters: 'value' is required")
	}

	var req projects.CustomFieldValueCreateRequest
	switch entity {
	case projects.CustomFieldEntityTask:
		req = projects.NewTaskCustomFieldValueCreateRequest(entityID, customFieldID, value)
	case projects.CustomFieldEntityProject:
		req = projects.NewProjectCustomFieldValueCreateRequest(entityID, customFieldID, value)
	case projects.CustomFieldEntityCompany:
		req = projects.NewCompanyCustomFieldValueCreateRequest(entityID, customFieldID, value)
	}
	req.CurrencyCode = currencyCode
	req.CountryCode = countryCode
	return req, nil
}

// CustomFieldValueCreateBatch sets many custom field values in Teamwork.com in one
// call.
//
// It is a loop of single creates because create is what sets a value: it
// replaces any value the entity already has for that field, except on a
// multiselect field, where it adds to the selection. The bulk update route only
// edits values that already exist, addressed by value ID.
func CustomFieldValueCreateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	itemSchema := batchItemSchema(CustomFieldValueCreate(engine).Tool.InputSchema, "entity")
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomFieldValueCreateBatch),
			Description: "Set many custom field values on tasks, projects or companies in one call: one field " +
				"across many records, many fields on one record, or any mix. Use instead of calling " +
				"twprojects-create_custom_field_value in a loop. Setting a value replaces the one the record " +
				"already has for that field, except on a multiselect field, where the options are added to " +
				"the selection. Items are validated first and nothing is set if one is invalid; after that " +
				"each value succeeds or fails on its own, and the result lists the error of each failure. " +
				"Costs one request per value; to change values that already exist on one record, " +
				string(MethodCustomFieldValueUpdateBatch) + " does it in a single request.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Custom Field Values",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of entity every value in this call is attached to.",
						Enum:        []any{"task", "project", "company"},
					},
					"values": {
						Type:     "array",
						Items:    itemSchema,
						MinItems: new(1),
						MaxItems: new(batchMaxItems),
						Description: "The values to set. A record and field pair may appear only once; " +
							"list every option of a multiselect in one value.",
					},
				},
				Required: []string{"entity", "values"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}
			entity, ok := arguments["entity"]
			if !ok {
				return helpers.NewToolResultTextError("invalid parameters: 'entity' is required"), nil
			}
			items, toolResult := batchItems(arguments, "values", batchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			type target struct{ entityID, customFieldID int64 }
			requests := make([]projects.CustomFieldValueCreateRequest, len(items))
			labels := make([]string, len(items))
			seen := make(map[target]bool, len(items))
			var invalid []string
			for i, item := range items {
				item["entity"] = entity
				req, toolResult := parseCustomFieldValueCreateArguments(item)
				if toolResult != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, toolResultText(toolResult)))
					continue
				}
				key := target{customFieldID: req.CustomFieldID}
				if err := helpers.ParamGroup(item, helpers.RequiredNumericParam(&key.entityID, "entity_id")); err != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, err.Error()))
					continue
				}
				// Two concurrent writes of one field on one record would race.
				if seen[key] {
					invalid = append(invalid, fmt.Sprintf("item %d: field %d on %s %d is listed more than once",
						i+1, key.customFieldID, entity, key.entityID))
					continue
				}
				seen[key] = true
				requests[i] = req
				labels[i] = fmt.Sprintf("field %d on %s %d", key.customFieldID, entity, key.entityID)
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			// Coerce with one field lookup per field rather than one per value.
			fieldTypes := make(map[int64]*projects.CustomFieldType)
			for i := range requests {
				if !containsNumber(requests[i].Value) {
					continue
				}
				id := requests[i].CustomFieldID
				if _, ok := fieldTypes[id]; !ok {
					fieldTypes[id] = nil
					if resp, err := projects.CustomFieldGet(ctx, engine, projects.NewCustomFieldGetRequest(id)); err == nil {
						fieldTypes[id] = &resp.CustomField.Type
					}
				}
				if fieldType := fieldTypes[id]; fieldType != nil {
					requests[i].Value = coerceCustomFieldValueForType(*fieldType, requests[i].Value)
				}
			}

			errs := runBatch(ctx, len(requests), batchConcurrency, func(ctx context.Context, i int) error {
				_, err := projects.CustomFieldValueCreate(ctx, engine, requests[i])
				return err
			})
			return batchReport("Set", "custom field values",
				"check with "+string(MethodCustomFieldValueList)+" before sending multiselect values again",
				labels, nil, errs), nil
		},
	}
}

// CustomFieldValueUpdate updates a custom field value on a task, project or
// company in Teamwork.com.
func CustomFieldValueUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomFieldValueUpdate),
			Description: "Update a custom field value on a task, project or company. To update several " +
				"values of one record, use " + string(MethodCustomFieldValueUpdateBatch) + " instead.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Custom Field Value",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of entity the custom field value is attached to.",
						Enum: []any{
							"task",
							"project",
							"company",
						},
					},
					"entity_id": {
						Type:        "integer",
						Description: "The ID of the task, project or company the custom field value belongs to.",
					},
					"value_id": {
						Type:        "integer",
						Description: "The ID of the custom field value entry to update.",
					},
					"custom_field_id": {
						Type:        "integer",
						Description: "The ID of the custom field the value belongs to.",
					},
					"value": {
						Description: "The value to assign, typed per the field: string (text), " +
							"number (number), boolean (checkbox), choice value string " +
							"(dropdown/status; array for multiselect), ISO-8601 string (date).",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "number"},
							{Type: "boolean"},
							{
								Type: "array",
								Items: &jsonschema.Schema{
									AnyOf: []*jsonschema.Schema{
										{Type: "string"},
										{Type: "number"},
									},
								},
							},
							{Type: "null"},
						},
					},
					"currency_code": {
						Description: "The ISO currency code for currency-type custom field values.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"country_code": {
						Description: "The country code for currency-type custom field values.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
				},
				Required: []string{"entity", "entity_id", "value_id", "custom_field_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var entity projects.CustomFieldEntity
			var entityID, valueID, customFieldID int64
			var currencyCode, countryCode *string
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&entity, "entity",
					helpers.RestrictValues(
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityCompany,
					),
				),
				helpers.RequiredNumericParam(&entityID, "entity_id"),
				helpers.RequiredNumericParam(&valueID, "value_id"),
				helpers.RequiredNumericParam(&customFieldID, "custom_field_id"),
				helpers.OptionalPointerParam(&currencyCode, "currency_code"),
				helpers.OptionalPointerParam(&countryCode, "country_code"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			value := arguments["value"]
			value = coerceCustomFieldValue(ctx, engine, customFieldID, value)

			var customFieldValueUpdateRequest projects.CustomFieldValueUpdateRequest
			switch entity {
			case projects.CustomFieldEntityTask:
				customFieldValueUpdateRequest =
					projects.NewTaskCustomFieldValueUpdateRequest(entityID, customFieldID, valueID, value)
			case projects.CustomFieldEntityProject:
				customFieldValueUpdateRequest =
					projects.NewProjectCustomFieldValueUpdateRequest(entityID, customFieldID, valueID, value)
			case projects.CustomFieldEntityCompany:
				customFieldValueUpdateRequest =
					projects.NewCompanyCustomFieldValueUpdateRequest(entityID, customFieldID, valueID, value)
			}
			customFieldValueUpdateRequest.CurrencyCode = currencyCode
			customFieldValueUpdateRequest.CountryCode = countryCode

			_, err = projects.CustomFieldValueUpdate(ctx, engine, customFieldValueUpdateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update custom field value")
			}
			return helpers.NewToolResultText("Custom field value updated successfully"), nil
		},
	}
}

// customFieldValueUpdateBatchMaxItems bounds update_custom_field_values. The
// whole list is one request, validated value by value before anything is
// written, so the cap keeps that request and its error easy to act on.
const customFieldValueUpdateBatchMaxItems = 100

// CustomFieldValueUpdateBatch updates many existing custom field values of one
// task, project or company in Teamwork.com with a single request.
//
// Unlike create_custom_field_values, which is a loop of single creates, this is
// one call to the bulk update route, so it costs one request whatever the list
// length. The route works on one entity per request, addresses values by value
// ID, and is all-or-nothing.
func CustomFieldValueUpdateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	valueSchema := CustomFieldValueUpdate(engine).Tool.InputSchema.(*jsonschema.Schema).Properties["value"]
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomFieldValueUpdateBatch),
			Description: fmt.Sprintf("Update many existing custom field values of ONE task, project or company "+
				"in a single request. Works on one entity per call: every value must already exist on that "+
				"entity, addressed by its value ID from "+string(MethodCustomFieldValueList)+". Use instead of "+
				"calling "+string(MethodCustomFieldValueUpdate)+" in a loop. To set values that do not exist "+
				"yet, or values across many records, use "+string(MethodCustomFieldValueCreateBatch)+" instead. "+
				"The batch is all-or-nothing: if any value is rejected, no value is updated. A multiselect "+
				"value replaces the whole selection. At most %d values per call.",
				customFieldValueUpdateBatchMaxItems),
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Custom Field Values",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of the one entity whose values are updated.",
						Enum:        []any{"task", "project", "company"},
					},
					"entity_id": {
						Type:        "integer",
						Description: "The ID of the one task, project or company every value belongs to.",
					},
					"values": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"value_id": {
									Type:        "integer",
									Description: "The ID of the existing custom field value entry to update.",
								},
								"custom_field_id": {
									Type:        "integer",
									Description: "The ID of the custom field the value belongs to.",
								},
								"value": valueSchema,
								"clear": {
									Type:        "boolean",
									Description: "Set to true instead of sending value to clear the field.",
								},
							},
							Required: []string{"value_id", "custom_field_id"},
						},
						MinItems: new(1),
						MaxItems: new(customFieldValueUpdateBatchMaxItems),
						Description: fmt.Sprintf("The values to update, at most %d. Each value ID may appear "+
							"only once, and each item needs either value or clear.",
							customFieldValueUpdateBatchMaxItems),
					},
				},
				Required: []string{"entity", "entity_id", "values"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var entity projects.CustomFieldEntity
			var entityID int64
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&entity, "entity",
					helpers.RestrictValues(
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityCompany,
					),
				),
				helpers.RequiredNumericParam(&entityID, "entity_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "values", customFieldValueUpdateBatchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			values := make([]projects.CustomFieldValueBulkUpdateItem, len(items))
			seen := make(map[int64]int, len(items))
			var invalid []string
			for i, item := range items {
				var valueID, customFieldID int64
				var clear bool
				err := helpers.ParamGroup(item,
					helpers.RequiredNumericParam(&valueID, "value_id"),
					helpers.RequiredNumericParam(&customFieldID, "custom_field_id"),
					helpers.OptionalParam(&clear, "clear"),
				)
				if err != nil {
					invalid = append(invalid, fmt.Sprintf("item %d: %s", i+1, err.Error()))
					continue
				}
				value, hasValue := item["value"]
				if hasValue && value == nil {
					hasValue = false
				}
				switch {
				case clear && hasValue:
					invalid = append(invalid, fmt.Sprintf("item %d: send either value or clear, not both", i+1))
					continue
				case !clear && !hasValue:
					invalid = append(invalid, fmt.Sprintf("item %d: value is required unless clear is true", i+1))
					continue
				}
				// The endpoint answers a repeated value ID with a 404 that names
				// no value, so catch it here where the item can be named.
				if first, ok := seen[valueID]; ok {
					invalid = append(invalid, fmt.Sprintf("item %d: value %d is already listed as item %d",
						i+1, valueID, first))
					continue
				}
				seen[valueID] = i + 1
				values[i] = projects.CustomFieldValueBulkUpdateItem{
					ID:            valueID,
					CustomFieldID: customFieldID,
					Value:         value,
				}
			}
			if len(invalid) > 0 {
				return helpers.NewToolResultTextError(
					"No values were updated. Fix these items and send the whole list again:\n%s",
					strings.Join(invalid, "\n")), nil
			}

			// Coerce numeric choices with one field lookup per field rather than
			// one per value.
			fieldTypes := make(map[int64]*projects.CustomFieldType)
			for i := range values {
				if !containsNumber(values[i].Value) {
					continue
				}
				id := values[i].CustomFieldID
				if _, ok := fieldTypes[id]; !ok {
					fieldTypes[id] = nil
					if resp, err := projects.CustomFieldGet(ctx, engine, projects.NewCustomFieldGetRequest(id)); err == nil {
						fieldTypes[id] = &resp.CustomField.Type
					}
				}
				if fieldType := fieldTypes[id]; fieldType != nil {
					values[i].Value = coerceCustomFieldValueForType(*fieldType, values[i].Value)
				}
			}

			var req projects.CustomFieldValueBulkUpdateRequest
			switch entity {
			case projects.CustomFieldEntityTask:
				req = projects.NewTaskCustomFieldValueBulkUpdateRequest(entityID, values)
			case projects.CustomFieldEntityProject:
				req = projects.NewProjectCustomFieldValueBulkUpdateRequest(entityID, values)
			case projects.CustomFieldEntityCompany:
				req = projects.NewCompanyCustomFieldValueBulkUpdateRequest(entityID, values)
			}

			response, err := projects.CustomFieldValueBulkUpdate(ctx, engine, req)
			if err != nil {
				return customFieldValueUpdateBatchFailure(err, entity, entityID)
			}
			ids := make([]string, len(response.CustomFieldValues))
			for i, value := range response.CustomFieldValues {
				ids[i] = strconv.FormatInt(value.ID, 10)
			}
			return helpers.NewToolResultText("Updated %d of %d custom field values on %s %d (value IDs: %s).",
				len(response.CustomFieldValues), len(values), entity, entityID, strings.Join(ids, ", ")), nil
		},
	}
}

// customFieldValueUpdateBatchFailure reports a rejected bulk update. The route
// is all-or-nothing, so a 4xx means nothing was written; it does not name the
// value that caused the rejection, so the result says how to find it.
func customFieldValueUpdateBatchFailure(
	apiErr error,
	entity projects.CustomFieldEntity,
	entityID int64,
) (*mcp.CallToolResult, error) {
	result, err := helpers.HandleAPIError(apiErr, "failed to update custom field values")
	if result == nil {
		return result, err
	}
	detail := toolResultText(result)
	if httpErr, ok := errors.AsType[*twapi.HTTPError](apiErr); ok && httpErr.StatusCode >= 500 {
		return helpers.NewToolResultTextError("The update may not have been applied: %s. Check the values with %s "+
			"before sending the list again.", detail, MethodCustomFieldValueList), nil
	}
	return helpers.NewToolResultTextError("No values were updated: the batch is all-or-nothing and the API "+
		"rejected it (%s). The API does not say which value caused it. Check that every value ID exists on %s %d "+
		"with %s, that each custom_field_id is the value's own field, and that each value fits its field type, "+
		"then send the whole list again.", detail, entity, entityID, MethodCustomFieldValueList), nil
}

// CustomFieldValueDelete clears a custom field value from a task, project or
// company in Teamwork.com.
func CustomFieldValueDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldValueDelete),
			Description: "Clear a custom field value from a task, project or company.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Custom Field Value",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of entity the custom field value is attached to.",
						Enum: []any{
							"task",
							"project",
							"company",
						},
					},
					"entity_id": {
						Type:        "integer",
						Description: "The ID of the task, project or company the custom field value belongs to.",
					},
					"value_id": {
						Type:        "integer",
						Description: "The ID of the custom field value entry to delete.",
					},
				},
				Required: []string{"entity", "entity_id", "value_id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var entity projects.CustomFieldEntity
			var entityID, valueID int64
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&entity, "entity",
					helpers.RestrictValues(
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityCompany,
					),
				),
				helpers.RequiredNumericParam(&entityID, "entity_id"),
				helpers.RequiredNumericParam(&valueID, "value_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			var customFieldValueDeleteRequest projects.CustomFieldValueDeleteRequest
			switch entity {
			case projects.CustomFieldEntityTask:
				customFieldValueDeleteRequest = projects.NewTaskCustomFieldValueDeleteRequest(entityID, valueID)
			case projects.CustomFieldEntityProject:
				customFieldValueDeleteRequest = projects.NewProjectCustomFieldValueDeleteRequest(entityID, valueID)
			case projects.CustomFieldEntityCompany:
				customFieldValueDeleteRequest = projects.NewCompanyCustomFieldValueDeleteRequest(entityID, valueID)
			}

			_, err = projects.CustomFieldValueDelete(ctx, engine, customFieldValueDeleteRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete custom field value")
			}
			return helpers.NewToolResultText("Custom field value deleted successfully"), nil
		},
	}
}

// CustomFieldValueGet retrieves a single custom field value from a task,
// project or company in Teamwork.com.
func CustomFieldValueGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldValueGet),
			Description: "Get a single custom field value from a task, project or company.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Custom Field Value",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of entity the custom field value is attached to.",
						Enum: []any{
							"task",
							"project",
							"company",
						},
					},
					"entity_id": {
						Type:        "integer",
						Description: "The ID of the task, project or company the custom field value belongs to.",
					},
					"value_id": {
						Type:        "integer",
						Description: "The ID of the custom field value entry to retrieve.",
					},
				},
				Required: []string{"entity", "entity_id", "value_id"},
			},
			OutputSchema: customFieldValueGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var entity projects.CustomFieldEntity
			var entityID, valueID int64
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&entity, "entity",
					helpers.RestrictValues(
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityCompany,
					),
				),
				helpers.RequiredNumericParam(&entityID, "entity_id"),
				helpers.RequiredNumericParam(&valueID, "value_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			var customFieldValueGetRequest projects.CustomFieldValueGetRequest
			switch entity {
			case projects.CustomFieldEntityTask:
				customFieldValueGetRequest = projects.NewTaskCustomFieldValueGetRequest(entityID, valueID)
			case projects.CustomFieldEntityProject:
				customFieldValueGetRequest = projects.NewProjectCustomFieldValueGetRequest(entityID, valueID)
			case projects.CustomFieldEntityCompany:
				customFieldValueGetRequest = projects.NewCompanyCustomFieldValueGetRequest(entityID, valueID)
			}

			response, err := projects.CustomFieldValueGet(ctx, engine, customFieldValueGetRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get custom field value")
			}
			return helpers.NewToolResultJSON(response)
		},
	}
}

// CustomFieldValueList lists the custom field values of a task, project or
// company in Teamwork.com.
func CustomFieldValueList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldValueList),
			Description: "List the custom field values of a task, project or company.",
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Custom Field Values",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"entity": {
						Type:        "string",
						Description: "The type of entity to list custom field values for.",
						Enum: []any{
							"task",
							"project",
							"company",
						},
					},
					"entity_id": {
						Type:        "integer",
						Description: "The ID of the task, project or company to list custom field values for.",
					},
					"custom_field_ids": {
						Description: "Filter by custom field.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"page":       helpers.PageSchema(),
					"page_size":  helpers.PageSizeSchema(),
					"verbose":    helpers.VerboseSchema(),
					"count_only": helpers.CountOnlySchema("custom field values"),
					"fields":     helpers.FieldsSchema[projects.CustomFieldValue]("custom field value"),
				},
				Required: []string{"entity", "entity_id"},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(customFieldValueListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var entity projects.CustomFieldEntity
			var entityID int64
			var customFieldIDs []int64
			var page, pageSize int64
			var fields []projects.CustomFieldValueField
			verbose := true
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.RequiredParam(&entity, "entity",
					helpers.RestrictValues(
						projects.CustomFieldEntityTask,
						projects.CustomFieldEntityProject,
						projects.CustomFieldEntityCompany,
					),
				),
				helpers.RequiredNumericParam(&entityID, "entity_id"),
				helpers.OptionalNumericListParam(&customFieldIDs, "custom_field_ids"),
				helpers.OptionalNumericParam(&page, "page"),
				helpers.OptionalNumericParam(&pageSize, "page_size"),
				helpers.OptionalParam(&verbose, "verbose"),
				helpers.OptionalParam(&countOnly, "count_only"),
				helpers.OptionalFieldsParam[projects.CustomFieldValue](&fields, "fields"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			var customFieldValueListRequest projects.CustomFieldValueListRequest
			switch entity {
			case projects.CustomFieldEntityTask:
				customFieldValueListRequest = projects.NewTaskCustomFieldValueListRequest(entityID)
			case projects.CustomFieldEntityProject:
				customFieldValueListRequest = projects.NewProjectCustomFieldValueListRequest(entityID)
			case projects.CustomFieldEntityCompany:
				customFieldValueListRequest = projects.NewCompanyCustomFieldValueListRequest(entityID)
			}
			customFieldValueListRequest.Filters.CustomFieldIDs = customFieldIDs
			if page > 0 {
				customFieldValueListRequest.Filters.Page = page
			}
			if pageSize > 0 {
				customFieldValueListRequest.Filters.PageSize = pageSize
			}
			switch {
			case len(fields) > 0:
				customFieldValueListRequest.Filters.Fields.CustomFieldValues = fields
			case !verbose:
				customFieldValueListRequest.Filters.Fields.CustomFieldValues = []projects.CustomFieldValueField{
					projects.CustomFieldValueFieldID,
					projects.CustomFieldValueFieldValue,
					projects.CustomFieldValueFieldCustomField,
				}
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, customFieldValueListRequest, "failed to count custom field values")
			}

			resp, err := twapi.ExecuteRaw(ctx, engine, customFieldValueListRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list custom field values")
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			if resp.StatusCode != http.StatusOK {
				return helpers.HandleAPIError(
					twapi.NewHTTPError(resp, "failed to list custom field values"),
					"failed to list custom field values",
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
			var structured any
			if err := json.Unmarshal(body, &structured); err != nil {
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			result.StructuredContent = structured
			return result, nil
		},
	}
}

// coerceCustomFieldValue normalizes the raw value into the wire shape the
// Teamwork API expects. Dropdown and multiselect custom fields store their
// choices as strings, but MCP clients frequently send a choice as a JSON
// number (the tool schema also permits numbers, for number-type fields). A
// numeric value posted to a dropdown field is rejected by the API with
// "cannot unmarshal number ... into string", so when the raw value carries a
// number we resolve the field type and stringify it for dropdown/multiselect
// fields. Every other case is passed through untouched; if the field type
// cannot be resolved we forward the value unchanged and let the API validate.
func coerceCustomFieldValue(ctx context.Context, engine *twapi.Engine, customFieldID int64, raw any) any {
	if customFieldID == 0 || !containsNumber(raw) {
		return raw
	}
	resp, err := projects.CustomFieldGet(ctx, engine, projects.NewCustomFieldGetRequest(customFieldID))
	if err != nil {
		return raw
	}
	return coerceCustomFieldValueForType(resp.CustomField.Type, raw)
}

// coerceCustomFieldValueForType is coerceCustomFieldValue with the field type
// already known.
func coerceCustomFieldValueForType(fieldType projects.CustomFieldType, raw any) any {
	// Dropdown, status and multiselect fields store their choices as strings,
	// so a numeric choice must be stringified. Dropdown and status are
	// single-valued; multiselect is an array.
	switch fieldType {
	case projects.CustomFieldTypeDropdown, projects.CustomFieldTypeStatus:
		if s, ok := stringifyScalar(raw); ok {
			return s
		}
	case projects.CustomFieldTypeMultiselect:
		if arr, ok := raw.([]any); ok {
			out := make([]any, len(arr))
			for i, v := range arr {
				if s, ok := stringifyScalar(v); ok {
					out[i] = s
				} else {
					out[i] = v
				}
			}
			return out
		}
		if s, ok := stringifyScalar(raw); ok {
			return s
		}
	}
	return raw
}

// containsNumber reports whether raw is a JSON number, or an array holding at
// least one number. It is the cheap pre-check that lets coerceCustomFieldValue
// skip the field lookup for the common case of already-string values.
func containsNumber(raw any) bool {
	switch v := raw.(type) {
	case float64, float32, int, int32, int64, json.Number:
		return true
	case []any:
		if slices.ContainsFunc(v, containsNumber) {
			return true
		}
	}
	return false
}

// stringifyScalar renders a scalar JSON value as the string the API expects for
// dropdown choices. Integer-valued numbers are rendered without a trailing
// decimal (1 -> "1", not "1.000000"). It reports false for values that have no
// meaningful scalar string form (e.g. arrays or objects).
func stringifyScalar(raw any) (string, bool) {
	switch v := raw.(type) {
	case string:
		return v, true
	case float64:
		if !math.IsInf(v, 0) && !math.IsNaN(v) && v == math.Trunc(v) {
			return strconv.FormatInt(int64(v), 10), true
		}
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return stringifyScalar(float64(v))
	case int:
		return strconv.FormatInt(int64(v), 10), true
	case int32:
		return strconv.FormatInt(int64(v), 10), true
	case int64:
		return strconv.FormatInt(v, 10), true
	case bool:
		return strconv.FormatBool(v), true
	case json.Number:
		return v.String(), true
	}
	return "", false
}
