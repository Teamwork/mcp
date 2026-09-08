package helpers_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
)

// TestStrictSchema covers the OpenAI strict-mode variant: every property
// required, optionality expressed as null.
func TestStrictSchema(t *testing.T) {
	strict := helpers.StrictSchema(&jsonschema.Schema{
		Type:     "object",
		Required: []string{"id"},
		Properties: map[string]*jsonschema.Schema{
			"id":       {Type: "integer", Description: "Mandatory."},
			"page":     {Type: "integer", Default: json.RawMessage(`1`)},
			"order_by": {Type: "string", Enum: []any{"asc", "desc"}},
			"fields":   {Type: "array", Items: &jsonschema.Schema{Type: "string", Enum: []any{"id", "name"}}},
			"date":     {AnyOf: []*jsonschema.Schema{{Type: "string", Format: "date-time"}, {Type: "string", Format: "date"}}},
			"assignees": {Type: "object", Properties: map[string]*jsonschema.Schema{
				"user_ids": {Type: "array", Items: &jsonschema.Schema{Type: "integer"}},
			}},
		},
	})

	t.Run("every property is required", func(t *testing.T) {
		want := []string{"assignees", "date", "fields", "id", "order_by", "page"}
		if !slices.Equal(strict.Required, want) {
			t.Errorf("required = %v, want %v", strict.Required, want)
		}
	})

	t.Run("additionalProperties is false", func(t *testing.T) {
		if strict.AdditionalProperties == nil {
			t.Fatal("additionalProperties is not set")
		}
		raw, err := json.Marshal(strict.AdditionalProperties)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(raw) != "false" {
			t.Errorf("additionalProperties marshalled to %s, want false", raw)
		}
	})

	t.Run("a mandatory property is not weakened", func(t *testing.T) {
		if id := strict.Properties["id"]; id.Type != "integer" || id.Types != nil {
			t.Errorf("id type = %q/%v, want integer and no union", id.Type, id.Types)
		}
	})

	t.Run("an optional property becomes nullable", func(t *testing.T) {
		page := strict.Properties["page"]
		if !slices.Equal(page.Types, []string{"integer", "null"}) {
			t.Errorf("page types = %v, want [integer null]", page.Types)
		}
		// Both set at once makes marshalling fail.
		if page.Type != "" {
			t.Errorf("page type = %q, want it cleared in favour of types", page.Type)
		}
	})

	t.Run("a nullable enum permits null", func(t *testing.T) {
		// Without null in the enum the widened type is unreachable: the schema
		// registers fine, then rejects every call that omits the parameter.
		orderBy := strict.Properties["order_by"]
		if !slices.Contains(orderBy.Enum, nil) {
			t.Errorf("order_by enum = %v, want null included", orderBy.Enum)
		}
	})

	t.Run("an item enum is left alone", func(t *testing.T) {
		// The array is what became optional, not its elements.
		items := strict.Properties["fields"].Items
		if slices.Contains(items.Enum, nil) {
			t.Errorf("fields items enum = %v, want null NOT added", items.Enum)
		}
	})

	t.Run("a composite gets a null branch", func(t *testing.T) {
		date := strict.Properties["date"]
		if len(date.AnyOf) != 3 {
			t.Fatalf("date has %d branches, want 3", len(date.AnyOf))
		}
		if date.AnyOf[2].Type != "null" {
			t.Errorf("date last branch = %q, want null", date.AnyOf[2].Type)
		}
	})

	t.Run("default is stripped", func(t *testing.T) {
		if strict.Properties["page"].Default != nil {
			t.Errorf("page default = %s, want it dropped", strict.Properties["page"].Default)
		}
	})

	t.Run("nested objects get the same treatment", func(t *testing.T) {
		assignees := strict.Properties["assignees"]
		if !slices.Equal(assignees.Required, []string{"user_ids"}) {
			t.Errorf("assignees required = %v, want [user_ids]", assignees.Required)
		}
		if assignees.AdditionalProperties == nil {
			t.Error("assignees is missing additionalProperties: false")
		}
		if userIDs := assignees.Properties["user_ids"]; !slices.Equal(userIDs.Types, []string{"array", "null"}) {
			t.Errorf("assignees.user_ids types = %v, want [array null]", userIDs.Types)
		}
	})
}

// TestStrictSchemaDoesNotTouchTheOriginal is the regression that matters most:
// CloneSchemas shares plain slices, so Required and Enum need cloning by hand
// or the registered surface is corrupted for every other client.
func TestStrictSchemaDoesNotTouchTheOriginal(t *testing.T) {
	published := &jsonschema.Schema{
		Type:     "object",
		Required: make([]string, 0, 8), // spare capacity: append would write in place
		Properties: map[string]*jsonschema.Schema{
			"order_by": {Type: "string", Enum: []any{"asc", "desc"}},
			"page":     {Type: "integer"},
		},
	}
	published.Required = append(published.Required, "page")

	before, err := json.Marshal(published)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	_ = helpers.StrictSchema(published)

	after, err := json.Marshal(published)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("the published schema was modified:\n before %s\n after  %s", before, after)
	}
}

// TestStrictToolsClonesTheTools checks the tool is copied too, not just the
// schema.
func TestStrictToolsClonesTheTools(t *testing.T) {
	published := &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{"page": {Type: "integer"}},
	}
	tool := &mcp.Tool{Name: "twprojects-list_tasks", InputSchema: published}

	strict := helpers.StrictTools([]*mcp.Tool{tool})
	if len(strict) != 1 {
		t.Fatalf("got %d tools, want 1", len(strict))
	}
	if strict[0] == tool {
		t.Error("the tool itself was not copied")
	}
	if tool.InputSchema != any(published) {
		t.Error("the registered tool no longer points at its published schema")
	}
	if strict[0].Name != tool.Name {
		t.Errorf("name = %q, want it carried over", strict[0].Name)
	}
	if strict[0].InputSchema == any(published) {
		t.Error("the strict tool shares the published schema")
	}
}

// TestStrictToolsPassesThroughForeignSchemas: a non-jsonschema input schema is
// passed through, not dropped.
func TestStrictToolsPassesThroughForeignSchemas(t *testing.T) {
	tool := &mcp.Tool{Name: "twprojects-list_tasks", InputSchema: map[string]any{"type": "object"}}
	strict := helpers.StrictTools([]*mcp.Tool{tool})
	if len(strict) != 1 || strict[0] != tool {
		t.Errorf("got %v, want the tool passed through unchanged", strict)
	}
}
