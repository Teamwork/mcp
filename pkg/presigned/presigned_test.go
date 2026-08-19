package presigned_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/teamwork/mcp/pkg/presigned"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{{
		name: "signature v4",
		raw:  "https://storage.example.com/tf_1a2b.md?X-Amz-Signature=deadbeef&X-Amz-SignedHeaders=host",
		want: true,
	}, {
		name: "signature v2",
		raw:  "https://storage.example.com/tf_1a2b.md?AWSAccessKeyId=AKIAEXAMPLE&Signature=deadbeef",
		want: true,
	}, {
		name: "api request",
		raw:  "https://example.com/projects/api/v3/tasks.json?pageSize=50",
		want: false,
	}, {
		// The step that asks for the URL is an ordinary API call, so it keeps
		// being rerouted and logged like one.
		name: "reservation request",
		raw:  "https://example.com/projects/api/v1/pendingfiles/presignedurl.json?fileName=plan.md&fileSize=7",
		want: false,
	}, {
		name: "no query",
		raw:  "https://example.com/projects/api/v3/tasks.json",
		want: false,
	}, {
		name: "empty parameter",
		raw:  "https://storage.example.com/tf_1a2b.md?X-Amz-Signature=",
		want: false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := url.Parse(tt.raw)
			if err != nil {
				t.Fatalf("failed to parse %q: %v", tt.raw, err)
			}
			if got := presigned.IsURL(parsed); got != tt.want {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestIsURLHandlesNil(t *testing.T) {
	if presigned.IsURL(nil) {
		t.Error("expected a nil URL not to be pre-signed")
	}
}

func TestRedactSignatures(t *testing.T) {
	payload := `{"ref":"tf_1a2b","url":"https://storage.example.com/tf_1a2b.md?` +
		`X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAEXAMPLE%2F20260817%2Feu-west-1&` +
		`X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeefcafe"}`

	got := string(presigned.RedactSignatures([]byte(payload)))

	for _, secret := range []string{"deadbeefcafe", "AKIAEXAMPLE"} {
		if strings.Contains(got, secret) {
			t.Errorf("expected %s to be redacted, got %q", secret, got)
		}
	}
	// What identifies the upload has to survive, or the log cannot be followed.
	for _, want := range []string{"storage.example.com/tf_1a2b.md", `"ref":"tf_1a2b"`, "X-Amz-SignedHeaders=host"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %s to survive, got %q", want, got)
		}
	}
}

func TestRedactSignaturesLeavesOtherPayloadsAlone(t *testing.T) {
	// The scan is skipped when no signature parameter is named, so an ordinary
	// body must come back byte for byte.
	payload := []byte(`{"task":{"name":"example","description":"a signature is not the same as Signature="}}`)

	if got := presigned.RedactSignatures(payload); string(got) != string(payload) {
		t.Errorf("expected the payload unchanged, got %q", string(got))
	}
}
