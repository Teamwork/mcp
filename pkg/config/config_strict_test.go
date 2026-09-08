package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamwork/mcp/pkg/request"
)

// TestWantsStrictSchemas pins the trigger for the OpenAI strict-mode variant.
// The User-Agent is the only OpenAI signal on tools/list; anything
// unrecognised must fall back to the published schema.
func TestWantsStrictSchemas(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		want      bool
	}{
		{name: "openai tools/list", userAgent: "openai-mcp/1.0.0", want: true},
		{name: "openai codex", userAgent: "openai-mcp/1.0.0 (Codex)", want: true},
		{name: "case is ignored", userAgent: "OpenAI-MCP/1.0.0", want: true},
		{name: "a future version still matches", userAgent: "openai-mcp/2.5.0", want: true},
		{name: "claude", userAgent: "Claude-User", want: false},
		{name: "claude code", userAgent: "claude-code/2.1.252 (sdk-cli)", want: false},
		{name: "gemini enterprise", userAgent: "python-httpx/0.27.0", want: false},
		{name: "cline", userAgent: "undici", want: false},
		{name: "no user agent", userAgent: "", want: false},
		// Matching "openai" alone would hand strict schemas to anything
		// mentioning the vendor.
		{name: "an unrelated openai agent", userAgent: "my-openai-agent/1.0", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.userAgent != "" {
				r.Header.Set("User-Agent", tt.userAgent)
			}
			ctx := request.WithInfo(r.Context(), request.NewInfo(r))
			if got := wantsStrictSchemas(ctx); got != tt.want {
				t.Errorf("wantsStrictSchemas(%q) = %v, want %v", tt.userAgent, got, tt.want)
			}
		})
	}
}

// TestWantsStrictSchemasWithoutRequestInfo covers STDIO, which attaches no
// request.Info.
func TestWantsStrictSchemasWithoutRequestInfo(t *testing.T) {
	if wantsStrictSchemas(context.Background()) {
		t.Error("a context with no request info must not select strict schemas")
	}
}
