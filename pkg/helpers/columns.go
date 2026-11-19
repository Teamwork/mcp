package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The list formats a client can ask for. ListFormatObjects is the default: each
// record is a JSON object. ListFormatColumns sends each list of records as
// column names plus rows of values. See ToColumns.
const (
	ListFormatObjects = "objects"
	ListFormatColumns = "columns"
)

const (
	// includedKey is the top-level key that holds the sideloaded records of a
	// list result, as a map of record type to records.
	includedKey = "included"

	columnsDescription = "Records in columns format. columns names each attribute once, " +
		"and each row holds the values of one record in the same order. A null cell means " +
		"the attribute is not set. A column named in relationships holds the ID of a record " +
		"of that type, or a list of such IDs, in place of an {id, type} object."
)

// ColumnSpec names the parts of a list result that the columns format changes.
// It is read from the tool's output schema (see NewColumnSpec), so the result
// and the schema published for it always change together.
type ColumnSpec struct {
	// Lists are the top-level keys whose value is a list of records.
	Lists []string
	// Included is true when the result can carry sideloaded records under
	// "included". Every record type there is changed.
	Included bool
}

// IsZero reports whether the spec changes nothing.
func (s ColumnSpec) IsZero() bool {
	return len(s.Lists) == 0 && !s.Included
}

// NewColumnSpec reads the column spec from a tool's output schema. A top-level
// property that is an array of objects is a list of records, and a top-level
// "included" object holds sideloaded records. A schema of another type gives a
// zero spec.
func NewColumnSpec(outputSchema any) ColumnSpec {
	var spec ColumnSpec
	schema, ok := outputSchema.(*jsonschema.Schema)
	if !ok || schema == nil {
		return spec
	}
	for name, property := range schema.Properties {
		switch {
		case name == includedKey && schemaHasType(property, "object"):
			spec.Included = true
		case isRecordListSchema(property):
			spec.Lists = append(spec.Lists, name)
		}
	}
	slices.Sort(spec.Lists)
	return spec
}

// isRecordListSchema reports whether a schema is an array of objects.
func isRecordListSchema(schema *jsonschema.Schema) bool {
	return schema != nil && schemaHasType(schema, "array") && schema.Items != nil &&
		schemaHasType(schema.Items, "object")
}

// schemaHasType reports whether a schema allows the given type.
func schemaHasType(schema *jsonschema.Schema, typ string) bool {
	return schema.Type == typ || slices.Contains(schema.Types, typ)
}

// ColumnsSchema returns a copy of a list tool's output schema that describes the
// columns format: each property the spec names is replaced by the columns
// object. The input is never modified.
func ColumnsSchema(schema *jsonschema.Schema, spec ColumnSpec) *jsonschema.Schema {
	if schema == nil || spec.IsZero() {
		return schema
	}
	columns := schema.CloneSchemas()
	for _, name := range spec.Lists {
		nullable := schemaHasType(columns.Properties[name], "null")
		columns.Properties[name] = columnsObjectSchema(nullable)
	}
	if included, ok := columns.Properties[includedKey]; ok && spec.Included {
		for name := range included.Properties {
			included.Properties[name] = columnsObjectSchema(false)
		}
		included.AdditionalProperties = columnsObjectSchema(false)
	}
	return columns
}

// columnsObjectSchema is the schema of one list of records in columns format.
func columnsObjectSchema(nullable bool) *jsonschema.Schema {
	schema := &jsonschema.Schema{
		Type:        "object",
		Description: columnsDescription,
		Properties: map[string]*jsonschema.Schema{
			"columns": {Type: "array", Items: &jsonschema.Schema{Type: "string"}},
			"rows": {
				Type:  "array",
				Items: &jsonschema.Schema{Type: "array", Items: &jsonschema.Schema{}},
			},
			"relationships": {
				Type:                 "object",
				AdditionalProperties: &jsonschema.Schema{Type: "string"},
			},
		},
		Required: []string{"columns", "rows"},
	}
	if nullable {
		schema.Type = ""
		schema.Types = []string{"null", "object"}
	}
	return schema
}

// ColumnsTools returns a copy of tools in which each tool named in specs
// publishes the columns output schema. The other tools, and a tool that has no
// output schema, are returned as they are. The input is never modified.
func ColumnsTools(tools []*mcp.Tool, specs map[string]ColumnSpec) []*mcp.Tool {
	columns := make([]*mcp.Tool, len(tools))
	for i, tool := range tools {
		columns[i] = tool
		spec, ok := specs[tool.Name]
		if !ok {
			continue
		}
		schema, ok := tool.OutputSchema.(*jsonschema.Schema)
		if !ok || schema == nil {
			continue
		}
		clone := *tool
		clone.OutputSchema = ColumnsSchema(schema, spec)
		columns[i] = &clone
	}
	return columns
}

// ColumnsToolResult changes a list tool result into the columns format, in its
// text and its structured content. A result that is not a single JSON text
// block, and an error result, are left as they are.
func ColumnsToolResult(result *mcp.CallToolResult, spec ColumnSpec) error {
	if result == nil || result.IsError || len(result.Content) != 1 || spec.IsZero() {
		return nil
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return nil
	}
	columns, err := ToColumns([]byte(text.Text), spec)
	if err != nil {
		return err
	}
	var structured any
	if err := json.Unmarshal(columns, &structured); err != nil {
		return fmt.Errorf("failed to decode columns result: %w", err)
	}
	text.Text = string(columns)
	if result.StructuredContent != nil {
		result.StructuredContent = structured
	}
	return nil
}

