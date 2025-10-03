//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestCompanyCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"company":{"id":123,"name":"Test Company","kind":"company"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCompanyCreate.String(), map[string]any{
		"id":          "123",
		"name":        "Test Company",
		"description": "A test company",
		"details":     "Company details",
		"industry":    "Technology",
		"website":     "https://example.com",
		"permission":  "own",
		"kind":        "company",
		"note":        "Test note",
		"domains":     []string{"example.com", "test.com"},
	})
}

func TestCompanyUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"company":{"id":123,"name":"Updated Company","kind":"company"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCompanyUpdate.String(), map[string]any{
		"id":          "123",
		"name":        "Updated Company",
		"description": "Updated description",
		"details":     "Updated details",
		"industry":    "Software",
		"website":     "https://updated.com",
		"permission":  "all",
		"kind":        "group",
		"note":        "Updated note",
		"domains":     []string{"updated.com"},
	})
}

func TestCompanyGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"company":{"id":123,"name":"Test Company","kind":"company"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCompanyGet.String(), map[string]any{
		"id": "123",
	})
}

func TestCompanyList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"companies":[{"id":123,"name":"Company 1","kind":"company"},{"id":124,"name":"Company 2","kind":"group"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCompanyList.String(), map[string]any{
		"name":      "Test Company",
		"domains":   []string{"example.com", "test.com"},
		"kind":      "company",
		"page":      float64(1),
		"page_size": float64(10),
	})
}

func TestCompanyListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"companies":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCompanyList.String(), map[string]any{})
}
