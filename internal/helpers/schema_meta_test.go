package helpers_test

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/helpers"
)

func TestWithMetaWebLinkSchema(t *testing.T) {
	tests := []struct {
		name   string
		schema *jsonschema.Schema
		check  func(t *testing.T, schema *jsonschema.Schema)
	}{
		{
			name:   "nil schema returns nil",
			schema: nil,
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema != nil {
					t.Error("expected nil schema")
				}
			},
		},
		{
			name:   "schema without properties returns unchanged",
			schema: &jsonschema.Schema{Type: "object"},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties != nil {
					t.Error("expected nil properties")
				}
			},
		},
		{
			name: "top-level array without items is left unchanged",
			schema: &jsonschema.Schema{
				Type: "array",
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Items != nil {
					t.Error("items schema should remain nil")
				}
			},
		},
		{
			name: "top-level array with items without properties is left unchanged",
			schema: &jsonschema.Schema{
				Type:  "array",
				Items: &jsonschema.Schema{Type: "string"},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Items.Properties != nil {
					t.Error("items properties should remain nil")
				}
			},
		},
		{
			name: "single entity object gets meta.webLink",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"id":   {Type: "integer"},
							"name": {Type: "string"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				task := schema.Properties["task"]
				if task == nil {
					t.Fatal("expected task property")
					return
				}
				meta := task.Properties["meta"]
				if meta == nil {
					t.Fatal("expected meta property on task")
					return
				}
				if meta.Type != "object" {
					t.Errorf("expected meta type 'object', got %q", meta.Type)
				}
				webLink := meta.Properties["webLink"]
				if webLink == nil {
					t.Fatal("expected webLink property in meta")
					return
				}
				if webLink.Type != "string" {
					t.Errorf("expected webLink type 'string', got %q", webLink.Type)
				}
				if webLink.Description == "" {
					t.Error("expected webLink to have a description")
				}
			},
		},
		{
			name: "array of entities gets meta.webLink on items",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tasks": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"id":   {Type: "integer"},
								"name": {Type: "string"},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				items := schema.Properties["tasks"].Items
				if items == nil {
					t.Fatal("expected tasks items")
					return
				}
				meta := items.Properties["meta"]
				if meta == nil {
					t.Fatal("expected meta property on task items")
					return
				}
				webLink := meta.Properties["webLink"]
				if webLink == nil {
					t.Fatal("expected webLink property in meta")
				}
			},
		},
		{
			name: "nullable array gets meta.webLink on items",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"tasks": {
						Types: []string{"null", "array"},
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"id": {Type: "integer"},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				items := schema.Properties["tasks"].Items
				meta := items.Properties["meta"]
				if meta == nil {
					t.Fatal("expected meta property on nullable array items")
					return
				}
				if meta.Properties["webLink"] == nil {
					t.Fatal("expected webLink in meta")
				}
			},
		},
		{
			name: "known root fields are skipped",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"meta": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"page": {Type: "object"},
						},
					},
					"included": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"id": {Type: "integer"},
							},
						},
					},
					"task": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"id": {Type: "integer"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				// "meta" root field should not be modified
				rootMeta := schema.Properties["meta"]
				if _, hasWebLink := rootMeta.Properties["webLink"]; hasWebLink {
					t.Error("root meta should not get webLink")
				}
				// "included" should not be modified
				includedItems := schema.Properties["included"].Items
				if _, hasMeta := includedItems.Properties["meta"]; hasMeta {
					t.Error("included items should not get meta")
				}
				// "task" should get meta
				task := schema.Properties["task"]
				if task.Properties["meta"] == nil {
					t.Error("task should get meta")
				}
			},
		},
		{
			name: "existing meta with webLink is not overwritten",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"id": {Type: "integer"},
							"meta": {
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"webLink": {
										Type:        "string",
										Description: "custom link",
									},
								},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				meta := schema.Properties["task"].Properties["meta"]
				webLink := meta.Properties["webLink"]
				if webLink.Description != "custom link" {
					t.Errorf("expected existing webLink to be preserved, got %q", webLink.Description)
				}
			},
		},
		{
			name: "existing meta without webLink gets webLink merged",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"task": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"id": {Type: "integer"},
							"meta": {
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"customField": {Type: "string"},
								},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				meta := schema.Properties["task"].Properties["meta"]
				if meta.Properties["customField"] == nil {
					t.Error("expected existing customField to be preserved")
				}
				if meta.Properties["webLink"] == nil {
					t.Error("expected webLink to be merged into existing meta")
				}
			},
		},
		{
			name: "multiple entity properties each get meta",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"project": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"id": {Type: "integer"},
						},
					},
					"tasks": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"id": {Type: "integer"},
							},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["project"].Properties["meta"] == nil {
					t.Error("expected project to get meta")
				}
				if schema.Properties["tasks"].Items.Properties["meta"] == nil {
					t.Error("expected task items to get meta")
				}
			},
		},
		{
			name: "non-object non-array properties are left unchanged",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"count": {Type: "integer"},
					"name":  {Type: "string"},
					"task": {
						Type: "object",
						Properties: map[string]*jsonschema.Schema{
							"id": {Type: "integer"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["count"].Type != "integer" {
					t.Error("count should remain integer")
				}
				if schema.Properties["name"].Type != "string" {
					t.Error("name should remain string")
				}
				if schema.Properties["task"].Properties["meta"] == nil {
					t.Error("task should get meta")
				}
			},
		},
		{
			name: "array without items is left unchanged",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"items": {
						Type: "array",
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["items"].Items != nil {
					t.Error("items schema should remain nil")
				}
			},
		},
		{
			name: "array with items that have no properties is left unchanged",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"items": {
						Type:  "array",
						Items: &jsonschema.Schema{Type: "string"},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Properties["items"].Items.Properties != nil {
					t.Error("items properties should remain nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.WithMetaWebLinkSchema(tt.schema)
			tt.check(t, result)
		})
	}
}
