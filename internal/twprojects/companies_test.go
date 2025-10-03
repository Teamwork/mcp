package twprojects_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestCompanyCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"company":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyCreate.String(), map[string]any{
		"name":         "Example",
		"address_one":  "123 Example St",
		"address_two":  "Suite 456",
		"city":         "Example City",
		"state":        "EX",
		"zip":          "12345",
		"country_code": "US",
		"phone":        "123-456-7890",
		"fax":          "098-765-4321",
		"email_one":    "example1@test.com",
		"email_two":    "example2@test.com",
		"email_three":  "example3@test.com",
		"website":      "https://www.example.com",
		"profile":      "Example Company Profile",
		"manager_id":   float64(456),
		"industry_id":  float64(789),
		"tag_ids":      []float64{1, 2, 3},
	})
}

func TestCompanyUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyUpdate.String(), map[string]any{
		"id":           float64(123),
		"name":         "Example",
		"address_one":  "123 Example St",
		"address_two":  "Suite 456",
		"city":         "Example City",
		"state":        "EX",
		"zip":          "12345",
		"country_code": "US",
		"phone":        "123-456-7890",
		"fax":          "098-765-4321",
		"email_one":    "example1@test.com",
		"email_two":    "example2@test.com",
		"email_three":  "example3@test.com",
		"website":      "https://www.example.com",
		"profile":      "Example Company Profile",
		"manager_id":   float64(456),
		"industry_id":  float64(789),
		"tag_ids":      []float64{1, 2, 3},
	})
}

func TestCompanyDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCompanyGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestCompanyList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyList.String(), map[string]any{
		"search_term":    "test",
		"tag_ids":        []float64{1, 2, 3},
		"match_all_tags": true,
		"page":           float64(1),
		"page_size":      float64(10),
	})
}
