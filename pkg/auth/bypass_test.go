package auth

import (
	"testing"
)

func TestBypassMethod(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		// SEP-2575 stateless capability discovery, the replacement for the
		// "initialize" handshake. Clients probe it before holding a token.
		{method: "server/discover", want: true},

		// legacy spec revisions must keep working unchanged
		{method: "initialize", want: true},
		{method: "notifications/initialized", want: true},
		{method: "logging/setLevel", want: true},
		{method: "tools/list", want: true},
		{method: "resources/list", want: true},
		{method: "resources/templates/list", want: true},
		{method: "prompts/list", want: true},

		// everything else must stay authenticated
		{method: "tools/call", want: false},
		{method: "resources/read", want: false},
		{method: "prompts/get", want: false},
		{method: "server/discovery", want: false},
		{method: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			if got := BypassMethod(tt.method); got != tt.want {
				t.Errorf("BypassMethod(%q) = %t, want %t", tt.method, got, tt.want)
			}
		})
	}
}

func TestBypass(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    bool
		wantErr bool
	}{{
		name: "server/discover request",
		data: `{"jsonrpc":"2.0","id":1,"method":"server/discover"}`,
		want: true,
	}, {
		name: "initialize request",
		data: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`,
		want: true,
	}, {
		name:    "tools/call request",
		data:    `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"twprojects-get_task"}}`,
		want:    false,
		wantErr: true,
	}, {
		name:    "malformed payload",
		data:    `{not json`,
		want:    false,
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Bypass([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("Bypass() error = %v, wantErr %t", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Bypass() = %t, want %t", got, tt.want)
			}
		})
	}
}
