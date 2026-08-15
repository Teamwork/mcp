package twprojects_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

func TestFileCreate(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"pendingFile":{"ref":"tf_1a2b"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), map[string]any{
		"name": "plan.md",
		"data": base64.StdEncoding.EncodeToString([]byte("# Plan\n")),
	})

	// The upload is multipart rather than JSON, so the assertion is on the part
	// itself: the file has to arrive under the name the API expects, carrying the
	// decoded bytes rather than the base64 the caller sent.
	body := string(*requestBody)
	if !strings.Contains(body, `name="file"`) {
		t.Errorf("expected a form part named file, got %q", body)
	}
	if !strings.Contains(body, `filename="plan.md"`) {
		t.Errorf("expected the file name in the body, got %q", body)
	}
	if !strings.Contains(body, "# Plan") {
		t.Errorf("expected the decoded contents in the body, got %q", body)
	}
	if strings.Contains(body, base64.StdEncoding.EncodeToString([]byte("# Plan\n"))) {
		t.Errorf("expected the contents to be decoded before upload, got %q", body)
	}
}

func TestFileCreateSanitizesFileName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{{
		name:  "posix path",
		input: "docs/plans/plan.md",
		want:  "plan.md",
	}, {
		name:  "windows path",
		input: `C:\Users\someone\plan.md`,
		want:  "plan.md",
	}, {
		name:  "traversal",
		input: "../../etc/passwd",
		want:  "passwd",
	}, {
		name:  "control characters",
		input: "pl\nan\r.md",
		want:  "plan.md",
	}, {
		name:  "surrounding whitespace",
		input: "  plan.md  ",
		want:  "plan.md",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated,
				[]byte(`{"pendingFile":{"ref":"tf_1a2b"}}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), map[string]any{
				"name": tt.input,
				"data": base64.StdEncoding.EncodeToString([]byte("contents")),
			})

			// The leading filename=" anchors the check to the multipart part's
			// name, so an un-sanitized path could not match by appearing elsewhere.
			want := `filename="` + tt.want + `"`
			if !strings.Contains(string(*requestBody), want) {
				t.Errorf("expected the body to carry %s, got %q", want, string(*requestBody))
			}
		})
	}
}

func TestFileCreateRejectsBadInput(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		wantIn    string
	}{{
		name:      "malformed base64",
		arguments: map[string]any{"name": "plan.md", "data": "not base64!!!"},
		wantIn:    "base64",
	}, {
		name:      "empty content",
		arguments: map[string]any{"name": "plan.md", "data": ""},
		wantIn:    "zero bytes",
	}, {
		name:      "name is only a path",
		arguments: map[string]any{"name": "../", "data": base64.StdEncoding.EncodeToString([]byte("x"))},
		wantIn:    "file name",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A caller mistake has to come back as a tool result it can correct,
			// not as a transport error.
			mcpServer := mcpServerMock(t, http.StatusCreated, []byte(`{"pendingFile":{"ref":"tf_1a2b"}}`))
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), tt.arguments,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					assertErrorResultContains(t, result, tt.wantIn)
				}),
			)
		})
	}
}

func TestFileCreateRejectsOversizeBeforeUploading(t *testing.T) {
	// The size check has to run before the upload, so an oversize payload never
	// reaches the API.
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"pendingFile":{"ref":"tf_1a2b"}}`))

	oversize := base64.StdEncoding.EncodeToString(make([]byte, (5<<20)+1))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), map[string]any{
		"name": "big.bin",
		"data": oversize,
	},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			assertErrorResultContains(t, result, "too large")
		}),
	)

	if len(*requestBody) != 0 {
		t.Errorf("expected no request to be sent, got a body of %d bytes", len(*requestBody))
	}
}

// assertErrorResultContains checks that a tool call came back as an error result
// the caller can read, rather than as a transport error.
func assertErrorResultContains(t *testing.T, result mcp.Result, want string) {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if !toolResult.IsError {
		t.Fatalf("expected an error tool result, got %+v", toolResult)
	}
	if len(toolResult.Content) == 0 {
		t.Fatal("error tool result should carry content the model can read")
	}
	textContent, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}
	if !strings.Contains(strings.ToLower(textContent.Text), want) {
		t.Errorf("expected the message to mention %q, got %q", want, textContent.Text)
	}
}

