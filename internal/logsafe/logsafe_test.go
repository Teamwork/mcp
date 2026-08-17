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

func TestBytesRedactsSmallFile(t *testing.T) {
	// A file is redacted whatever its size: a short secret must not survive just
	// because its base64 is short.
	content := base64.StdEncoding.EncodeToString([]byte("a short secret"))
	payload := `{"arguments":{"name":"secret.txt","data":"` + content + `"}}`

	got := logsafe.String(payload)

	if strings.Contains(got, content) {
		t.Errorf("expected the file content to be redacted, got %q", got)
	}
	if !strings.Contains(got, `"name":"secret.txt"`) {
		t.Errorf("expected the other parameters to survive, got %q", got)
	}
}

func TestBytesLeavesUnrelatedFieldsAlone(t *testing.T) {
	// The key is what's matched, so a field whose value is the word "data", and a
	// "content" field (page and comment bodies use it), are both left alone.
	payload := `{"arguments":{"search_term":"data","content":"the page body worth keeping"}}`

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

func TestBytesRedactsPresignedCredentials(t *testing.T) {
	// Scrubbing a body has to cover the upload URL a reservation answers with,
	// not only inline file content.
	payload := `{"ref":"tf_1a2b","url":"https://storage.example.com/tf_1a2b.md?` +
		`X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAEXAMPLE%2F20260817%2Feu-west-1&` +
		`X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeefcafe"}`

	got := logsafe.String(payload)

	for _, secret := range []string{"deadbeefcafe", "AKIAEXAMPLE"} {
		if strings.Contains(got, secret) {
			t.Errorf("expected %s to be redacted, got %q", secret, got)
		}
	}
	// What identifies the upload has to survive, or the log cannot be followed.
	if !strings.Contains(got, "storage.example.com/tf_1a2b.md") {
		t.Errorf("expected the URL itself to survive, got %q", got)
	}
	if !strings.Contains(got, `"ref":"tf_1a2b"`) {
		t.Errorf("expected the reference to survive, got %q", got)
	}
	if !strings.Contains(got, "X-Amz-SignedHeaders=host") {
		t.Errorf("expected the unsigned parameters to survive, got %q", got)
	}
}
