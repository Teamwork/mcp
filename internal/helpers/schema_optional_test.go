package helpers_test

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/helpers"
)

func TestWithOptionalFields(t *testing.T) {
	tests := []struct {
		name   string
		schema *jsonschema.Schema
		check  func(t *testing.T, schema *jsonschema.Schema)
	}{
		{
			name:   "nil schema is a no-op",
			schema: nil,
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema != nil {
					t.Error("expected nil schema")
				}
			},
		},
		{
			name: "top-level required is cleared",
			schema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "name"},
				Properties: map[string]*jsonschema.Schema{
					"id":   {Type: "integer"},
					"name": {Type: "string"},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Required != nil {
					t.Errorf("expected nil required, got %v", schema.Required)
				}
			},
		},
		{
			name: "nested object required is cleared",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task": {
						Type:     "object",
						Required: []string{"id", "name", "status"},
						Properties: map[string]*jsonschema.Schema{
							"id":     {Type: "integer"},
							"name":   {Type: "string"},
							"status": {Type: "string"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["task"].Required != nil {
					t.Errorf("expected nested required to be nil, got %v", schema.Properties["task"].Required)
				}
			},
		},
		{
			name: "array items required is cleared",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tasks": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type:     "object",
							Required: []string{"id"},
							Properties: map[string]*jsonschema.Schema{
								"id": {Type: "integer"},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["tasks"].Items.Required != nil {
					t.Errorf("expected items required to be nil, got %v", schema.Properties["tasks"].Items.Required)
				}
			},
		},
		{
			name: "anyOf branches required is cleared",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"value": {
						AnyOf: []*jsonschema.Schema{
							{
								Type:     "object",
								Required: []string{"a"},
								Properties: map[string]*jsonschema.Schema{
									"a": {Type: "string"},
								},
							},
							{Type: "null"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["value"].AnyOf[0].Required != nil {
					t.Errorf("expected anyOf branch required to be nil, got %v", schema.Properties["value"].AnyOf[0].Required)
				}
			},
		},
		{
			name: "top-level additionalProperties is cleared",
			schema: &jsonschema.Schema{
				Type:                 "object",
				AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}}, // falseSchema()
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer"},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.AdditionalProperties != nil {
					t.Errorf("expected additionalProperties to be nil, got %#v", schema.AdditionalProperties)
				}
			},
		},
		{
			name: "nested additionalProperties is cleared",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task": {
						Type:                 "object",
						AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
						Properties: map[string]*jsonschema.Schema{
							"id": {Type: "integer"},
						},
					},
					"tasks": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type:                 "object",
							AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
							Properties: map[string]*jsonschema.Schema{
								"id": {Type: "integer"},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["task"].AdditionalProperties != nil {
					t.Errorf("expected nested additionalProperties to be nil, got %#v",
						schema.Properties["task"].AdditionalProperties)
				}
				if schema.Properties["tasks"].Items.AdditionalProperties != nil {
					t.Errorf("expected items additionalProperties to be nil, got %#v",
						schema.Properties["tasks"].Items.AdditionalProperties)
				}
			},
		},
		{
			name: "returns same schema for chaining",
			schema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id"},
			},
			check: func(_ *testing.T, _ *jsonschema.Schema) {
				// nothing extra; sameness is checked at the test boundary
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := tt.schema
			got := helpers.WithOptionalFields(input)
			if input != nil && got != input {
				t.Errorf("expected WithOptionalFields to return the same pointer for chaining")
			}
			tt.check(t, got)
		})
	}
}
