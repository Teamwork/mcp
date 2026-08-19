package helpers_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	"github.com/teamwork/mcp/pkg/testutil"
	"github.com/teamwork/twapi-go-sdk/projects"
)

// TestNewCountToolResultReadsMetaPageCount covers the happy path: the count
// comes from meta.page.count, the rows are discarded.
func TestNewCountToolResultReadsMetaPageCount(t *testing.T) {
	engine := testutil.EngineMock(http.StatusOK, []byte(
		`{"tasks":[{"id":1,"name":"Ship it"}],"meta":{"page":{"count":42,"hasMore":true}}}`))

	result, err := helpers.NewCountToolResult(context.Background(), engine,
		projects.NewTaskListRequest(), "failed to count tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", toolResultText(t, result))
	}
	if got, want := toolResultText(t, result), `{"count":42}`; got != want {
		t.Errorf("expected %s but got %s", want, got)
	}
	if count, ok := helpers.CountFromToolResult(result); !ok || count != 42 {
		t.Errorf("expected the structured content to carry 42 but got %d (ok=%t)", count, ok)
	}
}

// TestNewCountToolResultRequiresACount guards the one failure that must never be
// answered with a number: a zero would read as "nothing matches".
func TestNewCountToolResultRequiresACount(t *testing.T) {
	engine := testutil.EngineMock(http.StatusOK, []byte(`{"tasks":[],"meta":{"page":{"hasMore":false}}}`))

	result, err := helpers.NewCountToolResult(context.Background(), engine,
		projects.NewTaskListRequest(), "failed to count tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected a tool error but got %s", toolResultText(t, result))
	}
	if text := toolResultText(t, result); !strings.Contains(text, "did not report a count") {
		t.Errorf("expected the error to say the count is missing but got %s", text)
	}
}

// TestNewCountToolResultZeroIsACount is the control for the test above: zero is
// an answer, not a missing count.
func TestNewCountToolResultZeroIsACount(t *testing.T) {
	engine := testutil.EngineMock(http.StatusOK, []byte(`{"tasks":[],"meta":{"page":{"count":0}}}`))

	result, err := helpers.NewCountToolResult(context.Background(), engine,
		projects.NewTaskListRequest(), "failed to count tasks")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected tool error: %s", toolResultText(t, result))
	}
	if got, want := toolResultText(t, result), `{"count":0}`; got != want {
		t.Errorf("expected %s but got %s", want, got)
	}
}

// TestNewCountToolResultAPIFailuresAreToolResults keeps the count path in line
// with the repo: a status comes back as an IsError result, not a Go error.
func TestNewCountToolResultAPIFailuresAreToolResults(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError} {
		engine := testutil.EngineMock(status, []byte(`{}`))

		result, err := helpers.NewCountToolResult(context.Background(), engine,
			projects.NewTaskListRequest(), "failed to count tasks")
		if err != nil {
			t.Fatalf("status %d: expected a tool result but got a Go error: %v", status, err)
		}
		if !result.IsError {
			t.Errorf("status %d: expected a tool error but got %s", status, toolResultText(t, result))
		}
	}
}

// requesterWithoutFilters stands in for a request that cannot carry a count. The
// rewiring addresses the filter slots by name, so a request without them has to
// be reported rather than sent as-is.
type requesterWithoutFilters struct{}

func (requesterWithoutFilters) HTTPRequest(ctx context.Context, server string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, server, nil)
}

func TestNewCountToolResultRejectsARequestItCannotCount(t *testing.T) {
	// The engine is never reached: the request is rejected before it is sent.
	result, err := helpers.NewCountToolResult(context.Background(), nil,
		requesterWithoutFilters{}, "failed to count things")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected a tool error but got %s", toolResultText(t, result))
	}
	if text := toolResultText(t, result); !strings.Contains(text, "no filters to count with") {
		t.Errorf("expected the error to name the missing filters but got %s", text)
	}
}

// TestCountFromToolResultRejectsOtherShapes pins the guard a wrapping tool
// relies on: anything else must fail rather than decode to zero.
func TestCountFromToolResultRejectsOtherShapes(t *testing.T) {
	for name, content := range map[string]any{
		"nil":            nil,
		"a list body":    map[string]any{"tasks": []any{}},
		"a wrong type":   map[string]any{"count": "many"},
		"an empty count": map[string]any{"count": nil},
	} {
		t.Run(name, func(t *testing.T) {
			result := &mcp.CallToolResult{StructuredContent: content}
			if count, ok := helpers.CountFromToolResult(result); ok {
				t.Errorf("expected no count but got %d", count)
			}
		})
	}
	if _, ok := helpers.CountFromToolResult(nil); ok {
		t.Error("expected no count from a nil result")
	}
}

// TestWithCountOnlySchemaDeclaresTheCount covers the published contract: a tool
// that can answer with a count says so in its output schema.
func TestWithCountOnlySchemaDeclaresTheCount(t *testing.T) {
	schema := helpers.WithCountOnlySchema(&jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{"tasks": {Type: "array", Items: &jsonschema.Schema{Type: "object"}}},
	})
	count, ok := schema.Properties["count"]
	if !ok {
		t.Fatal("expected the schema to declare a count property")
	}
	if count.Type != "integer" {
		t.Errorf("expected count to be an integer but got %q", count.Type)
	}
	if _, ok := schema.Properties["tasks"]; !ok {
		t.Error("expected the existing properties to survive")
	}
	if helpers.WithCountOnlySchema(nil) != nil {
		t.Error("expected a nil schema to stay nil")
	}
}
