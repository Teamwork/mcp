package twprojects_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// customItemRecordBatchFields is the field schema the batch tests resolve
// names against. Each test uses its own custom item ID, because the schema is
// cached per custom item.
var customItemRecordBatchFields = []byte(`{"customItemFields":[` +
	`{"id":1,"twId":"f_value","displayName":"Value","type":"number-integer","state":"active"}]}`)

// bulkCreateRequests returns the bulk create requests the tool sent.
func bulkCreateRequests(recorded []testutil.ProjectsRecordedRequest) []testutil.ProjectsRecordedRequest {
	var matched []testutil.ProjectsRecordedRequest
	for _, entry := range requestsOfMethod(recorded, http.MethodPost) {
		if strings.HasSuffix(entry.URL.Path, "/records/bulk.json") {
			matched = append(matched, entry)
		}
	}
	return matched
}

// TestCustomItemRecordCreateBatchReachesTheWire pins one bulk request for the
// whole list, with partial success on and each record's field values resolved
// from names to field TwIDs.
func TestCustomItemRecordCreateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/fields.json", Status: http.StatusOK, Body: customItemRecordBatchFields},
		{Method: http.MethodPost, Match: "/records/bulk.json", Status: http.StatusCreated,
			Body: []byte(`{"customItemRecords":[{"id":12345,"name":"Acme"},{"id":12346,"name":"Globex"}],"meta":{}}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordCreateBatch.String(), map[string]any{
		"custom_item_id": float64(2001),
		"records": []any{
			map[string]any{
				"name":         "Acme",
				"section_id":   float64(77),
				"field_values": []any{map[string]any{"field_name": "value", "value": "42"}},
			},
			map[string]any{"name": "Globex", "position_after_id": float64(12)},
		},
	}, expectBatchResult(false, "Created 2 of 2 custom item records",
		`item 1 ("Acme") → ID 12345`, `item 2 ("Globex") → ID 12346`))

	writes := bulkCreateRequests(*recorded)
	if len(writes) != 1 {
		t.Fatalf("expected one bulk request, got %d", len(writes))
	}
	if got := writes[0].URL.Path; got != "/projects/api/v3/customitems/2001/records/bulk.json" {
		t.Errorf("unexpected path %s", got)
	}
	if posts := requestsOfMethod(*recorded, http.MethodPost); len(posts) != 1 {
		t.Errorf("expected no other writes, got %d", len(posts))
	}

	var body struct {
		Records []map[string]any `json:"records"`
		Options map[string]any   `json:"options"`
	}
	if err := json.Unmarshal(writes[0].Body, &body); err != nil {
		t.Fatalf("request body is not JSON: %s", err)
	}
	if body.Options["partialSuccess"] != true {
		t.Errorf("expected partial success to be on, got %v", body.Options)
	}
	if len(body.Records) != 2 {
		t.Fatalf("expected 2 records on the wire, got %d", len(body.Records))
	}
	first, second := body.Records[0], body.Records[1]
	if first["name"] != "Acme" || first["sectionId"] != float64(77) {
		t.Errorf("unexpected first record %v", first)
	}
	if values, _ := first["fieldValues"].(map[string]any); values["f_value"] != float64(42) {
		t.Errorf("expected the field value keyed by TwID and coerced, got %v", first["fieldValues"])
	}
	if second["name"] != "Globex" || second["positionAfterId"] != float64(12) {
		t.Errorf("unexpected second record %v", second)
	}
	if _, ok := second["fieldValues"]; ok {
		t.Errorf("a record with no field values must not send any, got %v", second)
	}
}

// TestCustomItemRecordCreateBatchReportsEachFailure maps the created records
// back to their positions around the failed ones, and gives every failure its
// reason.
func TestCustomItemRecordCreateBatchReportsEachFailure(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/fields.json", Status: http.StatusOK, Body: customItemRecordBatchFields},
		{Method: http.MethodPost, Match: "/records/bulk.json", Status: http.StatusCreated,
			Body: []byte(`{"customItemRecords":[{"id":12345,"name":"Acme"},{"id":12347,"name":"Initech"}],` +
				`"failed":[{"index":1,"errors":[{"title":"invalid payload","detail":"section not found",` +
				`"meta":{"sectionId":999}}]}],"meta":{}}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordCreateBatch.String(), map[string]any{
		"custom_item_id": float64(2002),
		"records": []any{
			map[string]any{"name": "Acme"},
			map[string]any{"name": "Globex", "section_id": float64(999)},
			map[string]any{"name": "Initech"},
		},
	}, expectBatchResult(true, "Created 2 of 3 custom item records",
		`item 1 ("Acme") → ID 12345`, `item 3 ("Initech") → ID 12347`,
		`Failed: item 2 ("Globex"): invalid payload: section not found {"sectionId":999}`))

	if writes := bulkCreateRequests(*recorded); len(writes) != 1 {
		t.Errorf("expected one bulk request, got %d", len(writes))
	}
}

// TestCustomItemRecordCreateBatchReportsARejectedBatch reads the per-record
// errors out of the 400 the endpoint answers when no record could be created.
func TestCustomItemRecordCreateBatchReportsARejectedBatch(t *testing.T) {
	mcpServer, _ := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/fields.json", Status: http.StatusOK, Body: customItemRecordBatchFields},
		{Method: http.MethodPost, Match: "/records/bulk.json", Status: http.StatusBadRequest,
			Body: []byte(`{"errors":[` +
				`{"title":"invalid payload","detail":"section not found","meta":{"index":0}},` +
				`{"title":"forbidden action","detail":"maximum number of custom item records reached",` +
				`"meta":{"index":1}}]}`)},
	}, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordCreateBatch.String(), map[string]any{
		"custom_item_id": float64(2003),
		"records": []any{
			map[string]any{"name": "Acme", "section_id": float64(999)},
			map[string]any{"name": "Globex"},
		},
	}, expectBatchResult(true, "Created 0 of 2 custom item records. Nothing was written.",
		`Failed: item 1 ("Acme"): invalid payload: section not found.`,
		`Failed: item 2 ("Globex"): forbidden action: maximum number of custom item records reached.`))
}

// TestCustomItemRecordCreateBatchRejectsAnInvalidItem sends nothing when one
// record names a field the custom item does not have.
func TestCustomItemRecordCreateBatchRejectsAnInvalidItem(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodGet, Match: "/fields.json", Status: http.StatusOK, Body: customItemRecordBatchFields},
	}, http.StatusCreated, []byte(`{"customItemRecords":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordCreateBatch.String(), map[string]any{
		"custom_item_id": float64(2004),
		"records": []any{
			map[string]any{"name": "Acme"},
			map[string]any{"name": "Globex",
				"field_values": []any{map[string]any{"field_name": "Owner", "value": "x"}}},
		},
	}, expectBatchResult(true, "Nothing was written", `item 2 ("Globex")`, `unknown field "Owner"`))

	if posts := requestsOfMethod(*recorded, http.MethodPost); len(posts) != 0 {
		t.Errorf("expected nothing to be written, got %d writes", len(posts))
	}
}

// TestCustomItemRecordCreateBatchRejectsATooLongList refuses a list over the
// cap before sending anything.
func TestCustomItemRecordCreateBatchRejectsATooLongList(t *testing.T) {
	records := make([]any, 101)
	for i := range records {
		records[i] = map[string]any{"name": "Record"}
	}
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"customItemRecords":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordCreateBatch.String(), map[string]any{
		"custom_item_id": float64(2005),
		"records":        records,
	}, expectBatchResult(true))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests for an oversized list, got %d", len(*recorded))
	}
}
