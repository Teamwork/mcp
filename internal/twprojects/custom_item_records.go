package twprojects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/toolsets"
	twapi "github.com/teamwork/twapi-go-sdk"
	"github.com/teamwork/twapi-go-sdk/projects"
)

const (
	MethodCustomItemRecordCreate      toolsets.Method = "twprojects-create_custom_item_record"
	MethodCustomItemRecordCreateBatch toolsets.Method = "twprojects-create_custom_item_records"
	MethodCustomItemRecordUpdate      toolsets.Method = "twprojects-update_custom_item_record"
	MethodCustomItemRecordDelete      toolsets.Method = "twprojects-delete_custom_item_record"
	MethodCustomItemRecordBulkDelete  toolsets.Method = "twprojects-bulk_delete_custom_item_records"
	MethodCustomItemRecordGet         toolsets.Method = "twprojects-get_custom_item_record"
	MethodCustomItemRecordList        toolsets.Method = "twprojects-list_custom_item_records"
)

// customItemRecordBatchMaxItems bounds create_custom_item_records. The whole
// list goes in one request, and the endpoint creates its records one after
// another, so the cap keeps a single call's work bounded.
const customItemRecordBatchMaxItems = 100

var (
	customItemRecordGetOutputSchema  *jsonschema.Schema
	customItemRecordListOutputSchema *jsonschema.Schema
)

// customItemRecordOrdering is the order-by vocabulary of the custom item records list endpoint.
var customItemRecordOrdering = newOrdering("custom item records",
	projects.CustomItemRecordOrderByDisplayOrder,
	projects.CustomItemRecordOrderByName,
	projects.CustomItemRecordOrderByCustomItemField,
	projects.CustomItemRecordOrderByID,
)

func init() {
	var err error
	customItemRecordGetOutputSchema, err = jsonschema.For[projects.CustomItemRecordGetResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomItemRecordGetResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customItemRecordGetOutputSchema)
	customItemRecordListOutputSchema, err = jsonschema.For[projects.CustomItemRecordListResponse](&jsonschema.ForOptions{})
	if err != nil {
		panic(fmt.Sprintf("failed to generate JSON schema for CustomItemRecordListResponse: %v", err))
	}
	helpers.WithMetaWebLinkSchema(customItemRecordListOutputSchema)
}

// ---------------------------------------------------------------------------
// Field-value schema used by record create/update tools
// ---------------------------------------------------------------------------

func fieldValuesInputSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: "Field values to set on the record. Each entry is {field_name, value}. " +
			"Field names are matched case-insensitively against the custom item type's fields. " +
			"Values are coerced by field type: " +
			"dropdown/multiselect accept option labels or option twIds; " +
			"date/time/datetime accept ISO-8601 strings; " +
			"checkbox accepts bool or yes/no/true/false; " +
			"number accepts numeric or numeric string; " +
			"user accepts a user ID (or array of IDs for multi-user fields). " +
			"To clear a field, send null.",
		AnyOf: []*jsonschema.Schema{
			{
				Type: "array",
				Items: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"field_name": {
							Type:        "string",
							Description: "Display name of the field, case-insensitive.",
						},
						"value": {
							Description: "Value to set. Type depends on the field — see the tool description.",
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
								{Type: "object"},
								{Type: "null"},
							},
						},
					},
					Required: []string{"field_name", "value"},
				},
			},
			{Type: "null"},
		},
	}
}

// ---------------------------------------------------------------------------
// Tools
// ---------------------------------------------------------------------------

// CustomItemRecordCreate creates a record on a custom item type.
func CustomItemRecordCreate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemRecordCreate),
			Description: "Create a record (row) on a custom item type. " +
				"For example, add a Contract on the Contracts type. " +
				"Pass field values by name; the tool resolves names to the API's internal IDs." +
				customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Custom Item Record",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID that will own the new record.",
					},
					"name": {
						Type:        "string",
						Description: "Display name of the record (e.g. \"Acme Inc Contract\").",
					},
					"section_id": {
						Description: "Optional section ID to place the record in.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"position_after_id": {
						Description: "Place the record after the given record ID. Null appends to the end.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"field_values": fieldValuesInputSchema(),
				},
				Required: []string{"custom_item_id", "name"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemID int64
			var name string
			var sectionID, positionAfterID *int64
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemID, "custom_item_id"),
				helpers.RequiredParam(&name, "name"),
				helpers.OptionalNumericPointerParam(&sectionID, "section_id"),
				helpers.OptionalNumericPointerParam(&positionAfterID, "position_after_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			fieldValues, errResult := buildRecordFieldValues(ctx, engine, customItemID, arguments, true)
			if errResult != nil {
				return errResult, nil
			}

			req := projects.NewCustomItemRecordCreateRequest(customItemID, name)
			req.PositionAfterID = positionAfterID
			if sectionID != nil {
				req.SectionID = twapi.NewNullableInt64(*sectionID)
			}
			req.FieldValues = fieldValues

			resp, err := projects.CustomItemRecordCreate(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to create custom item record")
			}
			return helpers.NewToolResultText(
				"Custom item record created successfully with ID %d", resp.CustomItemRecord.ID,
			), nil
		},
	}
}

