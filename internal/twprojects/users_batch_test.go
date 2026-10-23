package twprojects_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// userBulkCreateBody is the request body create_users sends.
type userBulkCreateBody struct {
	People []map[string]any `json:"people"`
	// Options is decoded loosely so a missing partialSuccess reads as absent.
	Options map[string]any `json:"options"`
}

// singleUserBulkCreateRequest asserts that exactly one request reached the bulk
// route and decodes its body.
func singleUserBulkCreateRequest(t *testing.T, recorded []testutil.ProjectsRecordedRequest) userBulkCreateBody {
	t.Helper()
	if len(recorded) != 1 {
		t.Fatalf("expected exactly 1 request, got %d", len(recorded))
	}
	entry := recorded[0]
	if entry.Method != http.MethodPost || entry.URL.Path != "/projects/api/v3/people/bulk.json" {
		t.Errorf("expected POST /projects/api/v3/people/bulk.json, got %s %s", entry.Method, entry.URL.Path)
	}
	var body userBulkCreateBody
	if err := json.Unmarshal(entry.Body, &body); err != nil {
		t.Fatalf("failed to decode body %q: %v", entry.Body, err)
	}
	if body.Options["partialSuccess"] != true {
		t.Errorf("expected options.partialSuccess to be sent as true, got %v", body.Options)
	}
	return body
}

// TestUserCreateBatchSendsOneRequest pins a full success: every user goes in a
// single request with its own fields, no invitation is requested when
// send_invite is omitted, and each item is reported with its new ID.
func TestUserCreateBatchSendsOneRequest(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated,
		[]byte(`{"people":[{"id":101,"email":"john@example.com"},{"id":102,"email":"jane@example.com"}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreateBatch.String(), map[string]any{
		"users": []any{
			map[string]any{"first_name": "John", "last_name": "Doe", "email": "john@example.com"},
			map[string]any{
				"first_name": "Jane", "last_name": "Doe", "email": "jane@example.com",
				"title": "Designer", "admin": true, "type": "collaborator", "company_id": float64(777),
			},
		},
	}, expectBatchResult(false, "Created 2 of 2 users",
		`item 1 "john@example.com" → user 101`, `item 2 "jane@example.com" → user 102`))

	body := singleUserBulkCreateRequest(t, *recorded)
	if len(body.People) != 2 {
		t.Fatalf("expected 2 people in the request, got %d", len(body.People))
	}
	for i, person := range body.People {
		// The endpoint invites a person whose entry omits sendInvite, so the
		// field must be present and false.
		if sendInvite, ok := person["sendInvite"]; !ok || sendInvite != false {
			t.Errorf("person %d: expected sendInvite false, got %v (present: %t)", i, sendInvite, ok)
		}
	}
	first, second := body.People[0], body.People[1]
	if first["firstName"] != "John" || first["lastName"] != "Doe" || first["email"] != "john@example.com" {
		t.Errorf("unexpected first person: %v", first)
	}
	for key, want := range map[string]any{
		"title": "Designer", "isAdmin": true, "userType": "collaborator", "companyId": float64(777),
	} {
		if second[key] != want {
			t.Errorf("second person: expected %s=%v, got %v", key, want, second[key])
		}
	}
}

// TestUserCreateBatchSendInviteReachesTheWire pins the opt-in: send_invite is
// applied to every user, and false is sent as false.
func TestUserCreateBatchSendInviteReachesTheWire(t *testing.T) {
	for _, sendInvite := range []bool{true, false} {
		t.Run(fmt.Sprint(sendInvite), func(t *testing.T) {
			mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated,
				[]byte(`{"people":[{"id":101},{"id":102}]}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreateBatch.String(), map[string]any{
				"send_invite": sendInvite,
				"users": []any{
					map[string]any{"first_name": "John", "last_name": "Doe", "email": "john@example.com"},
					map[string]any{"first_name": "Jane", "last_name": "Doe", "email": "jane@example.com"},
				},
			}, expectBatchResult(false, "Created 2 of 2 users"))

			for i, person := range singleUserBulkCreateRequest(t, *recorded).People {
				if person["sendInvite"] != sendInvite {
					t.Errorf("person %d: expected sendInvite %t, got %v", i, sendInvite, person["sendInvite"])
				}
			}
		})
	}
}

