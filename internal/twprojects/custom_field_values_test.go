package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCustomFieldValueCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"custom_field_id": float64(555),
		"value":           "in progress",
	})
}

func TestCustomFieldValueCreateProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfieldProject":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "project",
		"entity_id":       float64(888),
		"custom_field_id": float64(555),
		"value":           float64(42),
	})
}

func TestCustomFieldValueCreateMultiselect(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"custom_field_id": float64(555),
		"value":           []any{"10", "20", "30"},
	})
}

func TestCustomFieldValueCreateDropdownCoercesNumberToString(t *testing.T) {
	// A numeric value bound for a dropdown field must be stringified: the API
	// stores dropdown choices as strings and rejects a raw number. The create
	// handler resolves the field type via a GET (200) before the value POST
	// (201), so route by URL to give each leg the right status.
	mcpServer, body := testutil.ProjectsMCPServerRoutedMockWithRequestBody(t, []testutil.ProjectsMockRoute{
		{Match: "/customfields/555", Status: http.StatusOK, Body: []byte(`{"customfield":{"id":555,"type":"dropdown"}}`)},
	}, http.StatusCreated, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"custom_field_id": float64(555),
		"value":           float64(1),
	})
	if got := string(*body); !strings.Contains(got, `"value":"1"`) {
		t.Errorf("expected posted value to be stringified as \"1\", got body: %s", got)
	}
}

func TestCustomFieldValueCreateNumberFieldKeepsNumber(t *testing.T) {
	// A numeric value bound for a number field must stay a number, not be
	// stringified — only dropdown/multiselect choices are coerced.
	mcpServer, body := testutil.ProjectsMCPServerRoutedMockWithRequestBody(t, []testutil.ProjectsMockRoute{
		{Match: "/customfields/555", Status: http.StatusOK, Body: []byte(`{"customfield":{"id":555,"type":"number-integer"}}`)},
	}, http.StatusCreated, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"custom_field_id": float64(555),
		"value":           float64(42),
	})
	if got := string(*body); !strings.Contains(got, `"value":42`) {
		t.Errorf("expected posted value to remain the number 42, got body: %s", got)
	}
}

func TestCustomFieldValueCreateStatusCoercesNumberToString(t *testing.T) {
	// Status fields are choice-based strings too, so a numeric value must be
	// stringified just like a dropdown.
	mcpServer, body := testutil.ProjectsMCPServerRoutedMockWithRequestBody(t, []testutil.ProjectsMockRoute{
		{Match: "/customfields/555", Status: http.StatusOK, Body: []byte(`{"customfield":{"id":555,"type":"status"}}`)},
	}, http.StatusCreated, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"custom_field_id": float64(555),
		"value":           float64(5),
	})
	if got := string(*body); !strings.Contains(got, `"value":"5"`) {
		t.Errorf("expected posted value to be stringified as \"5\", got body: %s", got)
	}
}

func TestCustomFieldValueUpdateDropdownCoercesNumberToString(t *testing.T) {
	// Same coercion applies on update. The field-type GET (custom_field_id=555)
	// and the value PATCH (value_id=123) both live under /customfields/, so the
	// route matches the field id specifically and the PATCH falls through.
	mcpServer, body := testutil.ProjectsMCPServerRoutedMockWithRequestBody(t, []testutil.ProjectsMockRoute{
		{Match: "/customfields/555", Status: http.StatusOK, Body: []byte(`{"customfield":{"id":555,"type":"dropdown"}}`)},
	}, http.StatusOK, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueUpdate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"value_id":        float64(123),
		"custom_field_id": float64(555),
		"value":           float64(3),
	})
	if got := string(*body); !strings.Contains(got, `"value":"3"`) {
		t.Errorf("expected patched value to be stringified as \"3\", got body: %s", got)
	}
}

func TestCustomFieldValueCreateMultiselectCoercesNumbersToStrings(t *testing.T) {
	// Multiselect values arrive as an array; each numeric element must be
	// stringified to match the stored choice values.
	mcpServer, body := testutil.ProjectsMCPServerRoutedMockWithRequestBody(t, []testutil.ProjectsMockRoute{
		{Match: "/customfields/555", Status: http.StatusOK, Body: []byte(`{"customfield":{"id":555,"type":"multiselect"}}`)},
	}, http.StatusCreated, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueCreate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"custom_field_id": float64(555),
		"value":           []any{float64(10), float64(20)},
	})
	if got := string(*body); !strings.Contains(got, `"value":["10","20"]`) {
		t.Errorf("expected posted value to be stringified array, got body: %s", got)
	}
}

func TestCustomFieldValueUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueUpdate.String(), map[string]any{
		"entity":          "task",
		"entity_id":       float64(777),
		"value_id":        float64(123),
		"custom_field_id": float64(555),
		"value":           "done",
	})
}

func TestCustomFieldValueDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueDelete.String(), map[string]any{
		"entity":    "task",
		"entity_id": float64(777),
		"value_id":  float64(123),
	})
}

func TestCustomFieldValueGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfieldTask":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueGet.String(), map[string]any{
		"entity":    "task",
		"entity_id": float64(777),
		"value_id":  float64(123),
	})
}

func TestCustomFieldValueList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"customfieldTasks":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCustomFieldValueList.String(), map[string]any{
		"entity":           "task",
		"entity_id":        float64(777),
		"custom_field_ids": []int64{1, 2},
		"page":             float64(1),
		"page_size":        float64(10),
	})
}
