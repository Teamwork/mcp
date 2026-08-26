package twprojects

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// indirectlyBoundParams maps a parameter no binder in the handler names to the
// helpers that read it out of the raw arguments map instead.
var indirectlyBoundParams = map[string][]string{
	"order_by":            {"param"},
	"order_mode":          {"param", "orderModeParam"},
	"notify":              {"parseNotify"},
	"attachment_refs":     {"parseAttachmentRefs", "parseTaskAttachments"},
	"attachment_file_ids": {"parseTaskAttachments"},
	"options":             {"parseCustomFieldOptions"},
	"field_values":        {"buildRecordFieldValues"},
}

// TestEveryToolParameterIsBound fails when a tool advertises a parameter its
// handler never reads. The MCP SDK validates the input schema but not the
// handler, so such a parameter is accepted and silently dropped: the caller is
// told the value was applied when it never reached the wire.
func TestEveryToolParameterIsBound(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("failed to list the package files: %v", err)
	}

	fset := token.NewFileSet()
	for _, path := range paths {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("failed to parse %s: %v", path, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			properties, handler := toolSchemaAndHandler(fn)
			if len(properties) == 0 || handler == nil {
				continue
			}
			literals, calls := handlerReads(handler)

			for _, property := range properties {
				if literals[property] {
					continue
				}
				var bound bool
				for _, binder := range indirectlyBoundParams[property] {
					if calls[binder] {
						bound = true
						break
					}
				}
				if !bound {
					t.Errorf("%s: %s advertises %q but its handler never reads it",
						path, fn.Name.Name, property)
				}
			}
		}
	}
}

// toolSchemaAndHandler returns the top-level input schema property names and the
// handler of the tool the function declares, if it declares one.
func toolSchemaAndHandler(fn *ast.FuncDecl) ([]string, *ast.FuncLit) {
	var properties []string
	var handler *ast.FuncLit

	ast.Inspect(fn, func(node ast.Node) bool {
		field, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok {
			return true
		}
		switch key.Name {
		case "InputSchema":
			properties = append(properties, schemaProperties(field.Value)...)
		case "Handler":
			if funcLit, ok := field.Value.(*ast.FuncLit); ok {
				handler = funcLit
			}
		}
		return true
	})
	return properties, handler
}

// schemaProperties returns the names of the schema's own properties, ignoring
// the ones nested inside them.
func schemaProperties(schema ast.Expr) []string {
	var properties []string
	ast.Inspect(schema, func(node ast.Node) bool {
		field, ok := node.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		if key, ok := field.Key.(*ast.Ident); !ok || key.Name != "Properties" {
			return true
		}
		composite, ok := field.Value.(*ast.CompositeLit)
		if !ok {
			return false
		}
		for _, element := range composite.Elts {
			entry, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if name, ok := stringLiteral(entry.Key); ok {
				properties = append(properties, name)
			}
		}
		return false
	})
	return properties
}

// handlerReads returns the string literals and the names of the functions the
// handler uses, which together cover both ways a parameter is bound.
func handlerReads(handler *ast.FuncLit) (map[string]bool, map[string]bool) {
	literals, calls := map[string]bool{}, map[string]bool{}
	ast.Inspect(handler, func(node ast.Node) bool {
		switch expr := node.(type) {
		case *ast.BasicLit:
			if value, ok := stringLiteral(expr); ok {
				literals[value] = true
			}
		case *ast.CallExpr:
			switch fn := expr.Fun.(type) {
			case *ast.Ident:
				calls[fn.Name] = true
			case *ast.SelectorExpr:
				calls[fn.Sel.Name] = true
			}
		}
		return true
	})
	return literals, calls
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	return strings.Trim(literal.Value, `"`), true
}
