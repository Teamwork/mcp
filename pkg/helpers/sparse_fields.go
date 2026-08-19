package helpers

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// OptionalFieldsParam retrieves an optional sparse-fieldset parameter from a
// map and writes it into a `Filters.Fields.<Entity>` slot of a v3 list request.
// E is the SDK entity struct the slot selects attributes of (for example
// projects.Task for Filters.Fields.Tasks); F is inferred from target:
//
//	helpers.OptionalFieldsParam[projects.Task](&taskListRequest.Filters.Fields.Tasks, "fields")
//
// Values are validated against the attributes E actually marshals to, and an
// unknown one fails the call with the valid names listed. The API ignores
// attributes it does not recognise, so without this a typo would come back as a
// response quietly missing a field the caller asked for — indistinguishable
// from the field being empty.
//
// The entity id is always appended: it is what makes a row addressable by a
// follow-up get_* call, and WebLinker needs it to attach a web link.
//
// An absent or empty value leaves target untouched, so callers can tell an
// explicit selection from none and keep their verbose default.
func OptionalFieldsParam[E any, F ~string](target *[]F, key string) ParamFunc {
	return func(params map[string]any) error {
		if target == nil {
			return fmt.Errorf("target cannot be nil")
		}
		value, ok := params[key]
		if !ok || value == nil {
			return nil
		}
		array, ok := value.([]any)
		if !ok {
			return fmt.Errorf("invalid type for %s: expected []any, got %T", key, value)
		}
		if len(array) == 0 {
			return nil
		}

		allowed := SparseFieldNames[F, E]()
		selected := make([]F, 0, len(array)+1)
		for _, item := range array {
			name, ok := item.(string)
			if !ok {
				return fmt.Errorf("invalid type in %s: expected string, got %T", key, item)
			}
			field := F(name)
			if !slices.Contains(allowed, field) {
				return fmt.Errorf("unknown value %q in %s, must be one of %s", name, key, joinFieldNames(allowed))
			}
			if !slices.Contains(selected, field) {
				selected = append(selected, field)
			}
		}
		if id := F("id"); slices.Contains(allowed, id) && !slices.Contains(selected, id) {
			selected = append(selected, id)
		}
		*target = selected
		return nil
	}
}

// SparseFieldNames returns every attribute name the v3 sparse-fieldsets API
// accepts for the SDK entity struct E, typed as the entity's field alias F:
//
//	helpers.SparseFieldNames[projects.TaskField, projects.Task]()
//
// The names are the entity's JSON attribute names, which is what both the SDK's
// generated `<Entity>Field` constants and the tool's generated output schema are
// built from — so the set is exactly the one a caller reads off that schema.
// Deriving them here rather than restating the SDK constants keeps the two in
// step across SDK upgrades that add or rename attributes.
func SparseFieldNames[F ~string, E any]() []F {
	names := jsonAttributeNames(reflect.TypeFor[E]())
	fields := make([]F, len(names))
	for i, name := range names {
		fields[i] = F(name)
	}
	return fields
}

// jsonAttributeNames returns the JSON attribute names a struct type marshals
// to, in declaration order. It mirrors the SDK's sparsefieldsgen: only tagged
// exported fields count, and untagged embedded structs are flattened without
// shadowing a name the outer struct already contributes.
func jsonAttributeNames(t reflect.Type) []string {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	return collectJSONAttributeNames(t, map[reflect.Type]bool{t: true})
}

func collectJSONAttributeNames(t reflect.Type, visited map[reflect.Type]bool) []string {
	var names []string
	seen := make(map[string]bool)
	add := func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		names = append(names, name)
	}

	// First pass: the struct's own fields, so they shadow anything an embedded
	// type contributes.
	for field := range t.Fields() {
		if field.Anonymous || field.PkgPath != "" {
			continue
		}
		if name := jsonAttributeName(field); name != "" {
			add(name)
		}
	}

	// Second pass: flatten embedded structs. A tag on an embedded field means
	// it is marshalled as a nested object rather than promoted, so its own
	// attributes are not selectable.
	for field := range t.Fields() {
		if !field.Anonymous || field.Tag != "" {
			continue
		}
		embedded := field.Type
		for embedded.Kind() == reflect.Pointer {
			embedded = embedded.Elem()
		}
		if embedded.Kind() != reflect.Struct || visited[embedded] {
			continue
		}
		visited[embedded] = true
		for _, name := range collectJSONAttributeNames(embedded, visited) {
			add(name)
		}
	}

	return names
}

// jsonAttributeName returns the JSON attribute name field marshals to, or an
// empty string when it is not part of the JSON representation.
func jsonAttributeName(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "-" {
		return ""
	}
	return name
}

// joinFieldNames renders field names for an error message.
func joinFieldNames[F ~string](fields []F) string {
	names := make([]string, len(fields))
	for i, field := range fields {
		names[i] = string(field)
	}
	return strings.Join(names, ", ")
}
