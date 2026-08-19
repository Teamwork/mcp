package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// TestProtectedResourceScopesComeFromTheGroups pins that the scopes advertised
// in the RFC 9728 metadata are the ones the registered ToolsetGroups declare.
// The two used to be written out separately, so a group could ship with a scope
// no client was ever told it could ask for — the tools would then be filtered
// out of tools/list for every token.
func TestProtectedResourceScopesComeFromTheGroups(t *testing.T) {
	resources := newTestResources(t)

	groups, err := newToolsetGroups(resources)
	if err != nil {
		t.Fatalf("failed to build toolset groups: %v", err)
	}

	want := toolsets.Scopes(groups)
	if len(want) == 0 {
		t.Fatal("no group declares a scope, so tools/list filtering is inert")
	}

	server := httptest.NewServer(newRouter(resources, want))
	defer server.Close()

	response, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource")
	if err != nil {
		t.Fatalf("failed to request protected resource metadata: %v", err)
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	var metadata struct {
		Resource        string   `json:"resource"`
		ScopesSupported []string `json:"scopes_supported"`
	}
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}

	if !slices.Equal(metadata.ScopesSupported, want) {
		t.Errorf("scopes_supported = %v, want %v", metadata.ScopesSupported, want)
	}
}

// TestProtectedResourceScopesAreValidJSON guards the hand-built JSON body: the
// scopes are interpolated into a string literal, so an encoding slip produces
// metadata no OAuth client can parse.
func TestProtectedResourceScopesAreValidJSON(t *testing.T) {
	resources := newTestResources(t)

	for _, scopes := range [][]string{nil, {}, {"projects"}, {"projects", "desk", "spaces", "chat"}} {
		server := httptest.NewServer(newRouter(resources, scopes))

		response, err := server.Client().Get(server.URL + "/.well-known/oauth-protected-resource")
		if err != nil {
			server.Close()
			t.Fatalf("scopes %v: failed to request metadata: %v", scopes, err)
		}

		var metadata map[string]any
		err = json.NewDecoder(response.Body).Decode(&metadata)
		response.Body.Close() //nolint:errcheck
		server.Close()

		if err != nil {
			t.Errorf("scopes %v: metadata is not valid JSON: %v", scopes, err)
		}
	}
}

func newTestResources(t *testing.T) config.Resources {
	t.Helper()

	var resources config.Resources
	resources.Info.MCPURL = "https://mcp.example.com"
	resources.Info.APIURL = "https://example.com"
	return resources
}
