package config

import (
	"io"
	"net/http"
	"strings"
	"testing"
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
