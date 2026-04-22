package config

import "testing"

// TestNewResourcesEnvPrefix pins that every setting this server owns is read
// under a configurable prefix. A second MCP server sharing a deployment
// configures itself with its own prefix; if any lookup stayed hardcoded to
// TW_MCP_ it would silently inherit this server's value.
func TestNewResourcesEnvPrefix(t *testing.T) {
	t.Setenv("TW_MCP_SERVER_ADDRESS", ":1111")
	t.Setenv("TW_MCP_API_URL", "https://default.example.com")
	t.Setenv("TW_MCP_PRO_SERVER_ADDRESS", ":2222")
	t.Setenv("TW_MCP_PRO_API_URL", "https://pro.example.com")

	t.Run("default prefix", func(t *testing.T) {
		resources := newResources(newOptions())
		if got := resources.Info.ServerAddress; got != ":1111" {
			t.Errorf("ServerAddress = %q, want %q", got, ":1111")
		}
		if got := resources.Info.APIURL; got != "https://default.example.com" {
			t.Errorf("APIURL = %q, want %q", got, "https://default.example.com")
		}
	})

	t.Run("custom prefix does not fall back to the default one", func(t *testing.T) {
		resources := newResources(newOptions(WithEnvPrefix("TW_MCP_PRO_")))
		if got := resources.Info.ServerAddress; got != ":2222" {
			t.Errorf("ServerAddress = %q, want %q", got, ":2222")
		}
		if got := resources.Info.APIURL; got != "https://pro.example.com" {
			t.Errorf("APIURL = %q, want %q", got, "https://pro.example.com")
		}
	})

	t.Run("an unset custom prefix uses the built-in defaults", func(t *testing.T) {
		resources := newResources(newOptions(WithEnvPrefix("TW_MCP_UNSET_")))
		if got := resources.Info.ServerAddress; got != ":8080" {
			t.Errorf("ServerAddress = %q, want the built-in default %q", got, ":8080")
		}
	})
}

// TestNewResourcesOTelEnvIsNotPrefixed guards the one group of variables that
// must stay bare: OTEL_SERVICE_NAME and friends are set by the OpenTelemetry
// specification, so prefixing them would leave tracing unconfigured.
func TestNewResourcesOTelEnvIsNotPrefixed(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-the-collector")
	t.Setenv("TW_MCP_PRO_OTEL_SERVICE_NAME", "should-be-ignored")

	resources := newResources(newOptions(WithEnvPrefix("TW_MCP_PRO_")))
	if got := resources.Info.OTel.Service; got != "from-the-collector" {
		t.Errorf("OTel.Service = %q, want %q", got, "from-the-collector")
	}
}

// TestNewResourcesServerIdentity covers the name and title reported in the
// initialize handshake. Clients key their connector UI off these, so a server
// built on this package must be able to set its own.
func TestNewResourcesServerIdentity(t *testing.T) {
	t.Run("defaults to this server", func(t *testing.T) {
		resources := newResources(newOptions())
		if got := resources.Info.Name; got != defaultServerName {
			t.Errorf("Name = %q, want %q", got, defaultServerName)
		}
		if got := resources.Info.Title; got != defaultServerTitle {
			t.Errorf("Title = %q, want %q", got, defaultServerTitle)
		}
	})

	t.Run("overridden by option", func(t *testing.T) {
		resources := newResources(newOptions(WithServerIdentity("Teamwork.com Pro", "Pro MCP")))
		if got := resources.Info.Name; got != "Teamwork.com Pro" {
			t.Errorf("Name = %q, want %q", got, "Teamwork.com Pro")
		}
		if got := resources.Info.Title; got != "Pro MCP" {
			t.Errorf("Title = %q, want %q", got, "Pro MCP")
		}
	})

	t.Run("environment wins over the option", func(t *testing.T) {
		t.Setenv("TW_MCP_NAME", "From The Environment")
		resources := newResources(newOptions(WithServerIdentity("Teamwork.com Pro", "Pro MCP")))
		if got := resources.Info.Name; got != "From The Environment" {
			t.Errorf("Name = %q, want %q", got, "From The Environment")
		}
	})
}

// TestNewResourcesProfilesAppendToTheURL pins the single-profile URL rule that
// the OAuth protected-resource metadata depends on.
func TestNewResourcesProfilesAppendToTheURL(t *testing.T) {
	t.Setenv("TW_MCP_URL", "https://mcp.example.com")

	tests := []struct {
		name     string
		profiles []string
		want     string
	}{
		{name: "no profile", profiles: nil, want: "https://mcp.example.com"},
		{name: "one profile", profiles: []string{"support"}, want: "https://mcp.example.com/support"},
		{name: "several profiles", profiles: []string{"support", "pm"}, want: "https://mcp.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources := newResources(newOptions(WithProfiles(tt.profiles...)))
			if got := resources.Info.MCPURL; got != tt.want {
				t.Errorf("MCPURL = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestNewResourcesDefaultMCPURL pins that a server can set its own resource
// identifier. It is the "resource" in the RFC 9728 metadata and the
// resource_metadata pointer in every 401 challenge, so a second server left on
// the default would tell clients to authorise against the first one.
func TestNewResourcesDefaultMCPURL(t *testing.T) {
	t.Run("defaults to this server", func(t *testing.T) {
		resources := newResources(newOptions())
		if got := resources.Info.MCPURL; got != defaultMCPURL {
			t.Errorf("MCPURL = %q, want %q", got, defaultMCPURL)
		}
	})

	t.Run("overridden by option", func(t *testing.T) {
		resources := newResources(newOptions(WithDefaultMCPURL("https://pro.example.com")))
		if got := resources.Info.MCPURL; got != "https://pro.example.com" {
			t.Errorf("MCPURL = %q, want %q", got, "https://pro.example.com")
		}
	})

	t.Run("environment wins over the option", func(t *testing.T) {
		t.Setenv("TW_MCP_URL", "https://from-the-environment.example.com/")
		resources := newResources(newOptions(WithDefaultMCPURL("https://pro.example.com")))
		if got := resources.Info.MCPURL; got != "https://from-the-environment.example.com" {
			t.Errorf("MCPURL = %q, want the environment value with its slash trimmed", got)
		}
	})
}
