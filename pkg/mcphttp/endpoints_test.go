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

// TestProtectedResourceRFC9728Path pins the metadata URL shape RFC 9728 tells a
// client to build for itself: the well-known segment goes between the host and
// the resource path. Serving only the other order answered those clients with a
// 405 from the MCP handler, stranding them before the authorization server.
func TestProtectedResourceRFC9728Path(t *testing.T) {
	tests := []struct {
		name     string
		mcpURL   string
		profiles []string
		path     string
		want     string
	}{{
		name:   "no profile",
		mcpURL: "https://mcp.example.com",
		path:   "/.well-known/oauth-protected-resource",
		want:   "https://mcp.example.com",
	}, {
		name:     "server scoped to one profile",
		mcpURL:   "https://mcp.example.com/analyst",
		profiles: []string{"analyst"},
		path:     "/.well-known/oauth-protected-resource/analyst",
		want:     "https://mcp.example.com/analyst",
	}, {
		name:     "server exposing several profiles",
		mcpURL:   "https://mcp.example.com",
		profiles: []string{"analyst", "ops"},
		path:     "/.well-known/oauth-protected-resource/ops",
		want:     "https://mcp.example.com/ops",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resources config.Resources
			resources.Info.MCPURL = tt.mcpURL
			resources.Info.MCPProfiles = tt.profiles
			resources.Info.APIURL = "https://example.com"

			mux := http.NewServeMux()
			mcphttp.ProtectedResource(mux, resources, nil)

			server := httptest.NewServer(mux)
			defer server.Close()

			response, err := server.Client().Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to request metadata: %v", err)
			}
			defer response.Body.Close() //nolint:errcheck

			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
			}

			var metadata struct {
				Resource string `json:"resource"`
			}
			if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
				t.Fatalf("metadata is not valid JSON: %v", err)
			}
			if metadata.Resource != tt.want {
				t.Errorf("resource = %q, want %q", metadata.Resource, tt.want)
			}
		})
	}
}

// TestProtectedResourceKeepsLegacyPath pins that the path shape this server
// advertised before still answers, so clients that cached it keep working.
func TestProtectedResourceKeepsLegacyPath(t *testing.T) {
	var resources config.Resources
	resources.Info.MCPURL = "https://mcp.example.com/analyst"
	resources.Info.MCPProfiles = []string{"analyst"}
	resources.Info.APIURL = "https://example.com"

	mux := http.NewServeMux()
	mcphttp.ProtectedResource(mux, resources, nil)

	server := httptest.NewServer(mux)
	defer server.Close()

	// StripProfile trims the profile before the mux sees it, so this is the path
	// the legacy URL arrives as.
	response, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatalf("failed to request metadata: %v", err)
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

// TestProtectedResourceURL pins the URL the 401 challenge advertises, which must
// be one the server actually serves.
func TestProtectedResourceURL(t *testing.T) {
	tests := []struct {
		resource string
		want     string
	}{
		{resource: "https://mcp.example.com", want: "https://mcp.example.com/.well-known/oauth-protected-resource"},
		{resource: "https://mcp.example.com/", want: "https://mcp.example.com/.well-known/oauth-protected-resource"},
		{
			resource: "https://mcp.example.com/analyst",
			want:     "https://mcp.example.com/.well-known/oauth-protected-resource/analyst",
		},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			if got := mcphttp.ProtectedResourceURL(tt.resource); got != tt.want {
				t.Errorf("ProtectedResourceURL(%q) = %q, want %q", tt.resource, got, tt.want)
			}
		})
	}
}
