//nolint:lll
package twdesk_test

import (
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twdesk"
)

func TestCustomerCreate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusCreated, []byte(`{"customer":{"id":123,"firstName":"John","lastName":"Doe","email":"john@example.com"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCustomerCreate.String(), map[string]any{
		"id":            "123",
		"firstName":     "John",
		"lastName":      "Doe",
		"email":         "john@example.com",
		"organization":  "Test Corp",
		"extraData":     "Some extra data",
		"notes":         "Test customer notes",
		"linkedinURL":   "https://linkedin.com/in/johndoe",
		"facebookURL":   "https://facebook.com/johndoe",
		"twitterHandle": "@johndoe",
		"jobTitle":      "Software Engineer",
		"phone":         "+1234567890",
		"mobile":        "+0987654321",
		"address":       "123 Test St, Test City",
	})
}

func TestCustomerUpdate(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customer":{"id":123,"firstName":"Jane","lastName":"Smith","email":"jane@example.com"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCustomerUpdate.String(), map[string]any{
		"id":            "123",
		"firstName":     "Jane",
		"lastName":      "Smith",
		"email":         "jane@example.com",
		"organization":  "Updated Corp",
		"extraData":     "Updated extra data",
		"notes":         "Updated customer notes",
		"linkedinURL":   "https://linkedin.com/in/janesmith",
		"facebookURL":   "https://facebook.com/janesmith",
		"twitterHandle": "@janesmith",
		"jobTitle":      "Senior Engineer",
		"phone":         "+1111111111",
		"mobile":        "+2222222222",
		"address":       "456 Updated St, Updated City",
	})
}

func TestCustomerGet(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customer":{"id":123,"firstName":"John","lastName":"Doe","email":"john@example.com"}}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCustomerGet.String(), map[string]any{
		"id": "123",
	})
}

func TestCustomerList(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customers":[{"id":123,"firstName":"John","lastName":"Doe"},{"id":124,"firstName":"Jane","lastName":"Smith"}]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCustomerList.String(), map[string]any{
		"companyIDs":   []float64{1, 2, 3},
		"companyNames": []string{"Test Corp", "Example Inc"},
		"emails":       []string{"john@example.com", "jane@example.com"},
		"page":         float64(1),
		"pageSize":     float64(10),
	})
}

func TestCustomerListMinimal(t *testing.T) {
	mcpServer, cleanup := mcpServerMock(t, http.StatusOK, []byte(`{"customers":[]}`))
	defer cleanup()

	testutil.ExecuteToolRequest(t, mcpServer, twdesk.MethodCustomerList.String(), map[string]any{})
}
