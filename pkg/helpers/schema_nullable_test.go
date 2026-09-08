package helpers_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/pkg/helpers"
)

// TestDropNullBranches covers the rewrite that makes an optional parameter
// Vertex-parseable — Vertex has no null type member.
func TestDropNullBranches(t *testing.T) {
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
			name: "two-branch anyOf collapses and keeps the annotations",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"page": {
						Description: "Page number.",
						Default:     json.RawMessage(`1`),
						Examples:    []any{"2"},
						AnyOf: []*jsonschema.Schema{
							{Type: "integer", Minimum: ptr(1.0)},
							{Type: "null"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				page := schema.Properties["page"]
				if page.AnyOf != nil {
					t.Errorf("anyOf = %v, want nil", page.AnyOf)
				}
				if page.Type != "integer" {
					t.Errorf("type = %q, want integer", page.Type)
				}
				if page.Minimum == nil || *page.Minimum != 1 {
					t.Errorf("minimum = %v, want 1", page.Minimum)
				}
				if page.Description != "Page number." {
					t.Errorf("description = %q, want it preserved", page.Description)
				}
				if string(page.Default) != "1" {
					t.Errorf("default = %s, want it preserved", page.Default)
				}
				if len(page.Examples) != 1 {
					t.Errorf("examples = %v, want them preserved", page.Examples)
				}
			},
		},
		{
			name: "a genuine composition keeps its remaining branches",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"updated_after": {
						Description: "A moment.",
						AnyOf: []*jsonschema.Schema{
							{Type: "string", Format: "date-time"},
							{Type: "string", Format: "date"},
							{Type: "null"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				branches := schema.Properties["updated_after"].AnyOf
				if len(branches) != 2 {
					t.Fatalf("anyOf has %d branches, want 2", len(branches))
				}
				for _, branch := range branches {
					if branch.Type == "null" {
						t.Error("a null branch survived")
					}
				}
			},
		},
		{
			name: "the property leaves required when its null branch goes",
			schema: &jsonschema.Schema{
				Type:     "object",
				Required: []string{"id", "fields"},
				Properties: map[string]*jsonschema.Schema{
					"id": {Type: "integer"},
					"fields": {
						AnyOf: []*jsonschema.Schema{
							{Type: "array", Items: &jsonschema.Schema{Type: "string"}},
							{Type: "null"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				// Null was the only way to leave `fields` unset, so keeping it
				// required would make it mandatory.
				if slices.Contains(schema.Required, "fields") {
					t.Errorf("required = %v, want fields dropped", schema.Required)
				}
				if !slices.Contains(schema.Required, "id") {
					t.Errorf("required = %v, want id kept", schema.Required)
				}
			},
		},
		{
			name: "a branch that constrains more than the null type is kept",
			schema: &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{
					{Type: "string"},
					{Type: "null", Description: "explicitly unset"},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if len(schema.AnyOf) != 2 {
					t.Errorf("anyOf = %v, want both branches kept", schema.AnyOf)
				}
			},
		},
		{
			name: "an anyOf of nothing but nulls is left alone",
			schema: &jsonschema.Schema{
				AnyOf: []*jsonschema.Schema{{Type: "null"}},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				// An empty anyOf is invalid, and widening to "anything" is worse.
				if len(schema.AnyOf) != 1 {
					t.Errorf("anyOf = %v, want the null branch left in place", schema.AnyOf)
				}
			},
		},
		{
			name:   "a union spelled as a type list drops null",
			schema: &jsonschema.Schema{Types: []string{"null", "string"}, Format: "date"},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if schema.Types != nil {
					t.Errorf("types = %v, want nil", schema.Types)
				}
				if schema.Type != "string" {
					t.Errorf("type = %q, want string", schema.Type)
				}
			},
		},
		{
			name: "nested properties and array items are rewritten too",
			schema: &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"assignees": {
						AnyOf: []*jsonschema.Schema{
							{
								Type: "object",
								Properties: map[string]*jsonschema.Schema{
									"user_ids": {
										AnyOf: []*jsonschema.Schema{
											{Type: "array", Items: &jsonschema.Schema{
												AnyOf: []*jsonschema.Schema{
													{Type: "integer"},
													{Type: "null"},
												},
											}},
											{Type: "null"},
										},
									},
								},
							},
							{Type: "null"},
						},
					},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				assignees := schema.Properties["assignees"]
				if assignees.Type != "object" {
					t.Fatalf("assignees type = %q, want object", assignees.Type)
				}
				userIDs := assignees.Properties["user_ids"]
				if userIDs.Type != "array" {
					t.Fatalf("user_ids type = %q, want array", userIDs.Type)
				}
				if userIDs.Items.Type != "integer" {
					t.Errorf("user_ids items type = %q, want integer", userIDs.Items.Type)
				}
			},
		},
		{
			name: "a conflict leaves the single branch rather than losing a value",
			schema: &jsonschema.Schema{
				Type: "string",
				AnyOf: []*jsonschema.Schema{
					{Type: "integer"},
					{Type: "null"},
				},
			},
			check: func(t *testing.T, schema *jsonschema.Schema) {
				if len(schema.AnyOf) != 1 || schema.AnyOf[0].Type != "integer" {
					t.Errorf("anyOf = %v, want the integer branch kept in place", schema.AnyOf)
				}
				if schema.Type != "string" {
					t.Errorf("type = %q, want the parent's own type untouched", schema.Type)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, helpers.DropNullBranches(tt.schema))
		})
	}
}

// TestDropNullBranchesLeavesNoNullBehind is the blanket assertion: nothing
// anywhere declares the null type afterwards.
func TestDropNullBranchesLeavesNoNullBehind(t *testing.T) {
	schema := helpers.DropNullBranches(&jsonschema.Schema{
		Type:     "object",
		Required: []string{"a", "b", "c"},
		Properties: map[string]*jsonschema.Schema{
			"a": {AnyOf: []*jsonschema.Schema{{Type: "string"}, {Type: "null"}}},
			"b": {AnyOf: []*jsonschema.Schema{
				{Type: "array", Items: &jsonschema.Schema{Type: "string", Enum: []any{"x"}}},
				{Type: "null"},
			}},
			"c": {Types: []string{"integer", "null"}},
		},
	})

	if paths := nullTypePaths(schema, ""); len(paths) > 0 {
		t.Errorf("null type still declared at %v", paths)
	}
	if len(schema.Required) != 0 {
		t.Errorf("required = %v, want every nullable property dropped", schema.Required)
	}
}

// nullTypePaths reports where the null type is still declared, in either form.
func nullTypePaths(s *jsonschema.Schema, path string) []string {
	if s == nil {
		return nil
	}
	var found []string
	if s.Type == "null" || slices.Contains(s.Types, "null") {
		found = append(found, path+"/type")
	}
	for name, property := range s.Properties {
		found = append(found, nullTypePaths(property, path+"/properties/"+name)...)
	}
	found = append(found, nullTypePaths(s.Items, path+"/items")...)
	found = append(found, nullTypePaths(s.AdditionalProperties, path+"/additionalProperties")...)
	for _, branches := range [][]*jsonschema.Schema{s.AnyOf, s.OneOf, s.AllOf} {
		for i, branch := range branches {
			found = append(found, nullTypePaths(branch, fmt.Sprintf("%s/branch/%d", path, i))...)
		}
	}
	return found
}

func ptr[T any](v T) *T { return &v }
