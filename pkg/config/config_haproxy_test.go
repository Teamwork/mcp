package config

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teamwork/mcp/pkg/twctx"
)

// captureTransport records the URL a request carried once the engine's
// middlewares are done with it.
type captureTransport struct {
	seen string
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.seen = r.URL.String()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func TestHAProxyLeavesPresignedURLsAlone(t *testing.T) {
	// A pre-signed upload leaves for storage, and its signature covers the host,
	// so rerouting it through HAProxy would send the file to the wrong server
	// with a signature that cannot match.
	const presigned = "https://storage.example.com/tf_1a2b.md?" +
		"X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeefcafe"

	t.Setenv("TW_MCP_HAPROXY_URL", "http://haproxy.internal:8080")

	resources, closer := Load(io.Discard)
	defer closer()

	capture := new(captureTransport)
	resources.teamworkHTTPClient.Transport = capture

	for _, tt := range []struct {
		name string
		url  string
		want string
	}{{
		name: "api request is rerouted",
		url:  "https://example.com/projects/api/v3/tasks.json",
		want: "http://haproxy.internal:8080/projects/api/v3/tasks.json",
	}, {
		name: "presigned upload is not",
		url:  presigned,
		want: presigned,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatalf("failed to build the request: %v", err)
			}
			resp, err := resources.teamworkEngine.Do(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			_ = resp.Body.Close()

			if capture.seen != tt.want {
				t.Errorf("expected %s, got %s", tt.want, capture.seen)
			}
		})
	}
}

func TestHAProxyKeepsTLSVerificationForOtherHosts(t *testing.T) {
	// Both servers present a certificate nobody trusts. Only the one standing
	// in for HAProxy may be reached without verification.
	haproxy, public := newUntrustedServer(t), newUntrustedServer(t)

	t.Setenv("TW_MCP_HAPROXY_URL", haproxy.URL)

	resources, closer := Load(io.Discard)
	defer closer()

	// The customer host is unreachable here; the engine reroutes it anyway.
	const apiURL = "https://example.com/projects/api/v3/tasks.json"

	for _, tt := range []struct {
		name    string
		do      func(*http.Request) (*http.Response, error)
		url     string
		ctx     func(context.Context) context.Context
		wantErr bool
	}{{
		name: "engine request rerouted to haproxy",
		do:   resources.teamworkEngine.Do,
		url:  apiURL,
	}, {
		name: "cross-region engine request",
		do:   resources.teamworkEngine.Do,
		url:  public.URL,
		ctx: func(ctx context.Context) context.Context {
			return twctx.WithCrossRegion(ctx, true)
		},
		wantErr: true,
	}, {
		// Desk, Spaces and token validation use the client directly.
		name:    "client request to another host",
		do:      resources.teamworkHTTPClient.Do,
		url:     public.URL,
		wantErr: true,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.ctx != nil {
				ctx = tt.ctx(ctx)
			}
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatalf("failed to build the request: %v", err)
			}
			resp, err := tt.do(request)
			if resp != nil {
				_ = resp.Body.Close()
			}

			_, certErr := errors.AsType[*tls.CertificateVerificationError](err)
			switch {
			case tt.wantErr && !certErr:
				t.Errorf("expected a certificate verification error, got %v", err)
			case !tt.wantErr && err != nil:
				t.Errorf("expected the request to succeed, got %v", err)
			}
		})
	}
}

// newUntrustedServer starts a TLS server with a self-signed certificate, which
// no client verifying certificates accepts.
func newUntrustedServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	// the rejected handshakes are expected; keep them out of the test output
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}
