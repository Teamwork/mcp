package twprojects_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestTeamCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamCreate.String(), map[string]any{
		"name":           "Example",
		"handle":         "example",
		"description":    "Example description",
		"parent_team_id": float64(123),
		"company_id":     float64(456),
		"project_id":     float64(789),
		"user_ids": []any{
			float64(1),
			float64(2),
			float64(3),
		},
	})
}

func TestTeamUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamUpdate.String(), map[string]any{
		"id":             float64(123),
		"name":           "Example",
		"handle":         "example",
		"description":    "Example description",
		"parent_team_id": float64(123),
		"company_id":     float64(456),
		"project_id":     float64(789),
		"user_ids": []any{
			float64(1),
			float64(2),
			float64(3),
		},
	})
}

// TestTeamUpdateParentReachesTheWire pins parentTeamId on the request body. The
// endpoint reads a missing parentTeamId as "leave the hierarchy alone" and a
// zero as "move to the top level", so an update that does not name it must
// carry no key at all.
func TestTeamUpdateParentReachesTheWire(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		want      any
		absent    bool
	}{{
		name:      "not named",
		arguments: map[string]any{"id": float64(123), "name": "Example"},
		absent:    true,
	}, {
		name:      "moved under another team",
		arguments: map[string]any{"id": float64(123), "parent_team_id": float64(777)},
		want:      float64(777),
	}, {
		name:      "moved to the top level",
		arguments: map[string]any{"id": float64(123), "parent_team_id": float64(0)},
		want:      float64(0),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, body := mcpServerMockWithRequestBody(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamUpdate.String(), tt.arguments)

			var payload struct {
				Team map[string]any `json:"team"`
			}
			if err := json.Unmarshal(*body, &payload); err != nil {
				t.Fatalf("failed to decode request body: %s", err)
			}
			got, ok := payload.Team["parentTeamId"]
			switch {
			case tt.absent && ok:
				t.Errorf("expected parentTeamId to be absent from the request body, got %v", got)
			case !tt.absent && !ok:
				t.Errorf("expected parentTeamId in the request body, got %v", payload.Team)
			case !tt.absent && got != tt.want:
				t.Errorf("expected parentTeamId to be %v, got %v", tt.want, got)
			}
		})
	}
}

func TestTeamDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTeamGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestTeamList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

// TestTeamDeletedDateValidatesAgainstOutputSchema pins the JSON shape of the
// team `deletedDate` against the tools' published output schemas. It is the
// SDK's only twapi.OptionalDateTime field, and that type is defined over
// time.Time, so the reflected schema described it as an object while
// MarshalJSON emitted an RFC3339 string — every validating client discarded the
// whole response.
//
// The other team tests pass `{}` as the body, which leaves DeletedAt nil and
// encodes as null, so they never exercised either half. Both cases belong here:
// the empty string the API sends for every live team (encoding/json allocates
// the pointer before UnmarshalJSON runs, so it survives as a non-nil pointer to
// the zero time), and the timestamp it sends for a deleted one.
func TestTeamDeletedDateValidatesAgainstOutputSchema(t *testing.T) {
	tests := []struct {
		name        string
		deletedDate string
	}{
		{name: "live team", deletedDate: `""`},
		{name: "deleted team", deletedDate: `"2026-01-02T03:04:05Z"`},
		{name: "null", deletedDate: `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team := `{"id":"123","name":"Example","deletedDate":` + tt.deletedDate + `}`

			mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{"team":`+team+`}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamGet.String(), map[string]any{
				"id": float64(123),
			}, testutil.ExecuteToolRequestWithCheckMessage(
				checkStructuredContentMatchesOutputSchema(twprojects.MethodTeamGet.String()),
			))

			mcpServer = mcpServerMock(t, http.StatusOK, []byte(`{"teams":[`+team+`]}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamList.String(), map[string]any{
				"page":      float64(1),
				"page_size": float64(10),
			}, testutil.ExecuteToolRequestWithCheckMessage(
				checkStructuredContentMatchesOutputSchema(twprojects.MethodTeamList.String()),
			))
		})
	}
}

