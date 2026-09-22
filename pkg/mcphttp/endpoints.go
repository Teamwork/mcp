package mcphttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// resourceDocumentation is the guide an OAuth client is pointed at to understand
// this authorization flow.
const resourceDocumentation = "https://apidocs.teamwork.com/guides/teamwork/app-login-flow"

// protectedResourcePath is the well-known segment RFC 9728 reserves for
// protected-resource metadata.
const protectedResourcePath = "/.well-known/oauth-protected-resource"

// ProtectedResourceURL returns where RFC 9728 says the metadata for the given
// resource identifier lives: the well-known segment goes between the host and
// the resource's path, so a profile URL such as https://host/analyst is
// described at https://host/.well-known/oauth-protected-resource/analyst.
//
// https://datatracker.ietf.org/doc/html/rfc9728#section-3.1
func ProtectedResourceURL(resource string) string {
	parsed, err := url.Parse(resource)
	if err != nil || parsed.Host == "" {
		// Not a URL we can take apart; keep the caller pointed somewhere rather
		// than at an empty string.
		return resource + protectedResourcePath
	}
	parsed.Path = protectedResourcePath + resourcePath(resource)
	return parsed.String()
}

// resourcePath returns the resource identifier's path component, normalised to
// either "" or a non-empty path with no trailing slash.
func resourcePath(resource string) string {
	parsed, err := url.Parse(resource)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(parsed.Path, "/")
}

// serverCardPaths are the well-known locations clients probe for an MCP server
// card, next to the endpoint rather than on the API host where the canonical
// document lives.
var serverCardPaths = []string{
	"/.well-known/mcp",
	"/.well-known/mcp.json",
	"/.well-known/mcp/server-card.json",
}

// ServerCard redirects the well-known server-card paths to the canonical
// document on the API host. Redirecting rather than serving a copy keeps one
// source of truth; without these the paths fall through to the MCP handler,
// which answers 405 because it only accepts POST.
func ServerCard(mux *http.ServeMux, resources config.Resources) {
	target := resources.Info.APIURL + "/.well-known/mcp.json"

	for _, path := range serverCardPaths {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if !allowGetOptions(w, r) {
				return
			}
			allowCORS(w)

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Redirect(w, r, target, http.StatusPermanentRedirect)
		})
	}
}

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

	serve := func(resource string) http.HandlerFunc {
		body := []byte(`{
  "resource": "` + resource + `",
  "authorization_servers": ["` + resources.Info.APIURL + `"],
  "bearer_methods_supported": ["header"],
  "resource_documentation": "` + resourceDocumentation + `",
  "scopes_supported": ` + string(scopesSupported) + `
}`)

		return func(w http.ResponseWriter, r *http.Request) {
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
		}
	}

	mux.HandleFunc(protectedResourcePath, serve(resources.Info.MCPURL))

	// A client that builds the metadata URL itself, as RFC 9728 tells it to,
	// asks for the well-known segment before the profile. Without these routes
	// that request falls through to the MCP handler and comes back 405, which
	// strands the client before it ever reaches the authorization server.
	if path := resourcePath(resources.Info.MCPURL); path != "" {
		// This server is one profile's resource, so it answers for that path.
		mux.HandleFunc(protectedResourcePath+path, serve(resources.Info.MCPURL))
		return
	}
	// This server answers for every profile it exposes as a path prefix.
	for _, profile := range slices.Compact(slices.Sorted(slices.Values(resources.Info.MCPProfiles))) {
		mux.HandleFunc(protectedResourcePath+"/"+profile, serve(resources.Info.MCPURL+"/"+profile))
	}
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
