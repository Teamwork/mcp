package twprojects

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/teamwork/mcp/pkg/helpers"
)

// vocabulary carries the accepted values of an enum-valued list filter, in the
// order they are published. It is the ordering type's counterpart for filtering
// rather than sorting, and exists for the same reason: the enum a client reads
// off the schema and the check the handler applies both come off one slice, so
// a tool cannot end up advertising a value its handler rejects. The enum is not
// optional — the SDK constants are Go identifiers the model never sees, so the
// schema is the only place the vocabulary is published, and without it a caller
// learns the values one rejection at a time.
//
// The published name of each value is kept separately from the value the SDK
// sends, because not every filter is a string on the wire. Project health is an
// integer whose meaning is positional — 1 is bad, 3 is good — and nothing in a
// tool definition would tell a model that, so the tool publishes names and maps
// them here.
type vocabulary[T comparable] struct {
	// names are the values as published in the schema enum.
	names []string

	// values are the values sent to the API, matched to names by position.
	values []T
}

// newVocabulary declares a vocabulary whose published names are the SDK values
// themselves. Take the values from the SDK constants, so a vocabulary the API
// widens arrives with an SDK bump rather than needing to be rediscovered here.
func newVocabulary[T ~string](values ...T) vocabulary[T] {
	v := vocabulary[T]{
		names:  make([]string, len(values)),
		values: values,
	}
	for i, value := range values {
		v.names[i] = string(value)
	}
	return v
}

// newNamedVocabulary declares a vocabulary whose published names differ from
// the values sent to the API, matched by position.
func newNamedVocabulary[T comparable](names []string, values []T) vocabulary[T] {
	if len(names) != len(values) {
		panic(fmt.Sprintf("vocabulary has %d names for %d values", len(names), len(values)))
	}
	return vocabulary[T]{names: names, values: values}
}

// arraySchema returns the schema of a filter accepting any number of the
// vocabulary's values.
func (v vocabulary[T]) arraySchema(description string) *jsonschema.Schema {
	enum := make([]any, len(v.names))
	for i, name := range v.names {
		enum[i] = name
	}
	return &jsonschema.Schema{
		Description: description,
		AnyOf: []*jsonschema.Schema{
			{Type: "array", Items: &jsonschema.Schema{Type: "string", Enum: enum}},
			{Type: "null"},
		},
	}
}

// listParam binds the named filter into target, translating each published name
// into the value the API expects. An unknown name is rejected here as well as by
// the schema, for the clients that skip validation.
func (v vocabulary[T]) listParam(target *[]T, key string) helpers.ParamFunc {
	return func(params map[string]any) error {
		var names []string
		if err := helpers.OptionalListParam(&names, key)(params); err != nil {
			return err
		}
		if len(names) == 0 {
			return nil
		}
		values := make([]T, 0, len(names))
		for _, name := range names {
			i := slices.Index(v.names, name)
			if i < 0 {
				return fmt.Errorf("value %q is not allowed for %s, must be one of %s",
					name, key, strings.Join(v.names, ", "))
			}
			values = append(values, v.values[i])
		}
		*target = values
		return nil
	}
}