// CustomItemRecordUpdate updates a record on a custom item type.
func CustomItemRecordUpdate(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemRecordUpdate),
			Description: "Update a record on a custom item type. " +
				"Only the fields you supply are changed; others are left as-is. " +
				"Set section_id to null to remove the record from any section." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Update Custom Item Record",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the record belongs to.",
					},
					"id": {
						Type:        "integer",
						Description: "Record ID to update.",
					},
					"name": {
						Description: "New display name for the record.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string"},
							{Type: "null"},
						},
					},
					"section_id": {
						Description: "New section ID, or null to remove the record from any section.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"clear_section": {
						Description: "Set to true to explicitly clear the record's section. " +
							"Use this instead of section_id when you want null semantics.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"position_after_id": {
						Description: "Move the record after the given record ID.",
						AnyOf: []*jsonschema.Schema{
							{Type: "integer"},
							{Type: "null"},
						},
					},
					"field_values": fieldValuesInputSchema(),
				},
				Required: []string{"custom_item_id", "id"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemID, recordID int64
			var name *string
			var sectionID, positionAfterID *int64
			var clearSection *bool
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemID, "custom_item_id"),
				helpers.RequiredNumericParam(&recordID, "id"),
				helpers.OptionalPointerParam(&name, "name"),
				helpers.OptionalNumericPointerParam(&sectionID, "section_id"),
				helpers.OptionalPointerParam(&clearSection, "clear_section"),
				helpers.OptionalNumericPointerParam(&positionAfterID, "position_after_id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			fieldValues, errResult := buildRecordFieldValues(ctx, engine, customItemID, arguments, false)
			if errResult != nil {
				return errResult, nil
			}

			req := projects.NewCustomItemRecordUpdateRequest(customItemID, recordID)
			req.Name = name
			req.PositionAfterID = positionAfterID
			switch {
			case clearSection != nil && *clearSection:
				req.SectionID = twapi.NullInt64()
			case sectionID != nil:
				req.SectionID = twapi.NewNullableInt64(*sectionID)
			}
			req.FieldValues = fieldValues

			_, err = projects.CustomItemRecordUpdate(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to update custom item record")
			}
			return helpers.NewToolResultText("Custom item record updated successfully"), nil
		},
	}
}

// CustomItemRecordDelete deletes a single record on a custom item type.
func CustomItemRecordDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name:        string(MethodCustomItemRecordDelete),
			Description: "Delete a single record on a custom item type." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Delete Custom Item Record",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the record belongs to.",
					},
					"id": {
						Type:        "integer",
						Description: "Record ID to delete.",
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

			var req projects.CustomItemRecordDeleteRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.RequiredNumericParam(&req.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			_, err = projects.CustomItemRecordDelete(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to delete custom item record")
			}
			return helpers.NewToolResultText("Custom item record deleted successfully"), nil
		},
	}
}

// CustomItemRecordBulkDelete deletes multiple records in a single API call.
// Prefer this over calling delete repeatedly.
func CustomItemRecordBulkDelete(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemRecordBulkDelete),
			Description: "Delete many records on a custom item type in one call. " +
				"Use this instead of calling delete_custom_item_record in a loop." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Bulk Delete Custom Item Records",
				DestructiveHint: new(true),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the records belong to.",
					},
					"ids": {
						Type:        "array",
						Items:       &jsonschema.Schema{Type: "integer"},
						Description: "Record IDs to delete.",
						MinItems:    new(1),
					},
				},
				Required: []string{"custom_item_id", "ids"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemID int64
			var ids []int64
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemID, "custom_item_id"),
				helpers.OptionalNumericListParam(&ids, "ids"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			if len(ids) == 0 {
				return helpers.NewToolResultTextError("invalid parameters: ids must not be empty"), nil
			}

			req := projects.NewCustomItemRecordBulkDeleteRequest(customItemID, ids)
			_, err = projects.CustomItemRecordBulkDelete(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to bulk delete custom item records")
			}
			return helpers.NewToolResultText("Deleted %d custom item record(s)", len(ids)), nil
		},
	}
}

