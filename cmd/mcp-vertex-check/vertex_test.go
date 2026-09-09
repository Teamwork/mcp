package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// schema decodes a JSON schema the way the checker receives it: the wire form,
// not the Go struct that produced it.
func schema(t *testing.T, raw string) any {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("bad test schema: %v", err)
	}
	return decoded
}

func TestCheckTool(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		output     string
		wantRules  []string
		wantNoRule string
	}{
		{
			name:      "a clean schema passes",
			input:     `{"type":"object","properties":{"id":{"type":"integer"}},"required":["id"]}`,
			wantRules: nil,
		},
		{
			name:      "a type array cannot be parsed",
			input:     `{"type":"object","properties":{"due":{"type":["string","null"]}}}`,
			wantRules: []string{"type-array"},
		},
		{
			name:      "a scalar null type has no enum member",
			input:     `{"type":"object","properties":{"due":{"anyOf":[{"type":"string"},{"type":"null"}]}}}`,
			wantRules: []string{"null-type"},
		},
		{
			name:      "examples is not a Vertex field",
			input:     `{"type":"object","properties":{"due":{"type":"string","examples":["2026-08-03"]}}}`,
			wantRules: []string{"unknown-field"},
		},
		{
			name:      "oneOf is not a Vertex field",
			input:     `{"type":"object","properties":{"x":{"oneOf":[{"type":"string"}]}}}`,
			wantRules: []string{"unknown-field"},
		},
		{
			name:      "an object-valued additionalProperties warns",
			input:     `{"type":"object","additionalProperties":{"type":"string"}}`,
			wantRules: []string{"additional-properties-object"},
		},
		{
			name:       "a boolean additionalProperties is fine",
			input:      `{"type":"object","additionalProperties":false}`,
			wantNoRule: "additional-properties-object",
		},
		{
			name:      "an undocumented format warns",
			input:     `{"type":"object","properties":{"d":{"type":"string","format":"date"}}}`,
			wantRules: []string{"undocumented-format"},
		},
		{
			name:       "a documented format is fine",
			input:      `{"type":"object","properties":{"d":{"type":"string","format":"date-time"}}}`,
			wantNoRule: "undocumented-format",
		},
		{
			name:      "output schemas are checked too",
			input:     `{"type":"object"}`,
			output:    `{"type":"object","properties":{"deletedAt":{"type":["string","null"]}}}`,
			wantRules: []string{"type-array"},
		},
		{
			name: "nested properties and items are reached",
			input: `{"type":"object","properties":{"rows":{"type":"array","items":` +
				`{"type":"object","properties":{"n":{"type":["integer","null"]}}}}}}`,
			wantRules: []string{"type-array"},
		},
		{
			name:      "$defs below the root is rejected",
			input:     `{"type":"object","properties":{"x":{"$defs":{"Y":{"type":"string"}}}}}`,
			wantRules: []string{"defs-not-at-root"},
		},
		{
			name:       "$defs at the root is allowed",
			input:      `{"type":"object","$defs":{"Y":{"type":"string"}}}`,
			wantNoRule: "defs-not-at-root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output any
			if tt.output != "" {
				output = schema(t, tt.output)
			}
			findings := CheckTool("tool", schema(t, tt.input), output)

			got := map[string]bool{}
			for _, f := range findings {
				got[f.Rule] = true
			}
			for _, want := range tt.wantRules {
				if !got[want] {
					t.Errorf("rule %q not reported; got %v", want, findings)
				}
			}
			if tt.wantRules == nil && tt.wantNoRule == "" && len(findings) > 0 {
				t.Errorf("expected no findings, got %v", findings)
			}
			if tt.wantNoRule != "" && got[tt.wantNoRule] {
				t.Errorf("rule %q should not fire; got %v", tt.wantNoRule, findings)
			}
		})
	}
}

