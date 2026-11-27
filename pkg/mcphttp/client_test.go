package mcphttp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/pkg/auth"
)

func TestClientVendor(t *testing.T) {
	tests := []struct {
		name       string
		app        *auth.BearerApp
		header     map[string]string
		wantVendor string
		wantSource string
	}{
		{
			name:       "redirect host",
			app:        &auth.BearerApp{RedirectHosts: []string{"claude.ai"}},
			wantVendor: "anthropic",
			wantSource: vendorSourceRedirect,
		},
		{
			name:       "redirect subdomain",
			app:        &auth.BearerApp{RedirectHosts: []string{"global.consent.azure-apim.net"}},
			wantVendor: "microsoft",
			wantSource: vendorSourceRedirect,
		},
		{
			name:       "custom scheme host",
			app:        &auth.BearerApp{RedirectHosts: []string{"anysphere.cursor-mcp"}},
			wantVendor: "cursor",
			wantSource: vendorSourceRedirect,
		},
		{
			name:       "redirect wins over what the client says",
			app:        &auth.BearerApp{Name: "ChatGPT 1", RedirectHosts: []string{"claude.ai"}},
			header:     map[string]string{"User-Agent": "openai-mcp/1.0.0"},
			wantVendor: "anthropic",
			wantSource: vendorSourceRedirect,
		},
		{
			name:       "loopback redirect falls back to the header",
			app:        &auth.BearerApp{RedirectHosts: []string{"localhost", "127.0.0.1"}},
			header:     map[string]string{"X-Anthropic-Client": "x"},
			wantVendor: "anthropic",
			wantSource: vendorSourceHeader,
		},
		{
			name:       "user agent",
			app:        &auth.BearerApp{RedirectHosts: []string{"localhost"}},
			header:     map[string]string{"User-Agent": "claude-code/2.1.332 (cli)"},
			wantVendor: "anthropic",
			wantSource: vendorSourceUserAgent,
		},
		{
			name:       "user agent without an app",
			header:     map[string]string{"User-Agent": "openai-mcp/1.0.0 (Codex)"},
			wantVendor: "openai",
			wantSource: vendorSourceUserAgent,
		},
		{
			name:   "user agent matches on the prefix only",
			header: map[string]string{"User-Agent": "xClaude-Gateway/0.10.328"},
		},
		{
			name:       "app name",
			app:        &auth.BearerApp{Name: "ChatGPT 1791910533.694", RedirectHosts: []string{"localhost"}},
			header:     map[string]string{"User-Agent": "node"},
			wantVendor: "openai",
			wantSource: vendorSourceAppName,
		},
		{
			name: "unknown domain is not matched by suffix text",
			app:  &auth.BearerApp{Name: "My app", RedirectHosts: []string{"evilx.ai", "notclaude.ai"}},
		},
		{
			name:   "nothing to go on",
			header: map[string]string{"User-Agent": "python-httpx/0.28.1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{}
			for k, v := range tt.header {
				header.Set(k, v)
			}
			vendor, source := clientVendor(tt.app, header)
			if vendor != tt.wantVendor || source != tt.wantSource {
				t.Errorf("got (%q, %q), want (%q, %q)", vendor, source, tt.wantVendor, tt.wantSource)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("got %q", got)
	}
	long := strings.Repeat("é", maxAppNameLength+5)
	if got := truncate(long, maxAppNameLength); got != strings.Repeat("é", maxAppNameLength) {
		t.Errorf("got %d runes, want %d", len([]rune(got)), maxAppNameLength)
	}
}

type fakeSpan map[string]any

func (f fakeSpan) SetTag(key string, value any) { f[key] = value }

func TestTagClient(t *testing.T) {
	longName := "Claude " + strings.Repeat("x", 200)
	span := fakeSpan{}
	tagClient(span, &auth.BearerApp{ID: 42, ClientID: "client-42", Name: longName, DCR: true},
		"anthropic", vendorSourceRedirect)

	want := fakeSpan{
		"oauth.client_id":          "client-42",
		"oauth.app.id":             int64(42),
		"oauth.app.name":           longName[:maxAppNameLength],
		"oauth.app.dcr":            true,
		"mcp.client.vendor":        "anthropic",
		"mcp.client.vendor_source": vendorSourceRedirect,
	}
	for key, value := range want {
		if span[key] != value {
			t.Errorf("tag %s = %#v, want %#v", key, span[key], value)
		}
	}
}

func TestTagClientWithoutAppOrVendor(t *testing.T) {
	span := fakeSpan{}
	tagClient(span, nil, "", "")
	if len(span) != 0 {
		t.Errorf("got tags %v, want none", span)
	}
}
