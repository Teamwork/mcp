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

// presignedUploadURL is the URL the reservation step hands back. It carries the
// AWS signature parameters a real one does, so the requests the upload makes are
// indistinguishable from production to everything downstream.
const presignedUploadURL = "https://storage.example.com/tf_1a2b.md?" +
	"X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAEXAMPLE&" +
	"X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeef"

// fileCreateMock wires the two steps of an upload: the reservation, which
// answers with a reference and the URL to send the contents to, and the upload
// itself, which any other request falls through to.
func fileCreateMock(t *testing.T) (*mcp.Server, *[]testutil.ProjectsRecordedRequest) {
	t.Helper()

	return mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Match:  "pendingfiles/presignedurl",
		Method: http.MethodGet,
		Status: http.StatusOK,
		Body:   []byte(`{"ref":"tf_1a2b","url":"` + presignedUploadURL + `"}`),
	}}, http.StatusOK, nil)
}

func TestFileCreate(t *testing.T) {
	mcpServer, recorded := fileCreateMock(t)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), map[string]any{
		"name": "plan.md",
		"data": base64.StdEncoding.EncodeToString([]byte("# Plan\n")),
	})

	// Reserve first, then upload. The file never travels to the API itself, so
	// the second request going anywhere but the pre-signed URL is a failure.
	if len(*recorded) != 2 {
		t.Fatalf("expected a reservation and an upload, got %d requests", len(*recorded))
	}

	reserve, upload := (*recorded)[0], (*recorded)[1]
	if reserve.Method != http.MethodGet {
		t.Errorf("expected the reservation to be a GET, got %s", reserve.Method)
	}
	if got := reserve.URL.Query().Get("fileName"); got != "plan.md" {
		t.Errorf("expected the reservation to name plan.md, got %q", got)
	}
	if got := reserve.URL.Query().Get("fileSize"); got != "7" {
		t.Errorf("expected the reservation to declare 7 bytes, got %q", got)
	}

	if upload.Method != http.MethodPut {
		t.Errorf("expected the upload to be a PUT, got %s", upload.Method)
	}
	if got := upload.URL.String(); got != presignedUploadURL {
		t.Errorf("expected the upload to go to the pre-signed URL, got %q", got)
	}
	if got := string(upload.Body); got != "# Plan\n" {
		t.Errorf("expected the decoded contents in the upload, got %q", got)
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
			mcpServer, recorded := fileCreateMock(t)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), map[string]any{
				"name": tt.input,
				"data": base64.StdEncoding.EncodeToString([]byte("contents")),
			})

			// The name reaches the API in the reservation's query string; the
			// upload only ever sees the reference.
			if len(*recorded) == 0 {
				t.Fatal("expected the reservation to be sent")
			}
			if got := (*recorded)[0].URL.Query().Get("fileName"); got != tt.want {
				t.Errorf("expected fileName %q, got %q", tt.want, got)
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
			mcpServer, _ := fileCreateMock(t)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), tt.arguments,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					assertErrorResultContains(t, result, tt.wantIn)
				}),
			)
		})
	}
}

func TestFileCreateReportsUploadFailureAsToolResult(t *testing.T) {
	// The upload is a second request, to a service that answers for itself. A
	// rejection there has to reach the caller as a readable tool result, the same
	// as one from the API.
	for _, tt := range []struct {
		name string
		// wantIn also says which step failed, so a message that stops at "403"
		// without naming the step does not pass.
		wantIn string
		routes []testutil.ProjectsMockRoute
	}{{
		name:   "reservation rejected",
		wantIn: "failed to reserve pending file",
		routes: []testutil.ProjectsMockRoute{{
			Match:  "pendingfiles/presignedurl",
			Method: http.MethodGet,
			Status: http.StatusForbidden,
			Body:   []byte(`{"error":"forbidden"}`),
		}},
	}, {
		name:   "storage rejected",
		wantIn: "failed to upload pending file",
		routes: []testutil.ProjectsMockRoute{{
			Match:  "pendingfiles/presignedurl",
			Method: http.MethodGet,
			Status: http.StatusOK,
			Body:   []byte(`{"ref":"tf_1a2b","url":"` + presignedUploadURL + `"}`),
		}, {
			Match:  "tf_1a2b",
			Method: http.MethodPut,
			Status: http.StatusForbidden,
			Body:   []byte(`<Error><Code>AccessDenied</Code></Error>`),
		}},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			mcpServer := testutil.ProjectsMCPServerRoutedMock(t, tt.routes, http.StatusOK, nil)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileCreate.String(), map[string]any{
				"name": "plan.md",
				"data": base64.StdEncoding.EncodeToString([]byte("# Plan\n")),
			},
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
	// reaches the API. Reserving space for it would already be too late.
	mcpServer, recorded := fileCreateMock(t)

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

	if len(*recorded) != 0 {
		t.Errorf("expected no request to be sent, got %d", len(*recorded))
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
