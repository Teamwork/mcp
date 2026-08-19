package helpers_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/teamwork/mcp/pkg/helpers"
)

type testField string

type testEmbedded struct {
	Embedded string `json:"embedded"`
	Shadowed string `json:"shadowed"`
}

type testTagged struct {
	Tagged string `json:"tagged"`
}

type testEntity struct {
	testEmbedded
	testTagged `json:"nested"`

	ID       int64  `json:"id"`
	Name     string `json:"name,omitempty"`
	Shadowed int64  `json:"shadowed"`
	Ignored  string `json:"-"`
	Untagged string
}

func TestSparseFieldNames(t *testing.T) {
	got := helpers.SparseFieldNames[testField, testEntity]()
	want := []testField{"id", "name", "shadowed", "embedded"}
	if !slices.Equal(got, want) {
		t.Errorf("expected %v but got %v", want, got)
	}
}

// TestSparseFieldNamesMatchMarshalledKeys ties the derived names to what the
// entity actually serialises as, since an attribute the API never returns is
// one a caller can select but never see. Two marshalled keys are deliberately
// not selectable, matching the SDK generator: an embedded struct carrying its
// own tag is a nested object rather than promoted attributes, and an untagged
// field is keyed by its Go name, which the v3 API does not recognise.
func TestSparseFieldNamesMatchMarshalledKeys(t *testing.T) {
	encoded, err := json.Marshal(testEntity{
		testEmbedded: testEmbedded{Embedded: "embedded", Shadowed: "shadowed"},
		testTagged:   testTagged{Tagged: "tagged"},
		ID:           1,
		Name:         "name",
		Shadowed:     2,
		Ignored:      "ignored",
		Untagged:     "untagged",
	})
	if err != nil {
		t.Fatalf("failed to marshal the entity: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("failed to decode the entity: %v", err)
	}

	notSelectable := []string{"nested", "Untagged"}
	for _, field := range helpers.SparseFieldNames[testField, testEntity]() {
		if _, ok := decoded[string(field)]; !ok {
			t.Errorf("%q is selectable but is not a marshalled key of the entity", field)
		}
	}
	for key := range decoded {
		selectable := slices.Contains(helpers.SparseFieldNames[testField, testEntity](), testField(key))
		if !selectable && !slices.Contains(notSelectable, key) {
			t.Errorf("%q is a marshalled key of the entity but is not selectable", key)
		}
	}
}

func TestSparseFieldNamesNonStruct(t *testing.T) {
	if got := helpers.SparseFieldNames[testField, string](); len(got) != 0 {
		t.Errorf("expected no field names for a non-struct entity but got %v", got)
	}
}

func TestOptionalFieldsParam(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]any
		want    []testField
		wantErr string
	}{{
		name:   "absent leaves the target untouched",
		params: map[string]any{},
	}, {
		name:   "null leaves the target untouched",
		params: map[string]any{"fields": nil},
	}, {
		name:   "empty selection leaves the target untouched",
		params: map[string]any{"fields": []any{}},
	}, {
		name:   "selection keeps the requested order",
		params: map[string]any{"fields": []any{"name", "embedded"}},
		want:   []testField{"name", "embedded", "id"},
	}, {
		name:   "id is not duplicated",
		params: map[string]any{"fields": []any{"id", "name"}},
		want:   []testField{"id", "name"},
	}, {
		name:   "duplicates are dropped",
		params: map[string]any{"fields": []any{"name", "name"}},
		want:   []testField{"name", "id"},
	}, {
		name:    "unknown attribute is rejected",
		params:  map[string]any{"fields": []any{"nope"}},
		wantErr: `unknown value "nope" in fields`,
	}, {
		name:    "an attribute the entity does not marshal is rejected",
		params:  map[string]any{"fields": []any{"Untagged"}},
		wantErr: `unknown value "Untagged" in fields`,
	}, {
		name:    "non-array is rejected",
		params:  map[string]any{"fields": "name"},
		wantErr: "invalid type for fields",
	}, {
		name:    "non-string item is rejected",
		params:  map[string]any{"fields": []any{float64(1)}},
		wantErr: "invalid type in fields",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []testField
			err := helpers.ParamGroup(test.params,
				helpers.OptionalFieldsParam[testEntity](&got, "fields"),
			)
			switch {
			case test.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case test.wantErr != "" && err == nil:
				t.Fatalf("expected an error containing %q but got none", test.wantErr)
			case test.wantErr != "":
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected an error containing %q but got %q", test.wantErr, err.Error())
				}
				return
			}
			if !slices.Equal(got, test.want) {
				t.Errorf("expected %v but got %v", test.want, got)
			}
		})
	}
}

// TestOptionalFieldsParamErrorListsValidValues guards the recovery path: an
// error that only says the value is wrong leaves a caller guessing, so it names
// the attributes it would have accepted.
func TestOptionalFieldsParamErrorListsValidValues(t *testing.T) {
	var got []testField
	err := helpers.ParamGroup(map[string]any{"fields": []any{"nope"}},
		helpers.OptionalFieldsParam[testEntity](&got, "fields"),
	)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"id", "name", "shadowed", "embedded"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected %q among the valid values in %q", want, err.Error())
		}
	}
}

func TestFieldsSchema(t *testing.T) {
	schema := helpers.FieldsSchema[testEntity]("thing")
	if !strings.Contains(schema.Description, "thing") {
		t.Errorf("expected the entity in the description but got %q", schema.Description)
	}
	// An LLM client never sees the output schema, so pointing at it is no help.
	if strings.Contains(schema.Description, "output schema") {
		t.Errorf("expected the description not to point at the output schema but got %q", schema.Description)
	}
	if len(schema.AnyOf) != 2 {
		t.Fatalf("expected a nullable array schema but got %d branches", len(schema.AnyOf))
	}
	// OpenAI's Responses API rejects an array node without items, so the branch
	// must declare them even though the tool is registered non-strict.
	items := schema.AnyOf[0].Items
	if items == nil {
		t.Fatal("expected the array branch to declare items")
	}
	// The enum must be the whole set OptionalFieldsParam accepts, not a subset.
	var got []testField
	for _, value := range items.Enum {
		name, ok := value.(string)
		if !ok {
			t.Fatalf("expected string enum values but got %T", value)
		}
		got = append(got, testField(name))
	}
	if want := helpers.SparseFieldNames[testField, testEntity](); !slices.Equal(got, want) {
		t.Errorf("expected the enum to be %v but got %v", want, got)
	}
}