// CustomItemRecordCreateBatch creates many records on one custom item type
// with a single request to the bulk create endpoint, which runs with partial
// success on: each record is created or rejected on its own.
func CustomItemRecordCreateBatch(engine *twapi.Engine) toolsets.ToolWrapper {
	itemSchema := batchItemSchema(CustomItemRecordCreate(engine).Tool.InputSchema, "custom_item_id")
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemRecordCreateBatch),
			Description: fmt.Sprintf("Create many records (rows) on one custom item type in a single request, "+
				"for example importing a list of Contracts. Use instead of calling "+
				"twprojects-create_custom_item_record in a loop. Accepts at most %d records per call; split a "+
				"longer list across several calls. Each record takes the same parameters as the single-record "+
				"tool, with field values passed by name. Records are checked first and nothing is created if "+
				"one is invalid (an unknown field or option, or a missing required field). After that each "+
				"record is created or rejected on its own: the result gives the ID of every new record and the "+
				"reason for every failure, numbering records from 1 in the order listed. Resend only the "+
				"failed ones.", customItemRecordBatchMaxItems) + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Create Custom Item Records",
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID that will own every new record.",
					},
					"records": {
						Type:     "array",
						Items:    itemSchema,
						MinItems: new(1),
						MaxItems: new(customItemRecordBatchMaxItems),
						Description: fmt.Sprintf("The records to create, at most %d. "+
							"Failures are reported by position in this list, counting from 1.",
							customItemRecordBatchMaxItems),
					},
				},
				Required: []string{"custom_item_id", "records"},
			},
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var customItemID int64
			if err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&customItemID, "custom_item_id"),
			); err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}
			items, toolResult := batchItems(arguments, "records", customItemRecordBatchMaxItems)
			if toolResult != nil {
				return toolResult, nil
			}

			records := make([]projects.CustomItemRecordBulkCreateRecord, len(items))
			labels := make([]string, len(items))
			var invalid []string
			for i, item := range items {
				labels[i] = fmt.Sprintf("item %d", i+1)
				var record projects.CustomItemRecordBulkCreateRecord
				err := helpers.ParamGroup(item,
					helpers.RequiredParam(&record.Name, "name"),
					helpers.OptionalNumericPointerParam(&record.SectionID, "section_id"),
					helpers.OptionalNumericPointerParam(&record.PositionAfterID, "position_after_id"),
				)
				if err != nil {
					invalid = append(invalid, fmt.Sprintf("%s: %s", labels[i], err.Error()))
					continue
				}
				labels[i] = fmt.Sprintf("item %d (%q)", i+1, record.Name)
				// The field schema is cached, so this is one lookup for the whole list.
				fieldValues, errResult := buildRecordFieldValues(ctx, engine, customItemID, item, true)
				if errResult != nil {
					invalid = append(invalid, fmt.Sprintf("%s: %s", labels[i], toolResultText(errResult)))
					continue
				}
				record.FieldValues = fieldValues
				records[i] = record
			}
			if len(invalid) > 0 {
				return batchInvalidItems(invalid), nil
			}

			req := projects.NewCustomItemRecordBulkCreateRequest(customItemID, records)
			req.Options.PartialSuccess = new(true)

			resp, err := projects.CustomItemRecordBulkCreate(ctx, engine, req)
			if err != nil {
				if failures, ok := customItemRecordBatchRejection(err); ok {
					return customItemRecordBatchReport(labels, nil, failures), nil
				}
				result, err := helpers.HandleAPIError(err, "failed to create custom item records")
				if result != nil && isServerError(err) {
					// With partial success each record is committed as it is
					// created, so a server error can follow records that exist.
					result.Content = append(result.Content, &mcp.TextContent{Text: "Some records may have " +
						"been created before the error; call twprojects-list_custom_item_records to check " +
						"before sending the list again."})
				}
				return result, err
			}
			return customItemRecordBatchReport(labels, resp.CustomItemRecords, resp.Failed), nil
		},
	}
}

