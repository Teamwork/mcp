package twprojects_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestCustomFieldValueCreateBatchReachesTheWire pins one value create per item, on the
// entity the batch names, covering one field across records and many fields on
// one record.
func TestCustomFieldValueCreateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"customfieldTask":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreateBatch.String(), map[string]any{
		"entity": "task",
		"values": []any{
			map[string]any{"entity_id": float64(1), "custom_field_id": float64(50), "value": "Phase 1"},
			map[string]any{"entity_id": float64(2), "custom_field_id": float64(50), "value": "Phase 2"},
			map[string]any{"entity_id": float64(2), "custom_field_id": float64(60), "value": true},
		},
	}, expectBatchResult(false, "Set 3 of 3 custom field values"))

	writes := requestsOfMethod(*recorded, http.MethodPost)
	if len(writes) != 3 {
		t.Fatalf("expected 3 writes, got %d", len(writes))
	}
	got := make([]string, 0, len(writes))
	for _, entry := range writes {
		got = append(got, entry.URL.Path+" "+string(entry.Body))
	}
	slices.Sort(got)
	for i, want := range []struct{ path, fragment string }{
		{"/projects/api/v3/tasks/1/customfields.json", `"customfieldId":50,"value":"Phase 1"`},
		{"/projects/api/v3/tasks/2/customfields.json", `"customfieldId":50,"value":"Phase 2"`},
		{"/projects/api/v3/tasks/2/customfields.json", `"customfieldId":60,"value":true`},
	} {
		if !strings.HasPrefix(got[i], want.path+" ") || !strings.Contains(got[i], want.fragment) {
			t.Errorf("write %d: expected %s with %s, got %s", i, want.path, want.fragment, got[i])
		}
	}
}

// TestCustomFieldValueCreateBatchLooksUpEachFieldOnce coerces numeric dropdown choices
// with one field lookup per field, not one per value.
func TestCustomFieldValueCreateBatchLooksUpEachFieldOnce(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/customfields/555", Status: http.StatusOK,
			Body: []byte(`{"customfield":{"id":555,"type":"dropdown"}}`)},
	}, http.StatusCreated, []byte(`{"customfieldTask":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreateBatch.String(), map[string]any{
		"entity": "task",
		"values": []any{
			map[string]any{"entity_id": float64(1), "custom_field_id": float64(555), "value": float64(1)},
			map[string]any{"entity_id": float64(2), "custom_field_id": float64(555), "value": float64(2)},
			map[string]any{"entity_id": float64(3), "custom_field_id": float64(555), "value": float64(3)},
		},
	}, expectBatchResult(false, "Set 3 of 3"))

	if reads := requestsOfMethod(*recorded, http.MethodGet); len(reads) != 1 {
		t.Errorf("expected 1 field lookup, got %d", len(reads))
	}
	for _, entry := range requestsOfMethod(*recorded, http.MethodPost) {
		if !strings.Contains(string(entry.Body), `"value":"`) {
			t.Errorf("expected a stringified choice, got %s", entry.Body)
		}
	}
}

// TestCustomFieldValueCreateBatchRejectsARepeatedTarget keeps two writes of one field on
// one record from racing.
func TestCustomFieldValueCreateBatchRejectsARepeatedTarget(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"customfieldTask":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreateBatch.String(), map[string]any{
		"entity": "project",
		"values": []any{
			map[string]any{"entity_id": float64(7), "custom_field_id": float64(50), "value": "a"},
			map[string]any{"entity_id": float64(7), "custom_field_id": float64(50), "value": "b"},
		},
	}, expectBatchResult(true, "Nothing was written", "item 2: field 50 on project 7 is listed more than once"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestCustomFieldValueCreateBatchReportsEachFailure keeps going past a failed value.
func TestCustomFieldValueCreateBatchReportsEachFailure(t *testing.T) {
	mcpServer, _ := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPost, Match: "/tasks/2/", Status: http.StatusForbidden, Body: []byte(`{}`)},
	}, http.StatusCreated, []byte(`{"customfieldTask":{"id":1}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreateBatch.String(), map[string]any{
		"entity": "task",
		"values": []any{
			map[string]any{"entity_id": float64(1), "custom_field_id": float64(50), "value": "x"},
			map[string]any{"entity_id": float64(2), "custom_field_id": float64(50), "value": "x"},
		},
	}, expectBatchResult(true, "Set 1 of 2", "Set: field 50 on task 1", "Failed: field 50 on task 2"))
}

// TestCustomFieldValueUpdateBatchSendsOneRequest pins a single bulk request per
// call, on each entity's route and wrapper key, carrying every value.
func TestCustomFieldValueUpdateBatchSendsOneRequest(t *testing.T) {
	tests := []struct {
		entity, path, key string
	}{
		{"task", "/projects/api/v3/tasks/7/customfields/bulk/update.json", "customfieldTasks"},
		{"project", "/projects/api/v3/projects/7/customfields/bulk/update.json", "customfieldProjects"},
		{"company", "/projects/api/v3/companies/7/customfields/bulk/update.json", "customfieldCompanies"},
	}
	for _, tt := range tests {
		t.Run(tt.entity, func(t *testing.T) {
			mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK,
				[]byte(`{"`+tt.key+`":[{"id":101},{"id":102}]}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueUpdateBatch.String(),
				map[string]any{
					"entity":    tt.entity,
					"entity_id": float64(7),
					"values": []any{
						map[string]any{"value_id": float64(101), "custom_field_id": float64(50), "value": "Phase 2"},
						map[string]any{"value_id": float64(102), "custom_field_id": float64(60), "clear": true},
					},
				}, expectBatchResult(false, "Updated 2 of 2 custom field values on "+tt.entity+" 7", "101, 102"))

			if len(*recorded) != 1 {
				t.Fatalf("expected exactly 1 request, got %d", len(*recorded))
			}
			sent := (*recorded)[0]
			if sent.Method != http.MethodPost || sent.URL.Path != tt.path {
				t.Errorf("expected POST %s, got %s %s", tt.path, sent.Method, sent.URL.Path)
			}
			want := `{"` + tt.key + `":[{"id":101,"customfieldId":50,"value":"Phase 2"},` +
				`{"id":102,"customfieldId":60,"value":null}]}`
			if got := strings.TrimSpace(string(sent.Body)); got != want {
				t.Errorf("expected body %s, got %s", want, got)
			}
		})
	}
}

