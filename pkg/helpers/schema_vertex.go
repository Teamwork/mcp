package helpers

import (
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// VertexDocumentedFormats are the `format` values the Vertex Schema message
// documents (google/cloud/aiplatform/v1/openapi.proto). cmd/mcp-vertex-check
// reads the same list, so the checker and the rewriter cannot drift.
var VertexDocumentedFormats = []string{
	"float", "double", "int32", "int64", "enum", "date-time", "email", "byte",
}

// VertexMaxFunctionDeclarations is the documented ceiling on how many function
// declarations one Gemini *model request* may carry.
//
// VertexTools deliberately does NOT truncate to it. Gemini Enterprise imports a
// connector's actions and then enables them individually — hence Google's own
// advice to keep enabled actions under 100 — so the ceiling applies to what the
// client eventually sends the model, not to loading the tool list. Truncating
// here would hide tools for no gain, and unevenly: the preference order in
// internal/cli names 20 tools and the rest fall back alphabetically, so a cut
// at 128 removes every twspaces tool.
//
// cmd/mcp-vertex-check reports the count as a warning for the same reason.
const VertexMaxFunctionDeclarations = 128

// VertexTools rewrites tools for a client backed by Vertex AI, which parses a
// tool schema as a protobuf message rather than as JSON Schema. Tools and
// schemas are copies, like StrictTools.
//
// Two changes, both of which cost the client nothing:
//
//   - outputSchema is dropped. Ours declare `type: ["string","null"]` on 63
//     tools because the responses really do carry null, so the type union
//     cannot be removed the way the input ones were — and Vertex models `type`
//     as a single field, so a list cannot be parsed at all. Gemini validates
//     outputSchema, so one such node loses the whole tool list; without it the
//     tools load and structuredContent is simply not validated.
//   - inputSchema keywords Vertex has no field for are stripped (`examples`),
//     along with `format` values outside VertexDocumentedFormats.
//
// The tool count is left alone — see VertexMaxFunctionDeclarations.
//
// Run cmd/mcp-vertex-check after changing this.
func VertexTools(tools []*mcp.Tool) []*mcp.Tool {
	safe := make([]*mcp.Tool, len(tools))
	for i, tool := range tools {
		clone := *tool
		clone.OutputSchema = nil
		if schema, ok := tool.InputSchema.(*jsonschema.Schema); ok && schema != nil {
			clone.InputSchema = VertexSchema(schema)
		}
		safe[i] = &clone
	}
	return safe
}

// VertexSchema returns a copy of an input schema with the keywords Vertex
// cannot parse removed. The input is never modified.
func VertexSchema(schema *jsonschema.Schema) *jsonschema.Schema {
	if schema == nil {
		return nil
	}
	safe := schema.CloneSchemas()
	walkSchema(safe, makeVertexSafe)
	return safe
}

// makeVertexSafe strips one schema node's unparseable keywords.
func makeVertexSafe(s *jsonschema.Schema) {
	// Vertex has singular `example`, not the JSON Schema 2020-12 plural, so
	// `examples` is an unknown field.
	s.Examples = nil

	// `format` is a free string there, but an undocumented value buys nothing
	// and only risks a strict parse.
	if s.Format != "" && !slices.Contains(VertexDocumentedFormats, s.Format) {
		s.Format = ""
	}
}