// customItemRecordBatchRejection reads the per-record errors out of a bulk
// create the endpoint rejected as a whole, which it does when no record could
// be created: 403 when every record was forbidden, 400 otherwise, each error
// carrying its record's index under meta.index. ok is false for any other
// failure, such as an error that names no record.
func customItemRecordBatchRejection(err error) ([]projects.CustomItemRecordBulkCreateFailure, bool) {
	httpErr, ok := errors.AsType[*twapi.HTTPError](err)
	if !ok || (httpErr.StatusCode != http.StatusBadRequest && httpErr.StatusCode != http.StatusForbidden) {
		return nil, false
	}
	var body struct {
		Errors []projects.CustomItemRecordBulkCreateError `json:"errors"`
	}
	if json.Unmarshal([]byte(httpErr.Details), &body) != nil || len(body.Errors) == 0 {
		return nil, false
	}
	var failures []projects.CustomItemRecordBulkCreateFailure
	byIndex := make(map[int]int)
	for _, apiErr := range body.Errors {
		index, ok := apiErr.Meta["index"].(float64)
		if !ok {
			return nil, false
		}
		position, seen := byIndex[int(index)]
		if !seen {
			position = len(failures)
			byIndex[int(index)] = position
			failures = append(failures, projects.CustomItemRecordBulkCreateFailure{Index: int(index)})
		}
		failures[position].Errors = append(failures[position].Errors, apiErr)
	}
	return failures, true
}

// isServerError reports whether err is an API answer with a 5xx status.
func isServerError(err error) bool {
	httpErr, ok := errors.AsType[*twapi.HTTPError](err)
	return ok && httpErr.StatusCode >= http.StatusInternalServerError
}

// customItemRecordBatchReport summarises a bulk create: the ID of each record
// created, by its position in the request, and every failure with its reasons.
// The endpoint returns the created records in request order without their
// index, so each is matched to the next position that did not fail. It is an
// error result when any record failed, even though the others were created.
func customItemRecordBatchReport(
	labels []string,
	created []projects.CustomItemRecord,
	failures []projects.CustomItemRecordBulkCreateFailure,
) *mcp.CallToolResult {
	failed := make(map[int]bool, len(failures))
	for _, failure := range failures {
		failed[failure.Index] = true
	}

	var report strings.Builder
	fmt.Fprintf(&report, "Created %d of %d custom item records.", len(created), len(labels))
	if len(created) == 0 {
		report.WriteString(" Nothing was written.")
	}
	if len(created) > 0 {
		var entries []string
		next := 0
		for i := range labels {
			if failed[i] || next >= len(created) {
				continue
			}
			entries = append(entries, fmt.Sprintf("%s → ID %d", labels[i], created[next].ID))
			next++
		}
		// A count that does not line up cannot be matched to positions; report
		// the IDs rather than guess which record each belongs to.
		if len(created)+len(failures) != len(labels) {
			entries = entries[:0]
			for _, record := range created {
				entries = append(entries, fmt.Sprintf("ID %d (%q)", record.ID, record.Name))
			}
		}
		fmt.Fprintf(&report, "\nCreated: %s.", strings.Join(entries, ", "))
	}
	for _, failure := range failures {
		label := fmt.Sprintf("item %d", failure.Index+1)
		if failure.Index >= 0 && failure.Index < len(labels) {
			label = labels[failure.Index]
		}
		reasons := make([]string, 0, len(failure.Errors))
		for _, apiErr := range failure.Errors {
			reasons = append(reasons, customItemRecordBatchReason(apiErr))
		}
		fmt.Fprintf(&report, "\nFailed: %s: %s.", label, strings.Join(reasons, "; "))
	}
	if len(failures) > 0 {
		return helpers.NewToolResultTextError("%s", report.String())
	}
	return helpers.NewToolResultText("%s", report.String())
}

// customItemRecordBatchReason renders one API error for a failed record, with
// any detail the endpoint attached other than the record's own index.
func customItemRecordBatchReason(apiErr projects.CustomItemRecordBulkCreateError) string {
	reason := apiErr.Detail
	if apiErr.Title != "" {
		reason = apiErr.Title + ": " + reason
	}
	meta := make(map[string]any, len(apiErr.Meta))
	for key, value := range apiErr.Meta {
		if key != "index" {
			meta[key] = value
		}
	}
	if len(meta) > 0 {
		if encoded, err := json.Marshal(meta); err == nil {
			reason += " " + string(encoded)
		}
	}
	return reason
}

