package twprojects_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	pkgtestutil "github.com/teamwork/mcp/pkg/testutil"
)

// companyBulkPayload is the body twprojects-create_companies sends.
type companyBulkPayload struct {
	Companies []struct {
		Name    string  `json:"name"`
		City    *string `json:"city"`
		TagIDs  []int64 `json:"tagIds"`
		Manager *int64  `json:"clientManagedBy"`
	} `json:"companies"`
	Options *struct {
		PartialSuccess bool `json:"partialSuccess"`
	} `json:"options"`
}

// expectOneBulkRequest asserts the batch went out as a single POST to the bulk
// route with partial success on, and returns its decoded body.
func expectOneBulkRequest(t *testing.T, recorded []testutil.ProjectsRecordedRequest) companyBulkPayload {
	t.Helper()
	if len(recorded) != 1 {
		t.Fatalf("expected exactly 1 request, got %d", len(recorded))
	}
	request := recorded[0]
	if request.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", request.Method)
	}
	if request.URL.Path != "/projects/api/v3/companies/bulk.json" {
		t.Errorf("expected the bulk route, got %s", request.URL.Path)
	}
	var payload companyBulkPayload
	if err := json.Unmarshal(request.Body, &payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	if payload.Options == nil || !payload.Options.PartialSuccess {
		t.Errorf("expected options.partialSuccess=true, got body %s", request.Body)
	}
	return payload
}

func companyNames(payload companyBulkPayload) []string {
	names := make([]string, len(payload.Companies))
	for i, company := range payload.Companies {
		names[i] = company.Name
	}
	return names
}

// TestCompanyCreateBatchReachesTheWire pins one bulk request carrying every
// company, in the order listed, with each company's own fields.
func TestCompanyCreateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Method: http.MethodPost,
		Match:  "/companies/bulk",
		Status: http.StatusCreated,
		Body:   []byte(`{"companies":[{"id":101,"name":"Acme"},{"id":102,"name":"Globex"}]}`),
	}}, http.StatusNotFound, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyCreateBatch.String(), map[string]any{
		"companies": []any{
			map[string]any{"name": "Acme", "city": "Cork", "tag_ids": []any{float64(7)}},
			map[string]any{"name": "Globex", "manager_id": float64(12345)},
		},
	}, expectBatchResult(false, "Created 2 of 2 companies", `item 1 "Acme" → company 101`,
		`item 2 "Globex" → company 102`))

	payload := expectOneBulkRequest(t, *recorded)
	if got := companyNames(payload); !slices.Equal(got, []string{"Acme", "Globex"}) {
		t.Fatalf("expected companies [Acme Globex], got %v", got)
	}
	if city := payload.Companies[0].City; city == nil || *city != "Cork" {
		t.Errorf("expected the first company's city to reach the wire, got %v", city)
	}
	if !slices.Equal(payload.Companies[0].TagIDs, []int64{7}) {
		t.Errorf("expected the first company's tags to reach the wire, got %v", payload.Companies[0].TagIDs)
	}
	if manager := payload.Companies[1].Manager; manager == nil || *manager != 12345 {
		t.Errorf("expected the second company's manager to reach the wire, got %v", manager)
	}
}

// TestCompanyCreateBatchReportsPartialFailure pins how a failed company is
// reported: by its item number and reasons, with the companies around it
// paired to the right IDs, and the result flagged as an error.
func TestCompanyCreateBatchReportsPartialFailure(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Method: http.MethodPost,
		Match:  "/companies/bulk",
		Status: http.StatusCreated,
		Body: []byte(`{"companies":[{"id":101},{"id":103}],"failed":[{"index":1,"errors":[` +
			`{"title":"invalid payload","detail":"emailOne must be a valid email address"}]}]}`),
	}}, http.StatusNotFound, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyCreateBatch.String(), map[string]any{
		"companies": []any{
			map[string]any{"name": "Acme"},
			map[string]any{"name": "Globex", "email_one": "not-an-email"},
			map[string]any{"name": "Initech"},
		},
	}, expectBatchResult(true, "Created 2 of 3 companies", `item 1 "Acme" → company 101`,
		`item 3 "Initech" → company 103`,
		`Failed: item 2 "Globex": invalid payload: emailOne must be a valid email address.`))

	payload := expectOneBulkRequest(t, *recorded)
	if got := companyNames(payload); !slices.Equal(got, []string{"Acme", "Globex", "Initech"}) {
		t.Errorf("expected all three companies in one request, got %v", got)
	}
}