// checkStructuredContentMatchesOutputSchema returns a check that validates a
// tool result's StructuredContent against the output schema the named tool
// publishes in tools/list.
//
// Neither the test harness nor the server validates this, so a mismatch is
// invisible in-process and only surfaces at a validating client, which then
// discards a response the server returned successfully. StructuredContent holds
// the Go value rather than decoded JSON, so it has to be round-tripped through
// encoding/json to see what the client will actually receive.
func checkStructuredContentMatchesOutputSchema(toolName string) func(t *testing.T, result mcp.Result) {
	return func(t *testing.T, result mcp.Result) {
		t.Helper()

		testutil.CheckMessage(t, result)

		toolResult, ok := result.(*mcp.CallToolResult)
		if !ok {
			t.Fatalf("unexpected result type: %T", result)
		}
		if toolResult.StructuredContent == nil {
			t.Fatalf("tool %s returned no structured content", toolName)
		}

		schema := outputSchemaFor(t, toolName)
		resolved, err := schema.Resolve(nil)
		if err != nil {
			t.Fatalf("failed to resolve output schema for %s: %s", toolName, err)
		}

		encoded, err := json.Marshal(toolResult.StructuredContent)
		if err != nil {
			t.Fatalf("failed to encode structured content for %s: %s", toolName, err)
		}
		var decoded any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("failed to decode structured content for %s: %s", toolName, err)
		}

		if err := resolved.Validate(decoded); err != nil {
			t.Errorf("tool %s: structured content does not match its output schema: %s\nbody: %s",
				toolName, err, encoded)
		}
		checkSchemaFormats(t, toolName, resolved.Schema(), decoded, "data")
	}
}

// checkSchemaFormats asserts that every string value sitting under a schema node
// carrying a "date" or "date-time" format actually parses as one.
//
// jsonschema-go treats format as an annotation and never asserts it — there is no
// option to turn it on, and validate.go does not mention the keyword — so
// Validate alone passes a body that a client running Ajv-with-formats rejects
// outright. That blind spot is why the empty string the v1 teams route sends for
// deletedDate survived two rounds of fixes: widening the declared type to
// null|string satisfied the validator while leaving the format unmet.
//
// It walks properties, items and additionalProperties only. A format nested in an
// anyOf/oneOf branch is conditional on that branch matching, which is not
// something this check can decide, so it is deliberately not asserted.
func checkSchemaFormats(t *testing.T, toolName string, schema *jsonschema.Schema, value any, path string) {
	t.Helper()

	if schema == nil {
		return
	}
	if held, ok := value.(string); ok {
		var layout string
		switch schema.Format {
		case "date-time":
			layout = time.RFC3339
		case "date":
			layout = time.DateOnly
		}
		if layout != "" {
			if _, err := time.Parse(layout, held); err != nil {
				t.Errorf("tool %s: %s is %q, which the schema declares as format %q: %s",
					toolName, path, held, schema.Format, err)
			}
		}
	}

	switch held := value.(type) {
	case map[string]any:
		for key, nested := range held {
			if property, ok := schema.Properties[key]; ok {
				checkSchemaFormats(t, toolName, property, nested, path+"/"+key)
				continue
			}
			checkSchemaFormats(t, toolName, schema.AdditionalProperties, nested, path+"/"+key)
		}
	case []any:
		for i, nested := range held {
			checkSchemaFormats(t, toolName, schema.Items, nested, fmt.Sprintf("%s/%d", path, i))
		}
	}
}

// outputSchemaFor looks up the output schema a registered tool publishes.
func outputSchemaFor(t *testing.T, toolName string) *jsonschema.Schema {
	t.Helper()

	group := twprojects.DefaultToolsetGroup(false, true, testutil.ProjectsEngineMock(http.StatusOK, nil))
	for _, toolset := range group.Toolsets {
		for _, tool := range toolset.GetAvailableTools() {
			if tool.Tool.Name != toolName {
				continue
			}
			schema, ok := tool.Tool.OutputSchema.(*jsonschema.Schema)
			if !ok {
				t.Fatalf("tool %s: OutputSchema is not *jsonschema.Schema (got %T)",
					toolName, tool.Tool.OutputSchema)
			}
			return schema
		}
	}
	t.Fatalf("tool %s is not registered", toolName)
	return nil
}

func TestTeamListByCompany(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamList.String(), map[string]any{
		"company_id":  float64(123),
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

func TestTeamListByProject(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTeamList.String(), map[string]any{
		"project_id":  float64(123),
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}