// TestCustomFieldValueUpdateBatchReportsTheRejectedBatch says nothing was
// written when the all-or-nothing endpoint rejects the list.
func TestCustomFieldValueUpdateBatchReportsTheRejectedBatch(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected []string
	}{
		{"not found", http.StatusNotFound,
			[]string{"No values were updated", "all-or-nothing", "task 7", twprojects.MethodCustomFieldValueList.String()}},
		{"invalid value", http.StatusBadRequest,
			[]string{"No values were updated", "fits its field type"}},
		{"server error", http.StatusInternalServerError,
			[]string{"may not have been applied", twprojects.MethodCustomFieldValueList.String()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := mcpServerRecordingMock(t, nil, tt.status,
				[]byte(`{"errors":[{"title":"rejected"}]}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueUpdateBatch.String(),
				map[string]any{
					"entity":    "task",
					"entity_id": float64(7),
					"values": []any{
						map[string]any{"value_id": float64(101), "custom_field_id": float64(50), "value": "a"},
						map[string]any{"value_id": float64(999), "custom_field_id": float64(50), "value": "b"},
					},
				}, expectBatchResult(true, tt.expected...))

			if len(*recorded) != 1 {
				t.Errorf("expected exactly 1 request, got %d", len(*recorded))
			}
		})
	}
}

// TestCustomFieldValueUpdateBatchRejectsBeforeSending refuses a list the
// endpoint would reject, or could not be told apart, without sending anything.
func TestCustomFieldValueUpdateBatchRejectsBeforeSending(t *testing.T) {
	tooMany := make([]any, 101)
	for i := range tooMany {
		tooMany[i] = map[string]any{"value_id": float64(i + 1), "custom_field_id": float64(50), "value": "x"}
	}
	tests := []struct {
		name     string
		values   []any
		expected []string
	}{
		// The published maxItems refuses it before the handler, which repeats the
		// check for clients that skip schema validation.
		{"too long", tooMany, []string{"101", "100"}},
		{"repeated value", []any{
			map[string]any{"value_id": float64(101), "custom_field_id": float64(50), "value": "a"},
			map[string]any{"value_id": float64(101), "custom_field_id": float64(50), "value": "b"},
		}, []string{"No values were updated", "item 2: value 101 is already listed as item 1"}},
		{"value and clear", []any{
			map[string]any{"value_id": float64(101), "custom_field_id": float64(50), "value": "a", "clear": true},
		}, []string{"item 1: send either value or clear, not both"}},
		{"neither value nor clear", []any{
			map[string]any{"value_id": float64(101), "custom_field_id": float64(50)},
		}, []string{"item 1: value is required unless clear is true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueUpdateBatch.String(),
				map[string]any{
					"entity":    "project",
					"entity_id": float64(7),
					"values":    tt.values,
				}, expectBatchResult(true, tt.expected...))

			if len(*recorded) != 0 {
				t.Errorf("expected no requests, got %d", len(*recorded))
			}
		})
	}
}

// TestCustomFieldValueUpdateBatchCoercesNumericChoices stringifies a numeric
// dropdown choice with one field lookup per field.
func TestCustomFieldValueUpdateBatchCoercesNumericChoices(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/customfields/555", Status: http.StatusOK,
			Body: []byte(`{"customfield":{"id":555,"type":"dropdown"}}`)},
	}, http.StatusOK, []byte(`{"customfieldTasks":[{"id":1},{"id":2}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueUpdateBatch.String(), map[string]any{
		"entity":    "task",
		"entity_id": float64(7),
		"values": []any{
			map[string]any{"value_id": float64(1), "custom_field_id": float64(555), "value": float64(1)},
			map[string]any{"value_id": float64(2), "custom_field_id": float64(555), "value": float64(2)},
		},
	}, expectBatchResult(false, "Updated 2 of 2"))

	if reads := requestsOfMethod(*recorded, http.MethodGet); len(reads) != 1 {
		t.Errorf("expected 1 field lookup, got %d", len(reads))
	}
	writes := requestsOfMethod(*recorded, http.MethodPost)
	if len(writes) != 1 || !strings.Contains(string(writes[0].Body), `"value":"1"`) ||
		!strings.Contains(string(writes[0].Body), `"value":"2"`) {
		t.Errorf("expected one write with stringified choices, got %v", writes)
	}
}
