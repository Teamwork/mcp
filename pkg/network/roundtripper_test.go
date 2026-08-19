package network_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/teamwork/mcp/pkg/network"
)

// presignedUploadURL carries the signature parameters a real pre-signed URL
// does, which is what marks the request as going to storage.
const presignedUploadURL = "https://storage.example.com/tf_1a2b.md?" +
	"X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAEXAMPLE&" +
	"X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeefcafe"

// stubTransport answers every request with the given response and keeps the last
// request it saw, so a test can check the round tripper left it intact.
type stubTransport struct {
	response *http.Response
	seen     *http.Request
}

func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.seen = r
	if s.response != nil {
		return s.response, nil
	}
	return newResponse("application/json", ""), nil
}

func newResponse(contentType, body string) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// logging returns a round tripper writing to a buffer the caller can read.
func logging(base http.RoundTripper) (*network.LoggingRoundTripper, *bytes.Buffer) {
	var logged bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logged, nil))
	return network.NewLoggingRoundTripper(logger, base), &logged
}

func TestRoundTripElidesPresignedUpload(t *testing.T) {
	// The upload's content type is the file's own, so a markdown or CSV
	// attachment looks textual. Only the signature in the URL says otherwise.
	contents := "# Plan\nsomething confidential\n"
	request, err := http.NewRequest(http.MethodPut, presignedUploadURL, strings.NewReader(contents))
	if err != nil {
		t.Fatalf("failed to build the request: %v", err)
	}
	request.Header.Set("Content-Type", "text/markdown")

	base := &stubTransport{}
	roundTripper, logged := logging(base)
	if _, err := roundTripper.RoundTrip(request); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	output := logged.String()
	if strings.Contains(output, "confidential") {
		t.Errorf("expected the file contents to be elided, got %q", output)
	}
	if !strings.Contains(output, "elided") {
		t.Errorf("expected the placeholder to say what happened, got %q", output)
	}
	if strings.Contains(output, "deadbeefcafe") {
		t.Errorf("expected the signature to be redacted, got %q", output)
	}

	// Eliding is a logging decision: the body still has to reach storage.
	body, err := io.ReadAll(base.seen.Body)
	if err != nil {
		t.Fatalf("failed to read the forwarded body: %v", err)
	}
	if string(body) != contents {
		t.Errorf("expected the contents to be forwarded, got %q", string(body))
	}
}

func TestRoundTripRedactsPresignedURLInResponse(t *testing.T) {
	// The reservation is an ordinary API call, so its body is logged; the upload
	// URL it answers with is the credential that must not be.
	request, err := http.NewRequest(http.MethodGet,
		"https://example.com/projects/api/v1/pendingfiles/presignedurl.json?fileName=plan.md&fileSize=7", nil)
	if err != nil {
		t.Fatalf("failed to build the request: %v", err)
	}

	response := newResponse("application/json", `{"ref":"tf_1a2b","url":"`+presignedUploadURL+`"}`)
	roundTripper, logged := logging(&stubTransport{response: response})
	if _, err := roundTripper.RoundTrip(request); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	output := logged.String()
	if strings.Contains(output, "deadbeefcafe") || strings.Contains(output, "AKIAEXAMPLE") {
		t.Errorf("expected the credentials to be redacted, got %q", output)
	}
	if !strings.Contains(output, "tf_1a2b") {
		t.Errorf("expected the reference to survive, got %q", output)
	}
}

func TestRoundTripStillLogsAPIBodies(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost,
		"https://example.com/projects/api/v3/tasks.json", strings.NewReader(`{"task":{"name":"example"}}`))
	if err != nil {
		t.Fatalf("failed to build the request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	roundTripper, logged := logging(&stubTransport{response: newResponse("application/json", `{"task":{"id":1}}`)})
	if _, err := roundTripper.RoundTrip(request); err != nil {
		t.Fatalf("round trip failed: %v", err)
	}

	output := logged.String()
	for _, want := range []string{"example", `id`} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in the log, got %q", want, output)
		}
	}
}

func TestPresignedSplitTransportRoutesByURL(t *testing.T) {
	base, presigned := &stubTransport{}, &stubTransport{}
	transport := &network.PresignedSplitTransport{Base: base, Presigned: presigned}

	for _, tt := range []struct {
		name string
		url  string
		want *stubTransport
	}{{
		name: "api request",
		url:  "https://example.com/projects/api/v3/tasks.json",
		want: base,
	}, {
		name: "presigned upload",
		url:  presignedUploadURL,
		want: presigned,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			base.seen, presigned.seen = nil, nil

			request, err := http.NewRequest(http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatalf("failed to build the request: %v", err)
			}
			if _, err := transport.RoundTrip(request); err != nil {
				t.Fatalf("round trip failed: %v", err)
			}
			if tt.want.seen == nil {
				t.Error("expected the request to reach the other transport")
			}
		})
	}
}
