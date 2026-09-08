package helpers

import (
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StrictTools rewrites every input schema for OpenAI strict mode. Tools and
// schemas are copies: tools/list is handed the registered *mcp.Tool pointers,
// so rewriting in place would corrupt them for every later request.
//
// A schema that is not a *jsonschema.Schema passes through unchanged.
func StrictTools(tools []*mcp.Tool) []*mcp.Tool {
	strict := make([]*mcp.Tool, len(tools))
	for i, tool := range tools {
		schema, ok := tool.InputSchema.(*jsonschema.Schema)
		if !ok || schema == nil {
			strict[i] = tool
			continue
		}
		clone := *tool
		clone.InputSchema = StrictSchema(schema)
		strict[i] = &clone
	}
	return strict
}

// StrictSchema copies an input schema and rewrites it for OpenAI strict mode:
// every property in `required`, `additionalProperties: false` on every object,
// and optional parameters made nullable — the only way strict mode can say
// "not provided" once everything is required.
//
// The inverse of DropNullBranches. One document cannot be both, since strict
// mode needs the null that Vertex AI cannot parse, so this is derived per
// request. The input is never modified.
func StrictSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return nil
	}
	strict := schema.CloneSchemas()
	makeStrict(strict)
	return strict
}

// makeStrict applies the strict-mode rules to s and every schema below it.
func makeStrict(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	// Meaningless once every property is required, and strict mode rejects it.
	s.Default = nil

	if s.Type == "object" || len(s.Properties) > 0 {
		// Collect the optional ones before `required` covers everything. Sorted
		// so the output does not depend on map iteration order.
		optional := make([]string, 0, len(s.Properties))
		for name := range s.Properties {
			if !slices.Contains(s.Required, name) {
				optional = append(optional, name)
			}
		}
		slices.Sort(optional)
		for _, name := range optional {
			allowNull(s.Properties[name])
		}
		// CloneSchemas shares plain slices, so Required still points at the
		// registered schema's array. Appending or sorting in place would rewrite
		// what every other client is served.
		s.Required = append(slices.Clone(s.Required), optional...)
		slices.Sort(s.Required)
		s.AdditionalProperties = &jsonschema.Schema{Not: &jsonschema.Schema{}}
	}

	for _, property := range s.Properties {
		makeStrict(property)
	}
	makeStrict(s.Items)
	for _, branches := range [][]*jsonschema.Schema{s.AnyOf, s.OneOf, s.AllOf} {
		for _, branch := range branches {
			makeStrict(branch)
		}
	}
}

// allowNull widens a schema to accept null, which is how strict mode spells an
// optional parameter.
func allowNull(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	switch {
	case s.Type != "":
		// Type and Types must not both be set — marshalling rejects it.
		s.Types, s.Type = []string{s.Type, "null"}, ""
	case len(s.Types) > 0:
		if !slices.Contains(s.Types, "null") {
			s.Types = append(slices.Clone(s.Types), "null")
		}
	case len(s.AnyOf) > 0:
		s.AnyOf = append(s.AnyOf, &jsonschema.Schema{Type: "null"})
	case len(s.OneOf) > 0:
		s.OneOf = append(s.OneOf, &jsonschema.Schema{Type: "null"})
	default:
		// An untyped schema already accepts null.
	}

	// An enum constrains the value independently of the type, so null must join
	// it or the widened type is unreachable — the schema registers fine and
	// then rejects every call that omits the parameter.
	if s.Enum != nil && !slices.Contains(s.Enum, nil) {
		s.Enum = append(slices.Clone(s.Enum), nil)
	}
}
