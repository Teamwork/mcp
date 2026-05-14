package twprojects_test

import (
	"fmt"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestToolInputSchemasArrayItems guards against array-typed schema nodes that
// omit `items`. OpenAI's Responses API rejects such tools at registration time
// (Anthropic does not), so a bare `{type: "array"}` branch silently breaks any
// downstream consumer that uses OpenAI. See PR fix for custom_field_values.
func TestToolInputSchemasArrayItems(t *testing.T) {
	group := twprojects.DefaultToolsetGroup(false, true, testutil.ProjectsEngineMock(200, nil))
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

// arrayNodesMissingItems walks a schema and returns the JSON-pointer-style
// paths of any node whose type is "array" but which has no items schema.
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