// TestCheckToolPathsPointAtTheNode keeps the reported path usable: it has to
// name the offending node, not just the tool.
func TestCheckToolPathsPointAtTheNode(t *testing.T) {
	findings := CheckTool("twprojects-get_task", schema(t,
		`{"type":"object","properties":{"a":{"type":"object","properties":{"b":{"type":["string","null"]}}}}}`), nil)
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %v", len(findings), findings)
	}
	if want := "/properties/a/properties/b/type"; findings[0].Path != want {
		t.Errorf("path = %q, want %q", findings[0].Path, want)
	}
	if findings[0].Schema != "inputSchema" {
		t.Errorf("schema = %q, want inputSchema", findings[0].Schema)
	}
}

// TestCapturedToolsAcceptsEveryEnvelope covers the three shapes a saved
// tools/list body arrives in.
func TestCapturedToolsAcceptsEveryEnvelope(t *testing.T) {
	body := `{"name":"t","inputSchema":{"type":"object"}}`
	for name, raw := range map[string]string{
		"jsonrpc response": `{"jsonrpc":"2.0","id":1,"result":{"tools":[` + body + `]}}`,
		"bare result":      `{"tools":[` + body + `]}`,
		"tools array":      `[` + body + `]`,
	} {
		t.Run(name, func(t *testing.T) {
			tools, err := capturedTools([]byte(raw))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tools) != 1 || tools[0].name != "t" {
				t.Errorf("got %+v, want one tool named t", tools)
			}
		})
	}
}

// TestRegisteredToolsReadsTheWireForm is the guard that this checker sees what
// a client sees: the schemas must come back as decoded JSON, so a Go-side
// Types slice shows up as a type array rather than being invisible.
func TestRegisteredToolsReadsTheWireForm(t *testing.T) {
	tools, err := registeredTools(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("no tools were registered")
	}
	for _, tool := range tools {
		if _, ok := tool.inputSchema.(map[string]any); !ok {
			t.Fatalf("%s input schema decoded to %T, want a JSON object", tool.name, tool.inputSchema)
		}
	}
}

// TestVertexVariantIsClean is the gate: every schema a Gemini Enterprise client
// is served must be parseable by Vertex, warnings included. This is the test to
// keep green — the published surface deliberately is not (see below).
func TestVertexVariantIsClean(t *testing.T) {
	tools, err := registeredTools(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tool := range tools {
		for _, f := range CheckTool(tool.name, tool.inputSchema, tool.outputSchema) {
			t.Errorf("%s", f)
		}
	}
}

// TestPublishedSurfaceStillCarriesOutputSchemaNulls stops the gate above from
// passing vacuously. The published output schemas declare type unions because
// the responses really do carry null, so they cannot be Vertex-safe — which is
// the whole reason the variant exists. If this ever goes green, the variant may
// no longer be needed.
func TestPublishedSurfaceStillCarriesOutputSchemaNulls(t *testing.T) {
	tools, err := registeredTools(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var typeArrays int
	for _, tool := range tools {
		for _, f := range CheckTool(tool.name, tool.inputSchema, tool.outputSchema) {
			if f.Rule == "type-array" && f.Schema == "outputSchema" {
				typeArrays++
			}
		}
	}
	if typeArrays == 0 {
		t.Error("no output-schema type unions found; the Vertex variant's reason for existing is gone")
	}
}

// TestCheckToolName covers the FunctionDeclaration.name contract, which is a
// documented hard limit rather than a schema concern.
func TestCheckToolName(t *testing.T) {
	empty := schema(t, `{"type":"object"}`)
	tests := []struct {
		name     string
		tool     string
		wantRule string
	}{
		{name: "a normal name passes", tool: "twprojects-move_task_to_workflow_stage"},
		{
			name:     "over 64 characters",
			tool:     "twprojects-" + strings.Repeat("a", 60),
			wantRule: "name-too-long",
		},
		{name: "a leading digit", tool: "1tool", wantRule: "name-charset"},
		{name: "a space", tool: "two words", wantRule: "name-charset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			for _, f := range CheckTool(tt.tool, empty, nil) {
				if f.Schema == "name" {
					got = f.Rule
				}
			}
			if got != tt.wantRule {
				t.Errorf("rule = %q, want %q", got, tt.wantRule)
			}
		})
	}
}