// CustomItemRecordGet retrieves a record on a custom item type. Field values
// in the response are keyed by field display name (not twId) and translated
// per field type — dropdown option twIds become option labels.
func CustomItemRecordGet(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemRecordGet),
			Description: "Get a single record. Field values come back keyed by display name " +
				"with dropdown values translated to their human-readable labels." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "Get Custom Item Record",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID the record belongs to.",
					},
					"id": {
						Type:        "integer",
						Description: "Record ID.",
					},
				},
				Required: []string{"custom_item_id", "id"},
			},
			OutputSchema: customItemRecordGetOutputSchema,
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			var req projects.CustomItemRecordGetRequest
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.RequiredNumericParam(&req.Path.ID, "id"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			resp, err := projects.CustomItemRecordGet(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to get custom item record")
			}

			fields, _ := resolveCustomItemFields(ctx, engine, req.Path.CustomItemID)
			if len(fields) > 0 {
				resp.CustomItemRecord.FieldValues = translateRecordFieldValuesOut(
					resp.CustomItemRecord.FieldValues, fields)
			}
			return helpers.NewToolResultJSON(resp)
		},
	}
}

// CustomItemRecordList lists records of a custom item type. Each record's
// field values are translated from twId-keyed to name-keyed and from option
// twIds back to option labels.
func CustomItemRecordList(engine *twapi.Engine) toolsets.ToolWrapper {
	return toolsets.ToolWrapper{
		Tool: &mcp.Tool{
			Name: string(MethodCustomItemRecordList),
			Description: "List records on a custom item type. " +
				"Returns each record with field values keyed by display name. " +
				"Use the section_ids filter to scope to a specific section." + customItemRoutingHint,
			Annotations: &mcp.ToolAnnotations{
				Title:           "List Custom Item Records",
				ReadOnlyHint:    true,
				DestructiveHint: new(false),
				OpenWorldHint:   new(false),
			},
			InputSchema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"custom_item_id": {
						Type:        "integer",
						Description: "Custom item type ID to list records for.",
					},
					"search_term": helpers.SearchTermSchema("custom item records", "name"),
					"ids": {
						Description: "Restrict to these record IDs.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"section_ids": {
						Description: "Restrict to records in these sections.",
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
							{Type: "null"},
						},
					},
					"show_deleted": {
						Description: "Include deleted records.",
						AnyOf: []*jsonschema.Schema{
							{Type: "boolean"},
							{Type: "null"},
						},
					},
					"order_by":          customItemRecordOrdering.orderBySchema(),
					"order_mode":        orderModeSchema(),
					"order_by_field_id": orderByFieldIDSchema("records", "customitemfield"),
					"page":              helpers.PageSchema(),
					"page_size":         helpers.PageSizeSchema(),
					"count_only":        helpers.CountOnlySchema("custom item records"),
				},
				Required: []string{"custom_item_id"},
			},
			OutputSchema: helpers.WithCountOnlySchema(helpers.WithOptionalFields(customItemRecordListOutputSchema)),
		},
		Handler: func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			var arguments map[string]any
			if err := json.Unmarshal(request.Params.Arguments, &arguments); err != nil {
				return helpers.NewToolResultTextError("failed to decode request: %s", err.Error()), nil
			}

			req := projects.NewCustomItemRecordListRequest(0)
			var countOnly bool
			err := helpers.ParamGroup(arguments,
				helpers.RequiredNumericParam(&req.Path.CustomItemID, "custom_item_id"),
				helpers.OptionalParam(&req.Filters.SearchTerm, "search_term"),
				helpers.OptionalNumericListParam(&req.Filters.IDs, "ids"),
				helpers.OptionalNumericListParam(&req.Filters.SectionIDs, "section_ids"),
				helpers.OptionalPointerParam(&req.Filters.ShowDeleted, "show_deleted"),
				customItemRecordOrdering.param(&req.Filters.OrderBy, &req.Filters.OrderMode),
				helpers.OptionalNumericParam(&req.Filters.OrderByFieldID, "order_by_field_id"),
				helpers.OptionalNumericParam(&req.Filters.Page, "page"),
				helpers.OptionalNumericParam(&req.Filters.PageSize, "page_size"),
				helpers.OptionalParam(&countOnly, "count_only"),
			)
			if err != nil {
				return helpers.NewToolResultTextError("invalid parameters: %s", err.Error()), nil
			}

			// Records can be many — clamp the default lower than other lists
			// (plan §6 sets page_size=25 default, hard cap 100).
			if req.Filters.PageSize <= 0 {
				req.Filters.PageSize = 25
			} else if req.Filters.PageSize > 100 {
				req.Filters.PageSize = 100
			}

			if countOnly {
				return helpers.NewCountToolResult(ctx, engine, req, "failed to count custom item records")
			}

			resp, err := projects.CustomItemRecordList(ctx, engine, req)
			if err != nil {
				return helpers.HandleAPIError(err, "failed to list custom item records")
			}

			fields, _ := resolveCustomItemFields(ctx, engine, req.Path.CustomItemID)
			if len(fields) > 0 {
				for i := range resp.CustomItemRecords {
					resp.CustomItemRecords[i].FieldValues = translateRecordFieldValuesOut(
						resp.CustomItemRecords[i].FieldValues, fields)
				}
			}
			return helpers.NewToolResultJSON(resp)
		},
	}
}

