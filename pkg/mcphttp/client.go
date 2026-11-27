package mcphttp

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/teamwork/mcp/pkg/auth"
)

// Where a client's vendor was read from, strongest first. Only the redirect
// host is verified: the authorization code is delivered there. The others are
// what the client says about itself.
const (
	vendorSourceRedirect  = "redirect"
	vendorSourceHeader    = "header"
	vendorSourceUserAgent = "user_agent"
	vendorSourceAppName   = "app_name"
)

// maxAppNameLength caps the client-chosen app name on spans and logs.
const maxAppNameLength = 100

// redirectVendors maps a redirect host, or a domain it is under, to a vendor.
var redirectVendors = map[string]string{
	"anthropic.com":              "anthropic",
	"claude.ai":                  "anthropic",
	"claude.com":                 "anthropic",
	"chatgpt.com":                "openai",
	"openai.com":                 "openai",
	"antigravity.google":         "google",
	"google.com":                 "google",
	"azure-apim.net":             "microsoft",
	"microsoft.com":              "microsoft",
	"vscode.dev":                 "microsoft",
	"amazonaws.cn":               "aws",
	"amazonaws.com":              "aws",
	"quicksight.aws.amazon.com":  "aws",
	"anysphere.cursor-mcp":       "cursor",
	"anysphere.cursor-retrieval": "cursor",
	"cursor.com":                 "cursor",
	"cursor.sh":                  "cursor",
	"atlassian.com":              "atlassian",
	"grok.com":                   "xai",
	"x.ai":                       "xai",
	"airia.ai":                   "airia",
	"gluegroups.com":             "glue",
}

// loopbackHosts are allowed for every native client, so they name no vendor.
var loopbackHosts = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
}

// vendorHeaders are headers only one vendor's platform sends.
var vendorHeaders = []struct{ header, vendor string }{
	{"X-Anthropic-Client", "anthropic"},
	{"X-Openai-Subject", "openai"},
	{"X-Openai-Session", "openai"},
	{"X-Integration-Connectors-Dapper-Trace-Id", "google"},
	{"X-Ms-Entra-Agent-Name", "microsoft"},
	{"X-Ms-Client-App-Id", "microsoft"},
	{"Glean-Tool-Name", "glean"},
	{"X-Glean-Chat-Session-Id", "glean"},
}

// userAgentVendors match the start of the lower-cased User-Agent. Prefixes,
// not substrings: "xClaude-Gateway" is not Anthropic.
var userAgentVendors = []struct{ prefix, vendor string }{
	{"claude-user", "anthropic"},
	{"claude-code/", "anthropic"},
	{"openai-mcp/", "openai"},
	{"codex-mcp-client/", "openai"},
	{"cursor/", "cursor"},
	{"copilot-", "microsoft"},
	{"grok-connectors-manager/", "xai"},
	{"opencode/", "opencode"},
}

// appNameVendors match the start of the lower-cased app name, which DCR
// clients set to "<product> <timestamp>".
var appNameVendors = []struct{ prefix, vendor string }{
	{"claude", "anthropic"},
	{"chatgpt", "openai"},
	{"codex", "openai"},
	{"cursor", "cursor"},
}

// clientVendor names the vendor of the client behind a request, and where that
// was read from. Both are empty when nothing matches. For telemetry only.
func clientVendor(app *auth.BearerApp, header http.Header) (vendor, source string) {
	if app != nil {
		for _, host := range app.RedirectHosts {
			if vendor := redirectVendor(host); vendor != "" {
				return vendor, vendorSourceRedirect
			}
		}
	}
	for _, h := range vendorHeaders {
		if header.Get(h.header) != "" {
			return h.vendor, vendorSourceHeader
		}
	}
	userAgent := strings.ToLower(header.Get("User-Agent"))
	for _, ua := range userAgentVendors {
		if strings.HasPrefix(userAgent, ua.prefix) {
			return ua.vendor, vendorSourceUserAgent
		}
	}
	if app != nil {
		name := strings.ToLower(app.Name)
		for _, n := range appNameVendors {
			if strings.HasPrefix(name, n.prefix) {
				return n.vendor, vendorSourceAppName
			}
		}
	}
	return "", ""
}

// redirectVendor looks the host up, then each domain above it.
func redirectVendor(host string) string {
	host = strings.ToLower(host)
	if loopbackHosts[host] {
		return ""
	}
	for {
		if vendor, ok := redirectVendors[host]; ok {
			return vendor
		}
		_, parent, found := strings.Cut(host, ".")
		if !found {
			return ""
		}
		host = parent
	}
}

// spanTagger is the part of *tracer.Span tagClient uses.
type spanTagger interface {
	SetTag(key string, value any)
}

// tagClient sets the OAuth app and vendor tags on the request span.
func tagClient(span spanTagger, app *auth.BearerApp, vendor, source string) {
	if app != nil {
		span.SetTag("oauth.client_id", app.ClientID)
		span.SetTag("oauth.app.id", app.ID)
		span.SetTag("oauth.app.name", truncate(app.Name, maxAppNameLength))
		span.SetTag("oauth.app.dcr", app.DCR)
	}
	if vendor != "" {
		span.SetTag("mcp.client.vendor", vendor)
		span.SetTag("mcp.client.vendor_source", source)
	}
}

func truncate(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
}
