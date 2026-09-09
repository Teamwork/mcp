package main

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/teamwork/mcp/pkg/helpers"
)

// Severity separates what Vertex cannot parse from what it parses but has been
// reported to reject. Only the first kind is a hard failure.
type Severity string

const (
	// SeverityError contradicts the published Vertex Schema definition
	// (google/cloud/aiplatform/v1/openapi.proto), so it cannot be parsed.
	SeverityError Severity = "error"

	// SeverityWarn is allowed by that definition but reported rejected by
	// Gemini clients, or outside the documented value set.
	SeverityWarn Severity = "warn"
)

// Finding is one schema node Vertex would refuse.
type Finding struct {
	Tool     string   `json:"tool"`
	Schema   string   `json:"schema"` // inputSchema or outputSchema
	Path     string   `json:"path"`
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule"`
	Detail   string   `json:"detail"`
}

func (f Finding) String() string {
	return fmt.Sprintf("%s %s %s%s: %s (%s)", f.Severity, f.Tool, f.Schema, f.Path, f.Detail, f.Rule)
}

// vertexFields are the fields of the Vertex Schema message, by JSON name. $ref
// and $defs are accepted in both the proto spelling and the JSON Schema one.
var vertexFields = []string{
	"type", "format", "title", "description", "nullable", "default",
	"items", "minItems", "maxItems", "enum",
	"properties", "propertyOrdering", "required", "minProperties", "maxProperties",
	"minimum", "maximum", "minLength", "maxLength", "pattern",
	"example", "anyOf", "additionalProperties",
	"ref", "$ref", "defs", "$defs",
}

// vertexTypes are the members of the Vertex Type enum. There is no NULL — that
// is the whole reason this checker exists.
var vertexTypes = []string{"string", "number", "integer", "boolean", "array", "object"}

// documentedFormats are the format values the Schema message documents, shared
// with the rewriter so the two cannot drift. The proto says "email, byte, etc"
// for STRING, so the list is not exhaustive and an unlisted value is a warning
// rather than an error.
var documentedFormats = helpers.VertexDocumentedFormats

// keywords whose value is a schema, a map of schemas, or a list of schemas.
var (
	schemaMapKeywords = []string{
		"properties", "$defs", "defs", "patternProperties", "definitions", "dependentSchemas",
	}
	schemaKeywords = []string{
		"items", "additionalProperties", "not", "if", "then", "else", "contains",
		"propertyNames", "unevaluatedItems", "unevaluatedProperties", "contentSchema", "additionalItems",
	}
	schemaListKeywords = []string{"anyOf", "oneOf", "allOf", "prefixItems"}
)

// checker accumulates findings for one tool.
type checker struct {
	tool     string
	schema   string
	findings []Finding
}

func (c *checker) add(path string, severity Severity, rule, detail string) {
	c.findings = append(c.findings, Finding{
		Tool: c.tool, Schema: c.schema, Path: path,
		Severity: severity, Rule: rule, Detail: detail,
	})
}

// maxFunctionNameLength is the limit on FunctionDeclaration.name.
const maxFunctionNameLength = 64

// maxFunctionDeclarations is the documented ceiling on function declarations per
// model request, shared with the rewriter so the two cannot drift.
const maxFunctionDeclarations = helpers.VertexMaxFunctionDeclarations

// CheckSurface validates the tool list as a whole rather than a single tool.
//
// The count is a warning, not an error: the ceiling is on what a client sends
// the model, and Gemini Enterprise imports a connector's actions and enables
// them individually, so a longer list does not by itself stop the tools
// loading.
func CheckSurface(toolCount int) []Finding {
	if toolCount <= maxFunctionDeclarations {
		return nil
	}
	return []Finding{{
		Tool: "(surface)", Schema: "tools", Severity: SeverityWarn, Rule: "many-tools",
		Detail: fmt.Sprintf("%d tools; at most %d function declarations reach the model per request, "+
			"so a client must enable a subset", toolCount, maxFunctionDeclarations),
	}}
}