// ---------------------------------------------------------------------------
// Field-value translation
// ---------------------------------------------------------------------------

// buildRecordFieldValues parses the field_values argument, resolves field
// names to twIds, coerces each value per the field's type, and (on create)
// validates that all required fields are supplied. Returns a result error
// directly when the caller should short-circuit.
func buildRecordFieldValues(
	ctx context.Context,
	engine *twapi.Engine,
	customItemID int64,
	arguments map[string]any,
	requireAllRequired bool,
) (projects.CustomItemRecordFieldValues, *mcp.CallToolResult) {
	raw, hasKey := arguments["field_values"]
	supplied := map[string]bool{}

	fields, err := resolveCustomItemFields(ctx, engine, customItemID)
	if err != nil {
		return nil, helpers.NewToolResultTextError("failed to load custom item schema: %s", err.Error())
	}

	out := projects.CustomItemRecordFieldValues{}
	if hasKey && raw != nil {
		entries, ok := raw.([]any)
		if !ok {
			return nil, helpers.NewToolResultTextError(
				"invalid parameters: field_values must be an array of {field_name, value}")
		}
		for i, entry := range entries {
			obj, ok := entry.(map[string]any)
			if !ok {
				return nil, helpers.NewToolResultTextError(
					"invalid parameters: field_values[%d] must be an object", i)
			}
			rawName, ok := obj["field_name"].(string)
			if !ok || rawName == "" {
				return nil, helpers.NewToolResultTextError(
					"invalid parameters: field_values[%d].field_name is required", i)
			}
			value, hasValue := obj["value"]
			if !hasValue {
				return nil, helpers.NewToolResultTextError(
					"invalid parameters: field_values[%d].value is required (use null to clear)", i)
			}

			field, ferr := lookupFieldByName(fields, rawName)
			if ferr != nil {
				return nil, helpers.NewToolResultTextError("invalid parameters: %s", ferr.Error())
			}

			coerced, cerr := coerceFieldValue(field, value)
			if cerr != nil {
				return nil, helpers.NewToolResultTextError("invalid parameters: %s", cerr.Error())
			}
			out[field.TwID] = coerced
			supplied[field.TwID] = true
		}
	}

	if requireAllRequired {
		var missing []string
		for _, field := range fields {
			if !isFieldRequired(field) {
				continue
			}
			if supplied[field.TwID] {
				continue
			}
			missing = append(missing, field.DisplayName)
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return nil, helpers.NewToolResultTextError(
				"invalid parameters: required field(s) missing: %s — "+
					"call list_custom_item_fields for the full schema",
				strings.Join(missing, ", "),
			)
		}
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// lookupFieldByName finds the field whose display name matches case-
// insensitively. Returns a clear error on no-match or ambiguous match.
func lookupFieldByName(fields []projects.CustomItemField, name string) (projects.CustomItemField, error) {
	needle := strings.ToLower(strings.TrimSpace(name))
	var matches []projects.CustomItemField
	for _, field := range fields {
		if strings.ToLower(field.DisplayName) == needle {
			matches = append(matches, field)
		}
	}
	switch len(matches) {
	case 0:
		var names []string
		for _, field := range fields {
			names = append(names, field.DisplayName)
		}
		sort.Strings(names)
		return projects.CustomItemField{}, fmt.Errorf(
			"unknown field %q — known fields: %s", name, strings.Join(names, ", "))
	case 1:
		return matches[0], nil
	default:
		var ids []string
		for _, field := range matches {
			ids = append(ids, fmt.Sprintf("twId=%q id=%d", field.TwID, field.ID))
		}
		return projects.CustomItemField{}, fmt.Errorf(
			"ambiguous field %q matches %d fields (%s) — refer to it by twId or rename one",
			name, len(matches), strings.Join(ids, ", "))
	}
}

// isFieldRequired reports whether a field must carry a value on create.
// The SDK does not yet model an explicit Required flag on CustomItemField;
// we look in the Definition map (the API stores per-type config there) and
// fall back to a conservative "not required". When the API exposes a
// dedicated Required field, swap this implementation.
func isFieldRequired(field projects.CustomItemField) bool {
	if field.Definition == nil {
		return false
	}
	if v, ok := field.Definition["required"].(bool); ok {
		return v
	}
	return false
}

// coerceFieldValue converts a raw MCP-decoded value into the wire shape the
// API expects for the field's type. Returns (nil, nil) for null inputs so
// the caller can store an explicit clear.
func coerceFieldValue(field projects.CustomItemField, raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}

	switch field.Type {
	case projects.CustomItemFieldTypeTextShort,
		projects.CustomItemFieldTypeTextLong,
		projects.CustomItemFieldTypeURL:
		s, ok := raw.(string)
		if !ok {
			return nil, typeMismatch(field, "string", raw)
		}
		return s, nil

	case projects.CustomItemFieldTypeNumberDecimal,
		projects.CustomItemFieldTypeNumberInteger:
		num, ok := coerceNumber(raw)
		if !ok {
			return nil, typeMismatch(field, "number or numeric string", raw)
		}
		if field.Type == projects.CustomItemFieldTypeNumberInteger {
			return int64(num), nil
		}
		return num, nil

	case projects.CustomItemFieldTypeCheckbox:
		b, ok := coerceBool(raw)
		if !ok {
			return nil, typeMismatch(field, "boolean or yes/no/true/false", raw)
		}
		return b, nil

	case projects.CustomItemFieldTypeDate,
		projects.CustomItemFieldTypeDateTime,
		projects.CustomItemFieldTypeTime:
		s, ok := raw.(string)
		if !ok {
			return nil, typeMismatch(field, "ISO-8601 string", raw)
		}
		if _, err := parseFieldDate(field.Type, s); err != nil {
			return nil, fmt.Errorf(
				"field %q (%s) expects %s — got %q (%w)",
				field.DisplayName, field.Type, expectedDateExample(field.Type), s, err,
			)
		}
		return s, nil

	case projects.CustomItemFieldTypeDropdown:
		return coerceOptionValue(field, raw)

	case projects.CustomItemFieldTypeMultiselect:
		return coerceMultiselectValue(field, raw)

	case projects.CustomItemFieldTypeUser:
		return coerceUserValue(field, raw)

	default:
		// Unknown / future field type — pass through and let the API validate.
		return raw, nil
	}
}

func typeMismatch(field projects.CustomItemField, expected string, got any) error {
	return fmt.Errorf("field %q (%s) expects %s — got %T",
		field.DisplayName, field.Type, expected, got)
}

func coerceNumber(raw any) (float64, bool) {
	switch n := raw.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case int:
		return float64(n), true
	case string:
		parsed, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}

func coerceBool(raw any) (bool, bool) {
	switch b := raw.(type) {
	case bool:
		return b, true
	case string:
		switch strings.ToLower(strings.TrimSpace(b)) {
		case "true", "yes", "on", "1":
			return true, true
		case "false", "no", "off", "0":
			return false, true
		}
	}
	return false, false
}

func parseFieldDate(t projects.CustomItemFieldType, value string) (time.Time, error) {
	switch t {
	case projects.CustomItemFieldTypeDate:
		if parsed, err := time.Parse("2006-01-02", value); err == nil {
			return parsed, nil
		}
		return time.Parse(time.RFC3339, value)
	case projects.CustomItemFieldTypeTime:
		return time.Parse("15:04:05", value)
	case projects.CustomItemFieldTypeDateTime:
		return time.Parse(time.RFC3339, value)
	}
	return time.Time{}, fmt.Errorf("unsupported date/time field type %s", t)
}

func expectedDateExample(t projects.CustomItemFieldType) string {
	switch t {
	case projects.CustomItemFieldTypeDate:
		return "YYYY-MM-DD"
	case projects.CustomItemFieldTypeTime:
		return "HH:MM:SS"
	case projects.CustomItemFieldTypeDateTime:
		return "RFC3339 timestamp"
	}
	return "ISO-8601 string"
}

// coerceOptionValue resolves a dropdown choice. Accepts the option label
// (case-insensitive) or the option twId. Returns the twId the API expects.
func coerceOptionValue(field projects.CustomItemField, raw any) (any, error) {
	s, ok := raw.(string)
	if !ok {
		return nil, typeMismatch(field, "option label or twId string", raw)
	}
	needle := strings.TrimSpace(s)
	// twId pass-through.
	for _, opt := range field.Options {
		if opt.TwID == needle {
			return opt.TwID, nil
		}
	}
	// Label lookup.
	lowerNeedle := strings.ToLower(needle)
	var matches []projects.CustomItemFieldOption
	for _, opt := range field.Options {
		if strings.ToLower(opt.Label) == lowerNeedle {
			matches = append(matches, opt)
		}
	}
	switch len(matches) {
	case 0:
		var labels []string
		for _, opt := range field.Options {
			labels = append(labels, opt.Label)
		}
		return nil, fmt.Errorf(
			"field %q has no option %q — known options: %s",
			field.DisplayName, s, strings.Join(labels, ", "),
		)
	case 1:
		return matches[0].TwID, nil
	default:
		var ids []string
		for _, opt := range matches {
			ids = append(ids, fmt.Sprintf("twId=%q id=%d", opt.TwID, opt.ID))
		}
		return nil, fmt.Errorf(
			"field %q has %d options labelled %q (%s) — refer to it by twId",
			field.DisplayName, len(matches), s, strings.Join(ids, ", "),
		)
	}
}

// coerceMultiselectValue resolves each entry in an array of dropdown
// choices.
func coerceMultiselectValue(field projects.CustomItemField, raw any) (any, error) {
	array, ok := raw.([]any)
	if !ok {
		return nil, typeMismatch(field, "array of option labels or twIds", raw)
	}
	out := make([]any, 0, len(array))
	for i, item := range array {
		resolved, err := coerceOptionValue(field, item)
		if err != nil {
			return nil, fmt.Errorf("value[%d]: %w", i, err)
		}
		out = append(out, resolved)
	}
	return out, nil
}

// coerceUserValue accepts a single user ID or an array of user IDs. Email
// resolution is off by default — plan §6 perf rules.
func coerceUserValue(field projects.CustomItemField, raw any) (any, error) {
	switch v := raw.(type) {
	case float64, int64, int, int32:
		return v, nil
	case []any:
		out := make([]int64, 0, len(v))
		for i, item := range v {
			num, ok := coerceNumber(item)
			if !ok {
				return nil, fmt.Errorf("field %q value[%d]: expected user ID, got %T",
					field.DisplayName, i, item)
			}
			out = append(out, int64(num))
		}
		return out, nil
	case string:
		// Allow a numeric string for convenience.
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			return parsed, nil
		}
		return nil, fmt.Errorf("field %q expects a user ID — got %q (email lookup is not supported)",
			field.DisplayName, v)
	}
	return nil, typeMismatch(field, "user ID or array of user IDs", raw)
}

