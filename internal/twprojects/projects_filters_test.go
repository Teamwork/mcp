package twprojects_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestProjectListFiltersReachTheWire drives every filter twprojects-list_projects
// advertises and asserts the query parameter it is supposed to produce. The mock
// replies with the same canned body whatever the query says, so a filter that is
// declared in the schema but never bound looks exactly like a working one from
// the tool result alone — the query string is the only place the difference shows.
func TestProjectListFiltersReachTheWire(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		query     string
		want      string
	}{{
		name:      "project category ids",
		arguments: map[string]any{"project_category_ids": []any{float64(777), float64(12345)}},
		query:     "projectCategoryIds",
		want:      "777,12345",
	}, {
		name:      "include subcategories",
		arguments: map[string]any{"include_subcategories": true},
		query:     "includeSubCategories",
		want:      "true",
	}, {
		name:      "project statuses",
		arguments: map[string]any{"project_statuses": []any{"late", "upcoming"}},
		query:     "projectStatuses",
		want:      "late,upcoming",
	}, {
		name:      "project healths",
		arguments: map[string]any{"project_healths": []any{"bad", "not_set"}},
		query:     "projectHealths",
		want:      "1,0",
	}, {
		name:      "user ids",
		arguments: map[string]any{"user_ids": []any{float64(777)}},
		query:     "usersWithExplicitMembershipIds",
		want:      "777",
	}, {
		name:      "team ids",
		arguments: map[string]any{"team_ids": []any{float64(777), float64(12345)}},
		query:     "teamIds",
		want:      "777,12345",
	}, {
		name:      "project owner ids",
		arguments: map[string]any{"project_owner_ids": []any{float64(777)}},
		query:     "projectOwnerIds",
		want:      "777",
	}, {
		name:      "company ids",
		arguments: map[string]any{"company_ids": []any{float64(12345)}},
		query:     "projectCompanyIds",
		want:      "12345",
	}, {
		name:      "only starred",
		arguments: map[string]any{"only_starred": true},
		query:     "onlyStarredProjects",
		want:      "true",
	}, {
		name:      "only admin access",
		arguments: map[string]any{"only_admin_access": true},
		query:     "onlyProjectsWithAdminAccess",
		want:      "true",
	}, {
		name:      "hide observed",
		arguments: map[string]any{"hide_observed": true},
		query:     "hideObservedProjects",
		want:      "true",
	}, {
		name:      "include archived",
		arguments: map[string]any{"include_archived": true},
		query:     "includeArchivedProjects",
		want:      "true",
	}, {
		name:      "only archived",
		arguments: map[string]any{"only_archived": true},
		query:     "onlyArchivedProjects",
		want:      "true",
	}, {
		name:      "include tentative",
		arguments: map[string]any{"include_tentative": true},
		query:     "includeTentativeProjects",
		want:      "true",
	}, {
		name:      "search term",
		arguments: map[string]any{"search_term": "migration"},
		query:     "searchTerm",
		want:      "migration",
	}, {
		name:      "tag ids",
		arguments: map[string]any{"tag_ids": []any{float64(777)}},
		query:     "projectTagIds",
		want:      "777",
	}, {
		name:      "match all tags",
		arguments: map[string]any{"tag_ids": []any{float64(777)}, "match_all_tags": true},
		query:     "matchAllProjectTags",
		want:      "true",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectList.String(), tt.arguments)

			if got := lastURL.Query().Get(tt.query); got != tt.want {
				t.Errorf("expected %s=%q in request query but got %q (raw query: %s)",
					tt.query, tt.want, got, lastURL.RawQuery)
			}
		})
	}
}

// TestProjectListUpdatedAfterReachesTheWire is separated from the table above
// because the SDK renders the filter as an RFC 3339 timestamp, so the value the
// caller passes is not the value on the wire.
func TestProjectListUpdatedAfterReachesTheWire(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectList.String(), map[string]any{
		"updated_after": "2026-08-01",
	})

	if got := lastURL.Query().Get("updatedAfter"); !strings.HasPrefix(got, "2026-08-01T") {
		t.Errorf("expected updatedAfter to start with %q but got %q (raw query: %s)",
			"2026-08-01T", got, lastURL.RawQuery)
	}
}

// TestProjectListFiltersAreOmittedWhenUnset guards the defaults: an unset filter
// must send no parameter at all, so the endpoint's own behaviour stands. A
// boolean bound to a plain bool rather than a *bool would send "false" here and
// silently override an API default that is true.
func TestProjectListFiltersAreOmittedWhenUnset(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectList.String(), map[string]any{})

	for _, query := range []string{
		"includeSubCategories", "projectStatuses", "projectHealths",
		"usersWithExplicitMembershipIds", "teamIds", "projectOwnerIds", "projectCompanyIds",
		"onlyStarredProjects", "onlyProjectsWithAdminAccess", "hideObservedProjects",
		"includeArchivedProjects", "onlyArchivedProjects", "includeTentativeProjects",
		"updatedAfter",
	} {
		if lastURL.Query().Has(query) {
			t.Errorf("expected no %s in request query but got %q (raw query: %s)",
				query, lastURL.Query().Get(query), lastURL.RawQuery)
		}
	}
}

// TestProjectCountInheritsTheNewFilters pins that the count tool, which copies
// its input schema from the list tool, keeps the filters and drops only the row
// shaping. "How many of my projects are late" is the question the count tool
// exists to answer in one call.
func TestProjectCountInheritsTheNewFilters(t *testing.T) {
	mcpServer, lastURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"meta":{"page":{"count":7}}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectCount.String(), map[string]any{
		"project_statuses": []any{"late"},
		"user_ids":         []any{float64(777)},
	})

	if got, want := lastURL.Query().Get("projectStatuses"), "late"; got != want {
		t.Errorf("expected projectStatuses=%q in request query but got %q (raw query: %s)",
			want, got, lastURL.RawQuery)
	}
	if got, want := lastURL.Query().Get("usersWithExplicitMembershipIds"), "777"; got != want {
		t.Errorf("expected usersWithExplicitMembershipIds=%q in request query but got %q (raw query: %s)",
			want, got, lastURL.RawQuery)
	}
}
