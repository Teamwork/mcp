package mcphttp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/mcphttp"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// TestProtectedResourceIsValidJSON guards the hand-built metadata body: the
// scopes are interpolated into a string literal, so an encoding slip produces
// metadata no OAuth client can parse. The no-group case matters most — it must
// advertise an empty array, not null.
func TestProtectedResourceIsValidJSON(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
	}{
		{name: "no groups", scopes: nil},
		{name: "one scope", scopes: []string{"projects"}},
		{name: "several scopes", scopes: []string{"projects", "desk", "spaces", "chat"}},
		{name: "groups sharing a scope", scopes: []string{"projects", "projects"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := make([]*toolsets.ToolsetGroup, 0, len(tt.scopes))
			for i, scope := range tt.scopes {
				groups = append(groups, toolsets.NewToolsetGroup(false).
					SetNamespace("tw"+scope+string(rune('a'+i)), scope))
			}

			var resources config.Resources
			resources.Info.MCPURL = "https://mcp.example.com"
			resources.Info.APIURL = "https://example.com"

			mux := http.NewServeMux()
			mcphttp.ProtectedResource(mux, resources, groups)

			server := httptest.NewServer(mux)
			defer server.Close()

			response, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource")
			if err != nil {
				t.Fatalf("failed to request metadata: %v", err)
			}
			defer response.Body.Close() //nolint:errcheck

			var metadata struct {
				Resource        string   `json:"resource"`
				ScopesSupported []string `json:"scopes_supported"`
			}
			if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
				t.Fatalf("metadata is not valid JSON: %v", err)
			}
			if metadata.Resource != resources.Info.MCPURL {
				t.Errorf("resource = %q, want %q", metadata.Resource, resources.Info.MCPURL)
			}
			if metadata.ScopesSupported == nil {
				t.Error("scopes_supported decoded as null; it must always be an array")
			}
			if want := toolsets.Scopes(groups); !slices.Equal(metadata.ScopesSupported, want) {
				t.Errorf("scopes_supported = %v, want %v", metadata.ScopesSupported, want)
			}
		})
	}
}

// TestHealthRejectsWrites pins that the unauthenticated health endpoint answers
// only GET and OPTIONS. It sits in front of authentication, so anything else it
// accepted would be an unauthenticated write path.
func TestHealthRejectsWrites(t *testing.T) {
	mux := http.NewServeMux()
	mcphttp.Health(mux, "/api/health")

	server := httptest.NewServer(mux)
	defer server.Close()

	for method, wantStatus := range map[string]int{
		http.MethodGet:     http.StatusOK,
		http.MethodOptions: http.StatusOK,
		http.MethodPost:    http.StatusMethodNotAllowed,
		http.MethodPut:     http.StatusMethodNotAllowed,
		http.MethodDelete:  http.StatusMethodNotAllowed,
	} {
		request, err := http.NewRequestWithContext(t.Context(), method, server.URL+"/api/health", nil)
		if err != nil {
			t.Fatalf("%s: failed to build request: %v", method, err)
		}
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatalf("%s: failed to request health: %v", method, err)
		}
		_ = response.Body.Close()

		if response.StatusCode != wantStatus {
			t.Errorf("%s /api/health = %d, want %d", method, response.StatusCode, wantStatus)
		}
	}
}
