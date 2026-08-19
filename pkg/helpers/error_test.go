package helpers_test

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/pkg/helpers"
	twapi "github.com/teamwork/twapi-go-sdk"
)

func TestHandleAPIError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		// wantText is the classification the tool result must carry; empty means
		// the error is expected to come back as a Go error instead.
		wantText string
	}{
		{
			name: "nil error",
		},
		{
			name:     "v3 typed server error",
			err:      newHTTPError(t, http.StatusBadGateway),
			wantText: "server error",
		},
		{
			name:     "v3 typed client error",
			err:      newHTTPError(t, http.StatusNotFound),
			wantText: "bad request",
		},
		{
			name:     "v3 typed redirect",
			err:      newHTTPError(t, http.StatusMovedPermanently),
			wantText: "unexpected HTTP status",
		},
		// The Desk SDK declares no error type: every non-2xx response leaves it
		// as a bare fmt.Errorf naming only the status, so these are the shapes
		// HandleAPIError has to read the status out of. See deskStatusCodePattern.
		{
			name:     "desk untyped client error",
			err:      errors.New("unexpected status code: 404"),
			wantText: "bad request",
		},
		{
			name:     "desk untyped client error with body",
			err:      errors.New(`unexpected status code: 422, body: {"errors":[{"detail":"nope"}]}`),
			wantText: "bad request",
		},
		{
			name:     "desk untyped server error",
			err:      errors.New("unexpected status code: 503"),
			wantText: "server error",
		},
		{
			name:     "desk file upload error",
			err:      errors.New("failed to upload file, status code: 500, status: 500 Internal Server Error, body: nope"),
			wantText: "server error",
		},
		{
			name:     "desk error reached through a wrap",
			err:      fmt.Errorf("get inbox: %w", errors.New("unexpected status code: 401")),
			wantText: "bad request",
		},
		{
			// Nothing to classify: a transport or programming failure stays a Go
			// error, the same as it does for twprojects.
			name: "error carrying no status",
			err:  errors.New("dial tcp: connection refused"),
		},
		{
			// A three-digit number that is not a status must not be mistaken for
			// one, or an unrelated failure would be reported as a bad request.
			name: "unrelated three-digit number",
			err:  errors.New("decode failed at offset 404"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := helpers.HandleAPIError(tt.err, "failed to do the thing")

			if tt.err == nil {
				if result != nil || err != nil {
					t.Fatalf("a nil error should yield nothing, got result=%v err=%v", result, err)
				}
				return
			}

			if tt.wantText == "" {
				if err == nil {
					t.Fatalf("expected a Go error, got result=%v", result)
				}
				if result != nil {
					t.Errorf("expected no tool result alongside the Go error, got %v", result)
				}
				if !strings.Contains(err.Error(), "failed to do the thing") {
					t.Errorf("Go error should carry the label, got %q", err.Error())
				}
				if !errors.Is(err, tt.err) {
					t.Errorf("Go error should wrap the cause, got %q", err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("an HTTP failure should be a tool result, not a Go error: %v", err)
			}
			if result == nil {
				t.Fatal("expected a tool result")
			}
			if !result.IsError {
				t.Error("tool result should be flagged as an error")
			}
			text := toolResultText(t, result)
			if !strings.Contains(text, tt.wantText) {
				t.Errorf("expected text to contain %q, got %q", tt.wantText, text)
			}
			// The underlying message has to survive, or the caller loses the
			// only detail the API gave it.
			if !strings.Contains(text, tt.err.Error()) {
				t.Errorf("expected text to carry the cause %q, got %q", tt.err.Error(), text)
			}
		})
	}
}

func toolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if len(result.Content) == 0 {
		t.Fatal("error tool result should carry content the model can read")
	}
	textContent, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", result.Content[0])
	}
	return textContent.Text
}

// newHTTPError builds the twapi.HTTPError that the v3 SDK returns, so the
// twprojects path is exercised alongside the Desk one.
func newHTTPError(t *testing.T, statusCode int) error {
	t.Helper()

	return twapi.NewHTTPError(&http.Response{
		StatusCode: statusCode,
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		Body:       http.NoBody,
	}, "request failed")
}