func TestTaskCreateSendsAttachments(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"task":{"id":123}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), map[string]any{
		"name":            "example",
		"tasklist_id":     float64(777),
		"attachment_refs": []any{"tf_A", "tf_B"},
	})

	// attachments is a sibling of task in the request body, not one of its
	// attributes, so the nesting is what this pins.
	var payload struct {
		Attachments struct {
			PendingFiles []struct {
				Reference string `json:"reference"`
			} `json:"pendingFiles"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}
	if len(payload.Attachments.PendingFiles) != 2 {
		t.Fatalf("expected two pending files, got body %q", string(*requestBody))
	}
	if payload.Attachments.PendingFiles[0].Reference != "tf_A" ||
		payload.Attachments.PendingFiles[1].Reference != "tf_B" {
		t.Errorf("unexpected references in body %q", string(*requestBody))
	}
}

func TestTaskUpdateSendsAttachments(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusOK, []byte(`{}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskUpdate.String(), map[string]any{
		"id":              float64(123),
		"attachment_refs": []any{"tf_A"},
	})

	if !strings.Contains(string(*requestBody), `"reference":"tf_A"`) {
		t.Errorf("expected the reference in the body, got %q", string(*requestBody))
	}
}

func TestTaskAttachmentsOmittedWhenNotRequested(t *testing.T) {
	// An empty attachments object would be a payload change for every caller
	// that attaches nothing, so the key has to stay absent.
	for _, tt := range []struct {
		name      string
		method    string
		status    int
		arguments map[string]any
	}{{
		name:      "create",
		method:    twprojects.MethodTaskCreate.String(),
		status:    http.StatusCreated,
		arguments: map[string]any{"name": "example", "tasklist_id": float64(777)},
	}, {
		name:      "update",
		method:    twprojects.MethodTaskUpdate.String(),
		status:    http.StatusOK,
		arguments: map[string]any{"id": float64(123), "name": "example"},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer, requestBody := mcpServerMockWithRequestBody(t, tt.status,
				[]byte(`{"task":{"id":123}}`))
			testutil.ExecuteToolRequest(t, mcpServer, tt.method, tt.arguments)

			var payload map[string]any
			if err := json.Unmarshal(*requestBody, &payload); err != nil {
				t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
			}
			if _, ok := payload["attachments"]; ok {
				t.Errorf("expected no attachments key, got body %q", string(*requestBody))
			}
			if _, ok := payload["attachmentOptions"]; ok {
				t.Errorf("expected no attachmentOptions key, got body %q", string(*requestBody))
			}
		})
	}
}

func TestCommentCreateSendsAttachments(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated, []byte(`{"id":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreate.String(), map[string]any{
		"object":          map[string]any{"type": "tasks", "id": float64(777)},
		"body":            "see attached",
		"attachment_refs": []any{"tf_A", "tf_B"},
	})

	var payload struct {
		Comment struct {
			PendingFileAttachments []string `json:"pendingFileAttachments"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}
	if len(payload.Comment.PendingFileAttachments) != 2 ||
		payload.Comment.PendingFileAttachments[0] != "tf_A" ||
		payload.Comment.PendingFileAttachments[1] != "tf_B" {
		t.Errorf("unexpected references in body %q", string(*requestBody))
	}
}

func TestMessageCreateSendsAttachments(t *testing.T) {
	mcpServer, requestBody := mcpServerMockWithRequestBody(t, http.StatusCreated,
		[]byte(`{"messageId":"123"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodMessageCreate.String(), map[string]any{
		"project_id":      float64(777),
		"title":           "example",
		"body":            "see attached",
		"attachment_refs": []any{"tf_A"},
	})

	var payload struct {
		Post struct {
			PendingFileAttachments []string `json:"pendingFileAttachments"`
		} `json:"post"`
	}
	if err := json.Unmarshal(*requestBody, &payload); err != nil {
		t.Fatalf("failed to decode request body %q: %v", string(*requestBody), err)
	}
	if len(payload.Post.PendingFileAttachments) != 1 || payload.Post.PendingFileAttachments[0] != "tf_A" {
		t.Errorf("unexpected references in body %q", string(*requestBody))
	}
}