// ToColumns changes the lists of records in a JSON list result into the
// columns format:
//
//	{"tasks": [{"id": 1, "name": "a", "tasklist": {"id": 7, "type": "tasklists"}}]}
//
// becomes
//
//	{"tasks": {"columns": ["id", "name", "tasklist"], "rows": [[1, "a", 7]],
//	           "relationships": {"tasklist": "tasklists"}}}
//
// Each attribute name is sent once for each list, not once for each record. A
// record that does not have an attribute gets null in that column. A column
// whose relationships ({"id": N, "type": T}, or a list of them) all have the
// same type holds the IDs only, and relationships names the type. A column with
// more than one type keeps the objects, so no type is lost.
//
// The keys in spec.Lists are changed, and every record type under "included"
// when spec.Included is true. Other keys, such as "meta", are left as they
// are. Numbers are copied as written. Data that is not a JSON object is
// returned unchanged.
func ToColumns(data []byte, spec ColumnSpec) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil || document == nil {
		return data, nil
	}
	for _, key := range spec.Lists {
		records, ok := recordList(document[key])
		if ok {
			document[key] = recordsToColumns(records)
		}
	}
	if spec.Included {
		includedToColumns(document[includedKey])
	}
	return json.Marshal(document)
}

// includedToColumns changes each record type under "included" in place. A
// sideload is a map of record ID to record; the ID is also in the record, so
// the map key is dropped.
func includedToColumns(value any) {
	included, ok := value.(map[string]any)
	if !ok {
		return
	}
	for name, section := range included {
		if records, ok := recordList(section); ok {
			included[name] = recordsToColumns(records)
		}
	}
}

// recordList returns the records in a list of objects, or in a map of ID to
// object. A map is read in the order of its keys, so the rows are in a fixed
// order. It reports false for any other value, or when an element is not an
// object.
func recordList(value any) ([]map[string]any, bool) {
	switch typed := value.(type) {
	case []any:
		return objectsOf(typed)
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sortIDs(keys)
		values := make([]any, len(keys))
		for i, key := range keys {
			values[i] = typed[key]
		}
		return objectsOf(values)
	}
	return nil, false
}

// objectsOf returns the elements of a list when every element is an object.
func objectsOf(values []any) ([]map[string]any, bool) {
	records := make([]map[string]any, len(values))
	for i, value := range values {
		record, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		records[i] = record
	}
	return records, true
}

// sortIDs sorts map keys as numbers when they are numeric IDs, so "9" comes
// before "10", and as text otherwise.
func sortIDs(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) != len(keys[j]) && isDigits(keys[i]) && isDigits(keys[j]) {
			return len(keys[i]) < len(keys[j])
		}
		return keys[i] < keys[j]
	})
}

// isDigits reports whether s is not empty and holds only the digits 0–9.
func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// recordsToColumns builds the columns object for a list of records.
func recordsToColumns(records []map[string]any) map[string]any {
	names := columnNames(records)
	relationships := map[string]any{}
	flatten := map[string]bool{}
	for _, name := range names {
		if typ, ok := relationshipType(records, name); ok {
			relationships[name] = typ
			flatten[name] = true
		}
	}
	rows := make([]any, len(records))
	for i, record := range records {
		row := make([]any, len(names))
		for j, name := range names {
			row[j] = record[name]
			if flatten[name] {
				row[j] = relationshipIDs(record[name])
			}
		}
		rows[i] = row
	}
	columns := map[string]any{"columns": names, "rows": rows}
	if len(relationships) > 0 {
		columns["relationships"] = relationships
	}
	return columns
}

// columnNames returns every attribute name in the records, "id" first and the
// others in alphabetical order.
func columnNames(records []map[string]any) []string {
	seen := map[string]bool{}
	for _, record := range records {
		for name := range record {
			seen[name] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		if name != "id" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if seen["id"] {
		names = slices.Insert(names, 0, "id")
	}
	return names
}

// relationshipType returns the one record type that every value in a column
// refers to. A value can be null, a relationship, or a list of relationships.
// It reports false when the column holds any other value, holds more than one
// type, or holds no relationship at all.
func relationshipType(records []map[string]any, name string) (string, bool) {
	var found string
	for _, record := range records {
		types, ok := relationshipTypes(record[name])
		if !ok {
			return "", false
		}
		for _, typ := range types {
			if found != "" && typ != found {
				return "", false
			}
			found = typ
		}
	}
	return found, found != ""
}

// relationshipTypes returns the record types one value refers to. It reports
// false when the value is not null, a relationship, or a list of
// relationships.
func relationshipTypes(value any) ([]string, bool) {
	switch typed := value.(type) {
	case nil:
		return nil, true
	case map[string]any:
		typ, ok := relationshipOf(typed)
		return []string{typ}, ok
	case []any:
		types := make([]string, 0, len(typed))
		for _, element := range typed {
			object, ok := element.(map[string]any)
			if !ok {
				return nil, false
			}
			typ, ok := relationshipOf(object)
			if !ok {
				return nil, false
			}
			types = append(types, typ)
		}
		return types, true
	}
	return nil, false
}

// relationshipOf returns the type of a relationship: an object with exactly the
// keys "id" and "type", where the type is text.
func relationshipOf(object map[string]any) (string, bool) {
	if len(object) != 2 || object["id"] == nil {
		return "", false
	}
	typ, ok := object["type"].(string)
	return typ, ok && typ != ""
}

// relationshipIDs replaces a relationship with its ID, and a list of
// relationships with the list of their IDs. relationshipType has already
// checked the value.
func relationshipIDs(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return typed["id"]
	case []any:
		ids := make([]any, len(typed))
		for i, element := range typed {
			ids[i] = element.(map[string]any)["id"]
		}
		return ids
	}
	return value
}
