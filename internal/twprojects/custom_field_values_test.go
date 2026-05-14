package twprojects_test

import (
	"net/http"
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
		"value":           []any{float64(10), float64(20), float64(30)},
	})
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
