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
