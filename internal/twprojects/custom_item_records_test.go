package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCustomItemRecordCreate(t *testing.T) {
	// Create dispatches a field-list GET (200) before the record POST (201);
	// route by URL so each leg sees the right status.
	mcpServer := testutil.ProjectsMCPServerRoutedMock(t, []testutil.ProjectsMockRoute{
		{Match: "/fields.json", Status: http.StatusOK, Body: []byte(`{"customItemFields":[]}`)},
		{Match: "/records.json", Status: http.StatusCreated, Body: []byte(`{"customItemRecord":{"id":123}}`)},
	}, http.StatusOK, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordCreate.String(), map[string]any{
		"custom_item_id":    float64(1001),
		"name":              "Acme Inc Contract",
		"section_id":        float64(789),
		"position_after_id": float64(456),
	})
}

func TestCustomItemRecordUpdate(t *testing.T) {
	// Update accepts 200 or 201 for the record PATCH, so a single 200 body
	// that merges the field-list and record payloads serves both calls.
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(
		`{"customItemFields":[],"customItemRecord":{"id":123}}`,
	))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordUpdate.String(), map[string]any{
		"custom_item_id":    float64(1002),
		"id":                float64(123),
		"name":              "Updated record",
		"clear_section":     true,
		"position_after_id": float64(456),
	})
}

func TestCustomItemRecordDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordDelete.String(), map[string]any{
		"custom_item_id": float64(1003),
		"id":             float64(123),
	})
}

func TestCustomItemRecordBulkDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordBulkDelete.String(), map[string]any{
		"custom_item_id": float64(1004),
		"ids":            []int64{10, 20, 30},
	})
}

func TestCustomItemRecordGet(t *testing.T) {
	// Get hits the record endpoint (200) and then the field-list endpoint
	// (200) for label translation; the merged body satisfies both decoders.
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(
		`{"customItemFields":[],"customItemRecord":{"id":123}}`,
	))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordGet.String(), map[string]any{
		"custom_item_id": float64(1005),
		"id":             float64(123),
	})
}

func TestCustomItemRecordList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(
		`{"customItemFields":[],"customItemRecords":[]}`,
	))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomItemRecordList.String(), map[string]any{
		"custom_item_id": float64(1006),
		"search_term":    "acme",
		"ids":            []int64{1, 2, 3},
		"section_ids":    []int64{10, 20},
		"show_deleted":   false,
		"order_by":       "name",
		"order_mode":     "asc",
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