// TestUserCreateBatchReportsEachFailure maps a partial success back to the
// items: the created users take the indexes the endpoint did not report as
// failed, in order.
func TestUserCreateBatchReportsEachFailure(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{
		"people":[{"id":101},{"id":103}],
		"failed":[{"index":1,"errors":[{"title":"invalid payload","detail":"email is already in use"}]}]
	}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreateBatch.String(), map[string]any{
		"users": []any{
			map[string]any{"first_name": "John", "last_name": "Doe", "email": "john@example.com"},
			map[string]any{"first_name": "Taken", "last_name": "User", "email": "taken@example.com"},
			map[string]any{"first_name": "Jane", "last_name": "Doe", "email": "jane@example.com"},
		},
	}, expectBatchResult(true, "Created 2 of 3 users",
		`item 1 "john@example.com" → user 101`,
		`item 3 "jane@example.com" → user 103`,
		`Failed: item 2 "taken@example.com": email is already in use`))

	singleUserBulkCreateRequest(t, *recorded)
}

// TestUserCreateBatchNothingCreated reports a request the endpoint rejected as
// a whole: nothing was written, and each item it blamed is named with its
// reason.
func TestUserCreateBatchNothingCreated(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusBadRequest, []byte(`{"errors":[
		{"title":"invalid payload","detail":"email is already in use","meta":{"index":0}},
		{"title":"forbidden action","detail":"no seats available","meta":{"index":1}}
	]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreateBatch.String(), map[string]any{
		"users": []any{
			map[string]any{"first_name": "John", "last_name": "Doe", "email": "john@example.com"},
			map[string]any{"first_name": "Jane", "last_name": "Doe", "email": "jane@example.com"},
		},
	}, expectBatchResult(true, "Created 0 of 2 users. Nothing was created.",
		`Failed: item 1 "john@example.com": email is already in use`,
		`Failed: item 2 "jane@example.com": no seats available`))

	singleUserBulkCreateRequest(t, *recorded)
}

// TestUserCreateBatchRejectsATooLongList refuses a list over the maximum before
// anything is sent.
func TestUserCreateBatchRejectsATooLongList(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"people":[]}`))

	users := make([]any, 51)
	for i := range users {
		users[i] = map[string]any{
			"first_name": "User", "last_name": fmt.Sprint(i), "email": fmt.Sprintf("user%d@example.com", i),
		}
	}
	// Call the handler directly: the published maxItems would stop the list in
	// the SDK's schema validation, and the handler must refuse it on its own
	// for clients that skip validation.
	tool := twprojects.UserCreateBatch(nil)
	arguments, err := json.Marshal(map[string]any{"users": users})
	if err != nil {
		t.Fatal(err)
	}
	result, err := tool.Handler(t.Context(), &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Name: twprojects.MethodUserCreateBatch.String(), Arguments: arguments},
	})
	if err != nil {
		t.Fatalf("expected a tool result rather than a Go error: %v", err)
	}
	var text string
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			text += textContent.Text
		}
	}
	if !result.IsError || !strings.Contains(text, "at most 50 items, got 51") {
		t.Errorf("expected the list to be refused, got: %s", text)
	}

	// Through the server too, where the published maxItems rejects it.
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreateBatch.String(), map[string]any{
		"users": users,
	}, expectBatchResult(true))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestUserCreateBatchRejectsAnInvalidItemBeforeSending keeps one bad item from
// sending the rest.
func TestUserCreateBatchRejectsAnInvalidItemBeforeSending(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"people":[{"id":1}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUserCreateBatch.String(), map[string]any{
		"users": []any{
			map[string]any{"first_name": "John", "last_name": "Doe", "email": "john@example.com"},
			map[string]any{"first_name": "Jane", "last_name": "Doe", "email": "jane@example.com", "type": "owner"},
		},
	}, expectBatchResult(true, "Nothing was written", "item 2:"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestUserCreateBatchPublishesItsLimitAndInviteDefault keeps the description,
// the schema and the handler agreeing on the maximum, and the invitation off by
// default.
func TestUserCreateBatchPublishesItsLimitAndInviteDefault(t *testing.T) {
	tool := twprojects.UserCreateBatch(nil).Tool
	schema, ok := tool.InputSchema.(*jsonschema.Schema)
	if !ok {
		t.Fatalf("unexpected input schema type %T", tool.InputSchema)
	}
	users := schema.Properties["users"]
	if users == nil || users.MaxItems == nil || *users.MaxItems != 50 {
		t.Fatalf("expected users to publish maxItems 50, got %+v", users)
	}
	if !strings.Contains(tool.Description, "at most 50 per call") {
		t.Errorf("expected the description to state the maximum: %s", tool.Description)
	}
	if !strings.Contains(tool.Description, "ask the user before setting it") {
		t.Errorf("expected the description to ask before inviting: %s", tool.Description)
	}
	if invite := schema.Properties["send_invite"]; invite == nil || string(invite.Default) != "false" {
		t.Errorf("expected send_invite to default to false, got %+v", invite)
	}
	if strings.Contains(strings.Join(schema.Required, ","), "send_invite") {
		t.Error("send_invite must stay optional")
	}
}
