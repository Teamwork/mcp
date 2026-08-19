package mcphttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// resourceDocumentation is the guide an OAuth client is pointed at to understand
// this authorization flow.
const resourceDocumentation = "https://apidocs.teamwork.com/guides/teamwork/app-login-flow"

// Health registers a GET/OPTIONS health check that requires no authentication.
func Health(mux *http.ServeMux, path string) {
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if !allowGetOptions(w, r) {
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// ProtectedResource registers the RFC 9728 protected-resource metadata an
// unauthorised client fetches to discover where and for what to authorise.
//
// The advertised scopes come from what the registered groups declare, so a scope
// a client may ask for always has a group behind it. A group declaring a scope
// the authorization server does not know will still fail at registration — the
// authorization server keeps its own catalogue, and this endpoint cannot check
// against it.
//
// https://datatracker.ietf.org/doc/html/rfc9728/#section-2
func ProtectedResource(mux *http.ServeMux, resources config.Resources, groups []*toolsets.ToolsetGroup) {
	scopesSupported, err := json.Marshal(toolsets.Scopes(groups))
	if err != nil {
		// Marshalling a []string cannot fail, but advertising no scope beats
		// serving malformed metadata if it ever does.
		resources.Logger().Error("failed to encode supported scopes",
			slog.String("error", err.Error()),
		)
		scopesSupported = []byte("[]")
	}

	body := []byte(`{
  "resource": "` + resources.Info.MCPURL + `",
  "authorization_servers": ["` + resources.Info.APIURL + `"],
  "bearer_methods_supported": ["header"],
  "resource_documentation": "` + resourceDocumentation + `",
  "scopes_supported": ` + string(scopesSupported) + `
}`)

	mux.HandleFunc("/.well-known/oauth-protected-resource", func(w http.ResponseWriter, r *http.Request) {
		if !allowGetOptions(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		allowCORS(w)
		w.WriteHeader(http.StatusOK)

		if r.Method == http.MethodOptions {
			return
		}
		_, _ = w.Write(body)
	})
}

// allowGetOptions rejects anything but GET and OPTIONS, reporting whether the
// caller should continue.
func allowGetOptions(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// allowCORS opens an unauthenticated discovery endpoint to browser clients,
// which fetch it cross-origin before they hold any credential.
func allowCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "*")
}
