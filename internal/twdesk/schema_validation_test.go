package twdesk_test

import (
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
)

// TestAllToolsJSONSchemaValidation tests that all twdesk tools generate valid JSON schemas
func TestAllToolsJSONSchemaValidation(t *testing.T) {
	suite := testutil.NewSchemaValidationTestSuite()
	suite.RunAllSchemaValidationTests(t)
}
