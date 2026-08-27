package twprojects

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/teamwork/twapi-go-sdk/projects"
)

// TestVocabularyPublishesWhatItAccepts is the point of the type: the enum a
// client reads off the schema and the values the handler accepts come off one
// slice, so a tool cannot advertise a value its handler rejects. Every published
// value is driven through the binder here, and the enum is compared against the
// same source.
func TestVocabularyPublishesWhatItAccepts(t *testing.T) {
	tests := []struct {
		name  string
		enum  []any
		bind  func(name string) error
		wants []any
	}{{
		name: "project statuses",
		enum: enumOf(t, projectStatusVocabulary.arraySchema("")),
		bind: func(name string) error {
			var target []projects.ProjectListStatus
			return projectStatusVocabulary.listParam(&target, "project_statuses")(
				map[string]any{"project_statuses": []any{name}},
			)
		},
		wants: []any{"active", "current", "late", "upcoming", "completed", "deleted"},
	}, {
		name: "project healths",
		enum: enumOf(t, projectHealthVocabulary.arraySchema("")),
		bind: func(name string) error {
			var target []projects.ProjectHealth
			return projectHealthVocabulary.listParam(&target, "project_healths")(
				map[string]any{"project_healths": []any{name}},
			)
		},
		wants: []any{"good", "ok", "bad", "not_set"},
	}, {
		name: "search types",
		enum: enumOf(t, searchTypeVocabulary.arraySchema("")),
		bind: func(name string) error {
			var target []projects.SearchRequestType
			return searchTypeVocabulary.listParam(&target, "types")(
				map[string]any{"types": []any{name}},
			)
		},
		wants: []any{
			"projects", "tasks", "tasklists", "milestones", "messages", "notebooks", "links",
			"comments", "taskcomments", "milestonecomments", "filecomments", "linkcomments",
			"notebookcomments", "timelogs", "users", "teams", "companies",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.enum) != len(tt.wants) {
				t.Fatalf("expected %d published values, got %v", len(tt.wants), tt.enum)
			}
			for i, want := range tt.wants {
				if tt.enum[i] != want {
					t.Errorf("published value %d is %v, expected %v", i, tt.enum[i], want)
				}
			}
			for _, value := range tt.enum {
				name, ok := value.(string)
				if !ok {
					t.Fatalf("published value %v is a %T, expected a string", value, value)
				}
				if err := tt.bind(name); err != nil {
					t.Errorf("published value %q is rejected by the handler: %v", name, err)
				}
			}
			if err := tt.bind("nonsense"); err == nil {
				t.Error("expected an unpublished value to be rejected by the handler")
			}
		})
	}
}

// TestVocabularyMapsNamesToAPIValues pins the translation a named vocabulary
// exists for. Project health is an integer on the wire whose meaning is
// positional, and "not_set" is zero — a mapping that dropped it, or an SDK query
// helper that skipped zero values, would silently widen the filter instead of
// narrowing it to the unrated projects.
func TestVocabularyMapsNamesToAPIValues(t *testing.T) {
	var healths []projects.ProjectHealth
	err := projectHealthVocabulary.listParam(&healths, "project_healths")(map[string]any{
		"project_healths": []any{"good", "not_set", "bad"},
	})
	if err != nil {
		t.Fatalf("failed to bind project_healths: %v", err)
	}
	want := []projects.ProjectHealth{
		projects.ProjectHealthGood,
		projects.ProjectHealthNotSet,
		projects.ProjectHealthBad,
	}
	if len(healths) != len(want) {
		t.Fatalf("expected %d healths, got %v", len(want), healths)
	}
	for i := range want {
		if healths[i] != want[i] {
			t.Errorf("health %d is %d, expected %d", i, healths[i], want[i])
		}
	}
}

// TestVocabularyOmittedLeavesTheTargetUnset guards the default: an unset filter
// must leave the request alone, so the endpoint's own behaviour stands.
func TestVocabularyOmittedLeavesTheTargetUnset(t *testing.T) {
	statuses := []projects.ProjectListStatus{projects.ProjectListStatusActive}
	bind := projectStatusVocabulary.listParam(&statuses, "project_statuses")

	for _, params := range []map[string]any{
		{},
		{"project_statuses": nil},
		{"project_statuses": []any{}},
	} {
		if err := bind(params); err != nil {
			t.Fatalf("failed to bind %v: %v", params, err)
		}
		if len(statuses) != 1 || statuses[0] != projects.ProjectListStatusActive {
			t.Errorf("binding %v overwrote the target, got %v", params, statuses)
		}
	}
}

// TestNewNamedVocabularyRejectsMismatchedLengths pins the panic: names and
// values are matched by position, so a vocabulary declared with a name missing
// its value would publish a value that maps to whatever sits at that index.
func TestNewNamedVocabularyRejectsMismatchedLengths(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected a panic when names and values differ in length")
		}
	}()
	newNamedVocabulary([]string{"one", "two"}, []projects.ProjectHealth{projects.ProjectHealthGood})
}

// enumOf reads the enum back out of the array schema a vocabulary publishes.
func enumOf(t *testing.T, schema *jsonschema.Schema) []any {
	t.Helper()

	for _, branch := range schema.AnyOf {
		if branch.Type == "array" && branch.Items != nil {
			return branch.Items.Enum
		}
	}
	t.Fatalf("no array branch with an enum in %+v", schema)
	return nil
}
