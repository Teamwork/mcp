package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestJobRoleCreate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"jobRole":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleCreate.String(), map[string]any{
		"name": "Example",
	})
}

func TestJobRoleUpdate(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleUpdate.String(), map[string]any{
		"id":   float64(123),
		"name": "Example",
	})
}

func TestJobRoleDelete(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNoContent, nil)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleDelete.String(), map[string]any{
		"id": float64(123),
	})
}

func TestJobRoleGet(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleGet.String(), map[string]any{
		"id": float64(123),
	})
}

func TestJobRoleList(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleList.String(), map[string]any{
		"search_term": "test",
		"page":        float64(1),
		"page_size":   float64(10),
	})
}

// TestJobRoleGetRequestsMembership pins the sideload the get depends on. The
// endpoint leaves users and primaryUsers out of the job role payload unless it
// is asked for, so without this the tool reports a role with members as having
// none — and reports it identically to a role that really is empty.
func TestJobRoleGetRequestsMembership(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"jobRole":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleGet.String(), map[string]any{
		"id": float64(123),
	})

	if !strings.HasSuffix(requestURL.Path, "/projects/api/v3/jobroles/123.json") {
		t.Errorf("expected the single job role route, got %s", requestURL.Path)
	}
	query := requestURL.Query()
	if got := query.Get("include"); got != "users" {
		t.Errorf("expected the users sideload, got %q", got)
	}
	if got := query.Get("fields[users]"); got != "id,firstName,lastName" {
		t.Errorf("expected the sideload narrowed to names, got %q", got)
	}
}

// TestJobRoleGetSelectionKeepsMembership is the case the membership defect was
// reported through: a selection naming users came back as an empty array,
// because a field selection cannot surface an attribute the sideload is what
// populates. Unlike the file tools, a selection here must not drop the include.
func TestJobRoleGetSelectionKeepsMembership(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"jobRole":{"id":123,"name":"Survey Tech Data","users":[{"id":456,"type":"users"}]}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleGet.String(), map[string]any{
		"id":     float64(123),
		"fields": []any{"name", "users"},
	})

	query := requestURL.Query()
	if got := query.Get("fields[jobRoles]"); got != "name,users,id" {
		t.Errorf("expected the selection plus id, got %q", got)
	}
	if got := query.Get("include"); got != "users" {
		t.Errorf("expected the users sideload to survive the selection, got %q", got)
	}
}

// TestJobRoleGetUnrelatedSelectionDropsMembership pins the other half: a
// selection that names neither membership attribute has no use for the
// sideload, and the endpoint computes it per row when asked.
func TestJobRoleGetUnrelatedSelectionDropsMembership(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"jobRole":{"id":123,"name":"Survey Tech Data"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleGet.String(), map[string]any{
		"id":     float64(123),
		"fields": []any{"name"},
	})

	query := requestURL.Query()
	if query.Has("include") {
		t.Errorf("expected no sideload under an unrelated selection, got %q", query.Get("include"))
	}
	if query.Has("fields[users]") {
		t.Errorf("expected no user selection without the sideload, got %q", query.Get("fields[users]"))
	}
}

// TestJobRoleListMembershipReachesTheWire pins the same rule on the list, per
// response shape: verbose rows carry membership, a compact row does not, and a
// count never pays for it.
func TestJobRoleListMembershipReachesTheWire(t *testing.T) {
	tests := []struct {
		name          string
		arguments     map[string]any
		wantInclude   string
		wantJobRoles  string
		wantUserField string
	}{{
		name:          "verbose by default",
		arguments:     map[string]any{},
		wantInclude:   "users",
		wantUserField: "id,firstName,lastName",
	}, {
		name:         "compact rows drop it",
		arguments:    map[string]any{"verbose": false},
		wantJobRoles: "id,name",
	}, {
		name:          "a selection naming users keeps it",
		arguments:     map[string]any{"fields": []any{"name", "primaryUsers"}},
		wantInclude:   "users",
		wantJobRoles:  "name,primaryUsers,id",
		wantUserField: "id,firstName,lastName",
	}, {
		name:         "an unrelated selection drops it",
		arguments:    map[string]any{"fields": []any{"name"}},
		wantJobRoles: "name,id",
	}, {
		name:      "a count never pays for it",
		arguments: map[string]any{"count_only": true},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
				[]byte(`{"jobRoles":[{"id":123}],"meta":{"page":{"count":1}}}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodJobRoleList.String(), tt.arguments)

			query := requestURL.Query()
			for key, want := range map[string]string{
				"include":          tt.wantInclude,
				"fields[jobRoles]": tt.wantJobRoles,
				"fields[users]":    tt.wantUserField,
			} {
				if got := query.Get(key); got != want {
					t.Errorf("expected %s=%q on the query string, got %q", key, want, got)
				}
			}
		})
	}
}
