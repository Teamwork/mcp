package helpers

import (
	"reflect"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
)

// DropNullBranches turns `anyOf: [{"type":"integer"}, {"type":"null"}]` into
// `{"type":"integer"}`, so optionality is absence from `required`.
//
// Vertex AI (Gemini Enterprise) has a single-valued type with no null member
// and rejects the whole tool list over one such branch. Clients that send null
// for unset parameters still work: dropNullArguments strips it before
// validation.
//
// Input schemas only — an output schema's null is a real wire value.
// Mutates in place and returns for chaining.
func DropNullBranches(schema *jsonschema.Schema) *jsonschema.Schema {
	dropNullBranches(schema)
	return schema
}

// dropNullBranches rewrites s and everything below it, reporting whether s
// stopped accepting null. The local rewrite runs first so a folded branch's own
// sub-schemas are still visited.
func dropNullBranches(s *jsonschema.Schema) bool {
	if s == nil {
		return false
	}
	dropped := dropNullBranch(s)
	for name, property := range s.Properties {
		if !dropNullBranches(property) {
			continue
		}
		// Null was this property's only way to say "not provided" (the Desk
		// shape, where everything is required). Keeping it required would now
		// make it mandatory.
		s.Required = slices.DeleteFunc(s.Required, func(required string) bool {
			return required == name
		})
	}
	dropNullBranches(s.Items)
	dropNullBranches(s.AdditionalProperties)
	for _, branches := range [][]*jsonschema.Schema{s.AnyOf, s.OneOf, s.AllOf} {
		for _, branch := range branches {
			dropNullBranches(branch)
		}
	}
	return dropped
}

// dropNullBranch removes bare null branches from one schema's composition
// keywords and folds a lone survivor into the schema itself. It reports whether
// anything was removed.
func dropNullBranch(s *jsonschema.Schema) bool {
	var dropped bool
	if branches, ok := withoutBareNulls(s.AnyOf); ok {
		s.AnyOf, dropped = branches, true
		if len(branches) == 1 {
			collapseInto(s, branches[0], "AnyOf")
		}
	}
	if branches, ok := withoutBareNulls(s.OneOf); ok {
		s.OneOf, dropped = branches, true
		if len(branches) == 1 {
			collapseInto(s, branches[0], "OneOf")
		}
	}
	// A union spelled as a type list. No input schema uses it today; handled so
	// the form cannot drift back in unnoticed.
	if len(s.Types) > 1 && slices.Contains(s.Types, "null") {
		types := slices.DeleteFunc(slices.Clone(s.Types), func(t string) bool {
			return t == "null"
		})
		if len(types) == 1 {
			s.Type, s.Types = types[0], nil
		} else {
			s.Types = types
		}
		dropped = true
	}
	return dropped
}

// withoutBareNulls drops every bare null branch, reporting whether it changed
// anything. Returns false if no branch would survive: an empty anyOf is invalid,
// and widening the parameter to "anything" is worse than leaving the null.
func withoutBareNulls(branches []*jsonschema.Schema) ([]*jsonschema.Schema, bool) {
	kept := slices.DeleteFunc(slices.Clone(branches), isBareNull)
	if len(kept) == len(branches) || len(kept) == 0 {
		return branches, false
	}
	return kept, true
}

// isBareNull reports whether s declares the null type and nothing else. A
// branch with other keywords constrains something, so it is left alone.
func isBareNull(s *jsonschema.Schema) bool {
	if s == nil || s.Type != "null" {
		return false
	}
	v := reflect.ValueOf(s).Elem()
	t := v.Type()
	for i := range t.NumField() {
		if field := t.Field(i); field.IsExported() && field.Name != "Type" && !v.Field(i).IsZero() {
			return false
		}
	}
	return true
}

// collapseInto folds branch into parent and clears parent's composition field
// (named by skip). It only fills fields parent leaves unset; on a conflict it
// changes nothing and returns false, leaving a one-branch anyOf rather than
// losing a value.
func collapseInto(parent, branch *jsonschema.Schema, skip string) bool {
	parentValue, branchValue := reflect.ValueOf(parent).Elem(), reflect.ValueOf(branch).Elem()
	fields := parentValue.Type()

	copyable := func(i int) bool {
		field := fields.Field(i)
		return field.IsExported() && field.Name != skip && !branchValue.Field(i).IsZero()
	}
	for i := range fields.NumField() {
		if copyable(i) && !parentValue.Field(i).IsZero() {
			return false
		}
	}
	for i := range fields.NumField() {
		if copyable(i) {
			parentValue.Field(i).Set(branchValue.Field(i))
		}
	}
	composition := parentValue.FieldByName(skip)
	composition.Set(reflect.Zero(composition.Type()))
	return true
}