// TestCompanyCreateBatchReportsEveryFailure pins the response for a batch in
// which nothing could be created: the endpoint answers 400 with each error
// carrying its company's position, and each one is still reported per item.
func TestCompanyCreateBatchReportsEveryFailure(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Method: http.MethodPost,
		Match:  "/companies/bulk",
		Status: http.StatusBadRequest,
		Body: []byte(`{"errors":[` +
			`{"title":"invalid payload","detail":"name must be set","meta":{"index":0}},` +
			`{"title":"forbidden action","detail":"cannot add companies","meta":{"index":1}}]}`),
	}}, http.StatusNotFound, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCompanyCreateBatch.String(), map[string]any{
		"companies": []any{
			map[string]any{"name": " "},
			map[string]any{"name": "Globex"},
		},
	}, expectBatchResult(true, "Created 0 of 2 companies",
		`Failed: item 1 " ": invalid payload: name must be set.`,
		`Failed: item 2 "Globex": forbidden action: cannot add companies.`))
	expectOneBulkRequest(t, *recorded)
}

// TestCompanyCreateBatchRejectsALongList pins the size limit on both halves:
// the published schema states it, and the handler refuses a longer list before
// sending anything, for clients that skip schema validation.
func TestCompanyCreateBatchRejectsALongList(t *testing.T) {
	engine, recorded := pkgtestutil.RecordingEngineMock(nil, http.StatusCreated, []byte(`{"companies":[]}`))
	tool := twprojects.CompanyCreateBatch(engine)

	const limit = 50
	property := tool.Tool.InputSchema.(*jsonschema.Schema).Properties["companies"]
	if property == nil || property.MaxItems == nil || *property.MaxItems != limit {
		t.Errorf("expected the companies schema to cap the list at %d", limit)
	}
	if !strings.Contains(tool.Tool.Description, fmt.Sprintf("up to %d per call", limit)) {
		t.Errorf("expected the description to state the limit of %d: %s", limit, tool.Tool.Description)
	}

	companies := make([]map[string]any, limit+1)
	for i := range companies {
		companies[i] = map[string]any{"name": fmt.Sprintf("Company %d", i+1)}
	}
	arguments, err := json.Marshal(map[string]any{"companies": companies})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Handler(t.Context(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: tool.Tool.Name, Arguments: arguments},
	})
	if err != nil {
		t.Fatalf("expected a tool result rather than a Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected a list over the limit to be rejected")
	}
	if len(*recorded) != 0 {
		t.Errorf("expected nothing to be sent, got %d requests", len(*recorded))
	}
}

// TestCompanyCreateBatchRejectsAnInvalidItemBeforeSending pins that a malformed
// item stops the whole batch locally, so the endpoint never sees a partial list.
func TestCompanyCreateBatchRejectsAnInvalidItemBeforeSending(t *testing.T) {
	engine, recorded := pkgtestutil.RecordingEngineMock(nil, http.StatusCreated, []byte(`{"companies":[]}`))
	tool := twprojects.CompanyCreateBatch(engine)
	result, err := tool.Handler(t.Context(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{
			Name:      tool.Tool.Name,
			Arguments: []byte(`{"companies":[{"name":"Acme"},{"city":"Cork"}]}`),
		},
	})
	if err != nil {
		t.Fatalf("expected a tool result rather than a Go error: %v", err)
	}
	if !result.IsError {
		t.Error("expected an item without a name to be rejected")
	}
	if len(*recorded) != 0 {
		t.Errorf("expected nothing to be sent, got %d requests", len(*recorded))
	}
}
