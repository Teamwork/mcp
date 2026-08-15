package logsafe_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/logsafe"
)

func TestBytesRedactsFileContent(t *testing.T) {
	content := base64.StdEncoding.EncodeToString(make([]byte, 4096))
	payload := `{"name":"twprojects-create_file","arguments":{"data":"` + content + `","name":"plan.md"}}`

	got := logsafe.String(payload)

	if strings.Contains(got, content[:256]) {
		t.Error("expected the file content to be redacted")
	}
	// The point of redacting rather than truncating is that the rest of the
	// record survives, so the log still says which tool ran and with what.
	if !strings.Contains(got, "twprojects-create_file") {
		t.Errorf("expected the tool name to survive redaction, got %q", got)
	}
	if !strings.Contains(got, `"name":"plan.md"`) {
		t.Errorf("expected the other parameters to survive redaction, got %q", got)
	}
	if !strings.Contains(got, "redacted") {
		t.Errorf("expected the placeholder to say what happened, got %q", got)
	}
}

func TestBytesLeavesShortValuesAlone(t *testing.T) {
	// A short value under one of the content keys is readable and worth keeping,
	// and a search term that happens to be the word "data" is not a key at all.
	payload := `{"arguments":{"data":"plan","search_term":"data"}}`

	if got := logsafe.String(payload); got != payload {
		t.Errorf("expected the payload unchanged, got %q", got)
	}
}

func TestBytesTruncatesOversizePayload(t *testing.T) {
	// A payload that carries no recognisable content key still must not fill the
	// log.
	payload := `{"body":"` + strings.Repeat("a", logsafe.MaxLoggedBytes*2) + `"}`

	got := logsafe.String(payload)

	if len(got) > logsafe.MaxLoggedBytes+len("...[truncated]") {
		t.Errorf("expected the payload capped, got %d bytes", len(got))
	}
	if !strings.HasSuffix(got, "...[truncated]") {
		t.Error("expected the payload to say it was truncated")
	}
}

func TestBytesHandlesEmpty(t *testing.T) {
	if got := logsafe.Bytes(nil); got != nil {
		t.Errorf("expected nil, got %q", got)
	}
	if got := logsafe.String(""); got != "" {
		t.Errorf("expected an empty string, got %q", got)
	}
}

func TestIsTextualContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"", true}, // no declared type: the bodies we send without one are JSON
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"text/plain", true},
		{"application/vnd.api+json", true},
		{"application/x-www-form-urlencoded", true},
		{"multipart/form-data; boundary=abc", false}, // a file upload
		{"application/octet-stream", false},
		{"image/png", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := logsafe.IsTextualContentType(tt.contentType); got != tt.want {
				t.Errorf("expected %v for %q, got %v", tt.want, tt.contentType, got)
			}
		})
	}
}

// TestBytesStaysLinear guards against a future rewrite of the pattern
// introducing backtracking. A multi-megabyte body runs through this on every
// upload.
func TestBytesStaysLinear(t *testing.T) {
	payload := []byte(`{"data":"` + strings.Repeat("A", 8<<20) + `"}`)

	got := logsafe.Bytes(payload)

	if len(got) > logsafe.MaxLoggedBytes+len("...[truncated]") {
		t.Errorf("expected the payload capped, got %d bytes", len(got))
	}
}