// functionNamePattern is what FunctionDeclaration.name accepts: it must start
// with a letter or underscore, then letters, digits, underscores, dots, dashes.
var functionNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]*$`)

// CheckTool validates a tool's name and its input and output schemas. The
// schemas must be the decoded JSON of what tools/list publishes — the wire form
// is what Vertex parses, and it differs from the Go structs that produced it.
func CheckTool(name string, inputSchema, outputSchema any) []Finding {
	var findings []Finding

	nameChecker := &checker{tool: name, schema: "name"}
	switch {
	case len(name) > maxFunctionNameLength:
		nameChecker.add("", SeverityError, "name-too-long",
			fmt.Sprintf("a function name is limited to %d characters, this is %d",
				maxFunctionNameLength, len(name)))
	case !functionNamePattern.MatchString(name):
		nameChecker.add("", SeverityError, "name-charset",
			"a function name must start with a letter or underscore, then only letters, digits, _ . -")
	}
	findings = append(findings, nameChecker.findings...)

	for _, s := range []struct {
		label  string
		schema any
	}{{"inputSchema", inputSchema}, {"outputSchema", outputSchema}} {
		if s.schema == nil {
			continue
		}
		c := &checker{tool: name, schema: s.label}
		c.walk(s.schema, "", true)
		findings = append(findings, c.findings...)
	}
	return findings
}

// walk validates one schema node and descends into the sub-schemas it holds.
func (c *checker) walk(node any, path string, root bool) {
	object, ok := node.(map[string]any)
	if !ok {
		// A boolean schema (true/false) is legal JSON Schema. Vertex models
		// schemas as a message, so only additionalProperties accepts one — the
		// caller checks that, and anything else is reported there.
		return
	}

	for _, key := range sortedKeys(object) {
		value := object[key]
		child := path + "/" + key

		if !slices.Contains(vertexFields, key) {
			c.add(child, SeverityError, "unknown-field",
				fmt.Sprintf("Vertex Schema has no %q field", key))
			continue
		}

		switch key {
		case "type":
			c.checkType(value, child)
		case "format":
			if text, isString := value.(string); isString && !slices.Contains(documentedFormats, text) {
				c.add(child, SeverityWarn, "undocumented-format",
					fmt.Sprintf("format %q is not in the documented set", text))
			}
		case "additionalProperties":
			c.checkAdditionalProperties(value, child)
		case "defs", "$defs":
			if !root {
				c.add(child, SeverityError, "defs-not-at-root", "$defs is only allowed at the schema root")
			}
		}

		switch {
		case slices.Contains(schemaMapKeywords, key):
			if m, isMap := value.(map[string]any); isMap {
				for _, sub := range sortedKeys(m) {
					c.walk(m[sub], child+"/"+sub, false)
				}
			}
		case slices.Contains(schemaListKeywords, key):
			if list, isList := value.([]any); isList {
				for i, sub := range list {
					c.walk(sub, fmt.Sprintf("%s/%d", child, i), false)
				}
			}
		case slices.Contains(schemaKeywords, key):
			c.walk(value, child, false)
		}
	}
}

// checkType is the rule that matters most: Vertex models type as a single
// enum value, so a list cannot be parsed at all and there is no NULL member.
func (c *checker) checkType(value any, path string) {
	switch typed := value.(type) {
	case string:
		if typed == "null" {
			c.add(path, SeverityError, "null-type", "the Vertex Type enum has no NULL member")
			return
		}
		if !slices.Contains(vertexTypes, typed) {
			c.add(path, SeverityError, "unknown-type", fmt.Sprintf("%q is not a Vertex Type member", typed))
		}
	case []any:
		// Reported by Gemini clients as "Proto field is not repeating, cannot
		// start list".
		names := make([]string, 0, len(typed))
		for _, entry := range typed {
			names = append(names, fmt.Sprint(entry))
		}
		c.add(path, SeverityError, "type-array",
			fmt.Sprintf("type is a single enum field, so the list [%s] cannot be parsed",
				strings.Join(names, ", ")))
	default:
		c.add(path, SeverityError, "unknown-type", fmt.Sprintf("type must be a string, got %T", value))
	}
}

// checkAdditionalProperties reports an object-valued additionalProperties. The
// Schema message takes a Value here so the proto permits it, but Gemini clients
// answer "Expected boolean, received object" (gemini-cli #13694, closed as not
// planned), hence a warning rather than an error.
func (c *checker) checkAdditionalProperties(value any, path string) {
	if _, isObject := value.(map[string]any); isObject {
		c.add(path, SeverityWarn, "additional-properties-object",
			"additionalProperties is a schema; Gemini clients expect a boolean")
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
