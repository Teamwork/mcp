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
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// List of methods available in the Teamwork.com MCP service.
//
// The naming convention for methods follows a pattern described here:
// https://github.com/github/github-mcp-server/issues/333
const (
	MethodCustomFieldValueCreate toolsets.Method = "twprojects-create_custom_field_value"
	MethodCustomFieldValueUpdate toolsets.Method = "twprojects-update_custom_field_value"
	MethodCustomFieldValueDelete toolsets.Method = "twprojects-delete_custom_field_value"
	MethodCustomFieldValueGet    toolsets.Method = "twprojects-get_custom_field_value"
	MethodCustomFieldValueList   toolsets.Method = "twprojects-list_custom_field_values"
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
				Title: "Create Custom Field Value",
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
						Description: "The value to assign to the custom field. " +
							"The concrete type depends on the custom field definition: " +
							"strings for text fields, numbers for number fields, booleans for checkboxes, " +
							"option IDs for dropdown fields (array of integers for multiselect), " +
							"ISO-8601 date strings for date fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "number"},
							{Type: "boolean"},
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "string",
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
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			value, ok := arguments["value"]
			if !ok {
				return helpers.NewToolResultTextError("invalid parameters: 'value' is required"), nil
			}

			var customFieldValueCreateRequest projects.CustomFieldValueCreateRequest
			switch entity {
			case projects.CustomFieldEntityTask:
				customFieldValueCreateRequest = projects.NewTaskCustomFieldValueCreateRequest(entityID, customFieldID, value)
			case projects.CustomFieldEntityProject:
				customFieldValueCreateRequest = projects.NewProjectCustomFieldValueCreateRequest(entityID, customFieldID, value)
			case projects.CustomFieldEntityCompany:
				customFieldValueCreateRequest = projects.NewCompanyCustomFieldValueCreateRequest(entityID, customFieldID, value)
			}
			customFieldValueCreateRequest.CurrencyCode = currencyCode
			customFieldValueCreateRequest.CountryCode = countryCode

			response, err := projects.CustomFieldValueCreate(ctx, engine, customFieldValueCreateRequest)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create custom field value")
			}
			return helpers.NewToolResultText("Custom field value created successfully with ID %d",
				response.CustomFieldValue.ID), nil
		},
	}
}

// CustomFieldValueUpdate updates a custom field value on a task, project or
// company in Teamwork.com.
func CustomFieldValueUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldValueUpdate),
			Description: "Update a custom field value on a task, project or company.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Update Custom Field Value",
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
						Description: "The value to assign to the custom field. " +
							"The concrete type depends on the custom field definition: " +
							"strings for text fields, numbers for number fields, booleans for checkboxes, " +
							"option IDs for dropdown fields (array of integers for multiselect), " +
							"ISO-8601 date strings for date fields.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "number"},
							{Type: "boolean"},
							{
								Type: "array",
								Items: &jsonschema.Schema{
									Type: "string",
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

// CustomFieldValueDelete clears a custom field value from a task, project or
// company in Teamwork.com.
func CustomFieldValueDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomFieldValueDelete),
			Description: "Clear a custom field value from a task, project or company.",
			Annotations: &mcp.ToolAnnotations{
				Title: "Delete Custom Field Value",
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
				Title:        "Get Custom Field Value",
				ReadOnlyHint: true,
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
				Title:        "List Custom Field Values",
				ReadOnlyHint: true,
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
						Description: "A list of custom field IDs to filter values by.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"page":      helpers.PageSchema(),
					"page_size": helpers.PageSizeSchema(),
					"verbose":   helpers.VerboseSchema(),
				},
				Required: []string{"entity", "entity_id"},
			},
			OutputSchema: customFieldValueListOutputSchema,
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
			verbose := true
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
			if !verbose {
				customFieldValueListRequest.Filters.Fields.CustomFieldValues = []projects.CustomFieldValueField{
					projects.CustomFieldValueFieldID,
					projects.CustomFieldValueFieldValue,
					projects.CustomFieldValueFieldCustomField,
				}
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
