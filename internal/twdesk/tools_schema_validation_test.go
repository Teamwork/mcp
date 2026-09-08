package twdesk_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/pkg/helpers"
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
	return slices.Contains(s.Types, "array")
}

// TestToolInputSchemasOpenAIStrictMode verifies the two shapes a tool is served
// in, which are opposites and cannot be one document:
//
//   - published: no null type anywhere, or Vertex AI (Gemini Enterprise)
//     rejects the whole tool list;
//   - the strict variant derived for OpenAI: every property required,
//     additionalProperties false, optional properties nullable — enums
//     included, or the widened type is unreachable.
func TestToolInputSchemasOpenAIStrictMode(t *testing.T) {
	group := twdesk.DefaultToolsetGroup(false, &http.Client{})

	for method, ts := range group.Toolsets {
		for _, tool := range ts.GetAvailableTools() {
			name := tool.Tool.Name
			published, ok := tool.Tool.InputSchema.(*jsonschema.Schema)
			if !ok {
				t.Errorf("toolset %s tool %s: InputSchema is not *jsonschema.Schema", method, name)
				continue
			}
			for _, path := range nullTypeNodes(published, "InputSchema") {
				t.Errorf("toolset %s tool %s: published schema declares the null type at %s, "+
					"which Vertex AI cannot parse", method, name, path)
			}
			for _, issue := range strictModeIssues(helpers.StrictSchema(published), published, "InputSchema") {
				t.Errorf("toolset %s tool %s: strict variant %s", method, name, issue)
			}
		}
	}
}

// nullTypeNodes reports every node declaring the null type, in either form.
func nullTypeNodes(s *jsonschema.Schema, path string) []string {
	if s == nil {
		return nil
	}
	var found []string
	if s.Type == "null" || slices.Contains(s.Types, "null") {
		found = append(found, path)
	}
	for name, property := range s.Properties {
		found = append(found, nullTypeNodes(property, fmt.Sprintf("%s/properties/%s", path, name))...)
	}
	found = append(found, nullTypeNodes(s.Items, path+"/items")...)
	for i, branch := range s.AnyOf {
		found = append(found, nullTypeNodes(branch, fmt.Sprintf("%s/anyOf/%d", path, i))...)
	}
	return found
}

// strictModeIssues reports every strict-mode requirement a schema fails,
// comparing against the published one it came from.
func strictModeIssues(strict, published *jsonschema.Schema, path string) []string {
	if strict == nil {
		return nil
	}
	var issues []string
	if strict.Default != nil {
		issues = append(issues, path+" keeps a default, which strict mode does not accept")
	}
	if len(strict.Properties) == 0 {
		return append(issues, strictModeIssues(strict.Items, publishedItems(published), path+"/items")...)
	}
	if strict.AdditionalProperties == nil {
		issues = append(issues, path+" is missing additionalProperties: false")
	}
	for name, property := range strict.Properties {
		propertyPath := fmt.Sprintf("%s/properties/%s", path, name)
		if !slices.Contains(strict.Required, name) {
			issues = append(issues, propertyPath+" is not in required")
		}
		// Optional in the published schema, so it must accept null.
		if published != nil && !slices.Contains(published.Required, name) {
			if !acceptsNullType(property) {
				issues = append(issues, propertyPath+" is optional but cannot be null")
			}
			if property.Enum != nil && !slices.Contains(property.Enum, nil) {
				issues = append(issues, propertyPath+" is a nullable enum that does not permit null")
			}
		}
		issues = append(issues, strictModeIssues(property, publishedProperty(published, name), propertyPath)...)
	}
	return issues
}

// acceptsNullType reports whether s permits null. An untyped schema does.
func acceptsNullType(s *jsonschema.Schema) bool {
	if s == nil {
		return true
	}
	if s.Type == "null" || slices.Contains(s.Types, "null") {
		return true
	}
	for _, branches := range [][]*jsonschema.Schema{s.AnyOf, s.OneOf} {
		for _, branch := range branches {
			if acceptsNullType(branch) {
				return true
			}
		}
	}
	return s.Type == "" && len(s.Types) == 0 && len(s.AnyOf) == 0 && len(s.OneOf) == 0
}

func publishedProperty(s *jsonschema.Schema, name string) *jsonschema.Schema {
	if s == nil {
		return nil
	}
	return s.Properties[name]
}

func publishedItems(s *jsonschema.Schema) *jsonschema.Schema {
	if s == nil {
		return nil
	}
	return s.Items
}
