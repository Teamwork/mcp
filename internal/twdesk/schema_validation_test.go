package twdesk_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

// TestAllToolsJSONSchemaValidation tests that all twdesk tools generate valid JSON schemas
func TestAllToolsJSONSchemaValidation(t *testing.T) {
	suite := testutil.NewSchemaValidationTestSuite()
	suite.RunAllSchemaValidationTests(t)
}

// TestToolInputSchemasArrayItems guards against array-typed schema nodes that
// omit `items`. OpenAI's Responses API rejects such tools at registration time
// (Anthropic does not), so a bare `{type: "array"}` branch inside anyOf/oneOf
// silently breaks downstream consumers that use OpenAI. This mirrors the same
// guard in internal/twprojects/tools_test.go.
func TestToolInputSchemasArrayItems(t *testing.T) {
	group := twdesk.DefaultToolsetGroup(false, &http.Client{})
	for method, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			name := tool.Tool.Name
			schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				t.Errorf("toolset %s tool %s: InputSchema is not *jsonschema.Schema (got %T)",
					method, name, tool.Tool.InputSchema)
				continue
			}
			for _, path := range arrayNodesMissingItems(schema, "InputSchema") {
				t.Errorf("toolset %s tool %s: array schema missing items at %s", method, name, path)
			}
		}
	}
}

// arrayNodesMissingItems walks a schema recursively and returns JSON-pointer-style
// paths of any node typed "array" that has no items schema.
func arrayNodesMissingItems(s *jsonschema.Schema, path string) []string {
	if s == nil {
		return nil
	}
	var issues []string
	if isArrayType(s) && s.Items == nil && len(s.ItemsArray) == 0 && len(s.PrefixItems) == 0 {
		issues = append(issues, path)
	}
	for name, sub := range s.Properties {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/properties/%s", path, name))...)
	}
	for name, sub := range s.PatternProperties {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/patternProperties/%s", path, name))...)
	}
	issues = append(issues, arrayNodesMissingItems(s.AdditionalProperties, path+"/additionalProperties")...)
	issues = append(issues, arrayNodesMissingItems(s.Items, path+"/items")...)
	for i, sub := range s.PrefixItems {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/prefixItems/%d", path, i))...)
	}
	for i, sub := range s.ItemsArray {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/items/%d", path, i))...)
	}
	for i, sub := range s.AnyOf {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/anyOf/%d", path, i))...)
	}
	for i, sub := range s.OneOf {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/oneOf/%d", path, i))...)
	}
	for i, sub := range s.AllOf {
		issues = append(issues, arrayNodesMissingItems(sub, fmt.Sprintf("%s/allOf/%d", path, i))...)
	}
	issues = append(issues, arrayNodesMissingItems(s.Not, path+"/not")...)
	issues = append(issues, arrayNodesMissingItems(s.If, path+"/if")...)
	issues = append(issues, arrayNodesMissingItems(s.Then, path+"/then")...)
	issues = append(issues, arrayNodesMissingItems(s.Else, path+"/else")...)
	issues = append(issues, arrayNodesMissingItems(s.Contains, path+"/contains")...)
	return issues
}

func isArrayType(s *jsonschema.Schema) bool {
	if s.Type == "array" {
		return true
	}
	for _, t := range s.Types {
		if t == "array" {
			return true
		}
	}
	return false
}

// TestToolInputSchemasOpenAIStrictMode documents the current state of OpenAI
// strict mode compatibility. Strict mode requires:
//   - All properties (including optional ones) listed in `required`
//   - `additionalProperties: false` on every object schema
//
// Optional parameters currently use `anyOf: [{type: T}, {type: null}]` but are
// NOT listed in `required`, which means they are incompatible with strict mode.
// This is a pre-existing pattern across ALL optional parameters in every twdesk
// tool — it applies equally to `inboxIDs`, `statusIDs`, `fields`, etc. and is
// NOT something introduced by sparse-fields support.
//
// If strict mode support is needed in the future the fix is:
//  1. Add every optional property to the `required` slice (it already allows
//     null, satisfying the "typed nullable" requirement).
//  2. Set `AdditionalProperties: boolSchema(false)` on every object schema.
//
// This test intentionally passes today as a documentation anchor; adjust it
// when strict mode support is implemented.
func TestToolInputSchemasOpenAIStrictMode(t *testing.T) {
	group := twdesk.DefaultToolsetGroup(false, &http.Client{})

	toolsWithOptionalParams := 0
	for _, ts := range group.Toolsets {
		for _, tool := range ts.GetAvailableTools() {
			schema, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				continue
			}
			requiredSet := make(map[string]bool, len(schema.Required))
			for _, r := range schema.Required {
				requiredSet[r] = true
			}
			for propName := range schema.Properties {
				if !requiredSet[propName] {
					toolsWithOptionalParams++
					break
				}
			}
		}
	}

	// All tools with optional params are currently not strict-mode compatible.
	// Update this expectation (to 0) once strict mode support is implemented.
	if toolsWithOptionalParams == 0 {
		t.Log("All tools appear strict-mode compatible — remove this test and the comment above")
	} else {
		t.Logf("%d tool(s) have optional properties not in 'required' (pre-existing; not strict-mode compatible)",
			toolsWithOptionalParams)
	}
}
