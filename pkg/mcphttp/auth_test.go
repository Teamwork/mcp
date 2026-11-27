package mcphttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teamwork/mcp/pkg/auth"
	"github.com/teamwork/mcp/pkg/config"
	"github.com/teamwork/mcp/pkg/request"
	"github.com/teamwork/mcp/pkg/twctx"
)

// TestAuthMarksRestrictedInstallations pins that a request is flagged for
// restricted data exactly when its token belongs to a configured installation.
func TestAuthMarksRestrictedInstallations(t *testing.T) {
	userInfo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user_id":12345,"installation_id":777,"url":"https://test.teamwork.com"}`))
	}))
	defer userInfo.Close()

	tests := []struct {
		name       string
		restricted []int64
		want       bool
	}{
		{name: "listed installation", restricted: []int64{1, 777}, want: true},
		{name: "unlisted installation", restricted: []int64{1}, want: false},
		{name: "no list", restricted: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resources config.Resources
			resources.Info.RestrictedInstallationIDs = tt.restricted
			validator := auth.NewValidator(userInfo.Client(), userInfo.URL, resources.Logger())

			var got, reached bool
			handler := Auth(resources, validator, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				reached = true
				got = twctx.IsRestrictedData(r.Context())
			}))
			r := httptest.NewRequest(http.MethodPost, "/", nil)
			r.Header.Set("Authorization", "Bearer token")
			handler.ServeHTTP(httptest.NewRecorder(), r)

			if !reached {
				t.Fatal("the request did not reach the handler")
			}
			if got != tt.want {
				t.Errorf("IsRestrictedData = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAuthRecordsTheOAuthClient pins that the app the userinfo endpoint
// reports is decoded and reaches the request info the log lines read.
func TestAuthRecordsTheOAuthClient(t *testing.T) {
	userInfo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"user_id":12345,"installation_id":777,"url":"https://test.teamwork.com",` +
			`"meta":{"app":{"id":42,"clientId":"client-42","name":"Claude 1",` +
			`"dcr":true,"redirectHosts":["claude.ai"]}}}`))
	}))
	defer userInfo.Close()

	var resources config.Resources
	validator := auth.NewValidator(userInfo.Client(), userInfo.URL, resources.Logger())

	info := request.NewInfo(httptest.NewRequest(http.MethodPost, "/", nil))
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Authorization", "Bearer token")
	r = r.WithContext(request.WithInfo(r.Context(), info))

	Auth(resources, validator, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(httptest.NewRecorder(), r)

	if info.ClientID() != "client-42" || info.ClientVendor() != "anthropic" {
		t.Errorf("request info = (%q, %q), want (client-42, anthropic)", info.ClientID(), info.ClientVendor())
	}
}