// translateRecordFieldValuesOut converts the wire-format field-values map
// (keyed by field twId, with dropdown values as option twIds) into a
// display-friendly map (keyed by field display name, with dropdown values
// as option labels). Unknown twIds pass through under their raw key.
func translateRecordFieldValuesOut(
	values projects.CustomItemRecordFieldValues,
	fields []projects.CustomItemField,
) projects.CustomItemRecordFieldValues {
	if len(values) == 0 {
		return values
	}
	byTwID := make(map[string]projects.CustomItemField, len(fields))
	for _, field := range fields {
		byTwID[field.TwID] = field
	}
	out := make(projects.CustomItemRecordFieldValues, len(values))
	for twID, value := range values {
		field, ok := byTwID[twID]
		if !ok {
			out[twID] = value
			continue
		}
		out[field.DisplayName] = translateValueOut(field, value)
	}
	return out
}

func translateValueOut(field projects.CustomItemField, value any) any {
	if value == nil || len(field.Options) == 0 {
		return value
	}
	switch field.Type {
	case projects.CustomItemFieldTypeDropdown:
		s, ok := value.(string)
		if !ok {
			return value
		}
		for _, opt := range field.Options {
			if opt.TwID == s {
				return opt.Label
			}
		}
		return value
	case projects.CustomItemFieldTypeMultiselect:
		array, ok := value.([]any)
		if !ok {
			return value
		}
		out := make([]any, 0, len(array))
		for _, item := range array {
			s, ok := item.(string)
			if !ok {
				out = append(out, item)
				continue
			}
			matched := false
			for _, opt := range field.Options {
				if opt.TwID == s {
					out = append(out, opt.Label)
					matched = true
					break
				}
			}
			if !matched {
				out = append(out, item)
			}
		}
		return out
	}
	return value
}
