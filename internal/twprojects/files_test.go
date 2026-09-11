package twprojects_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
	pkgtestutil "github.com/teamwork/mcp/pkg/testutil"
	twapi "github.com/teamwork/twapi-go-sdk"
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

// uploadURLCreateMock answers the reservation and fails the test if anything
// else is sent: the point of the tool is that the contents never come here.
func uploadURLCreateMock(t *testing.T, presignedURL string) (*mcp.Server, *[]testutil.ProjectsRecordedRequest) {
	t.Helper()

	return mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Match:  "pendingfiles/presignedurl",
		Method: http.MethodGet,
		Status: http.StatusOK,
		Body:   []byte(`{"ref":"tf_1a2b","url":"` + presignedURL + `"}`),
	}}, http.StatusOK, nil)
}

// decodeToolJSON reads the JSON a tool handed back, so a test can assert on
// what the caller actually receives rather than only on what went to the wire.
func decodeToolJSON(t *testing.T, result mcp.Result) map[string]any {
	t.Helper()

	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}
	if toolResult.IsError {
		t.Fatalf("unexpected error result: %+v", toolResult)
	}
	if len(toolResult.Content) == 0 {
		t.Fatal("expected content in the tool result")
	}
	text, ok := toolResult.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content type: %T", toolResult.Content[0])
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(text.Text), &decoded); err != nil {
		t.Fatalf("failed to decode the tool result: %s", err)
	}
	return decoded
}

func TestUploadURLCreate(t *testing.T) {
	// Signed headers include the canned ACL, and the signature carries a
	// readable deadline, so the tool has both to report.
	const presignedURL = "https://storage.example.com/tf_1a2b.pdf?" +
		"X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAEXAMPLE&" +
		"X-Amz-Date=20260826T120000Z&X-Amz-Expires=600&" +
		"X-Amz-SignedHeaders=host%3Bx-amz-acl&X-Amz-Signature=deadbeef"

	mcpServer, recorded := uploadURLCreateMock(t, presignedURL)

	var result map[string]any
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUploadURLCreate.String(), map[string]any{
		"name": "contract.pdf",
		"size": 204800,
	},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, message mcp.Result) {
			t.Helper()
			result = decodeToolJSON(t, message)
		}),
	)

	// The whole point: one request, and it carries no file. A second request
	// would mean the contents came through this server after all.
	if len(*recorded) != 1 {
		t.Fatalf("expected only the reservation, got %d requests", len(*recorded))
	}
	reserve := (*recorded)[0]
	if reserve.Method != http.MethodGet {
		t.Errorf("expected the reservation to be a GET, got %s", reserve.Method)
	}
	if got := reserve.URL.Query().Get("fileName"); got != "contract.pdf" {
		t.Errorf("expected the reservation to name contract.pdf, got %q", got)
	}
	// The signature covers this number, so an upload of any other length fails.
	if got := reserve.URL.Query().Get("fileSize"); got != "204800" {
		t.Errorf("expected the reservation to declare 204800 bytes, got %q", got)
	}

	if got := result["reference"]; got != "tf_1a2b" {
		t.Errorf("expected the reference from the reservation, got %v", got)
	}
	if got := result["upload_url"]; got != presignedURL {
		t.Errorf("expected the pre-signed URL unchanged, got %v", got)
	}
	if got := result["method"]; got != http.MethodPut {
		t.Errorf("expected a PUT, got %v", got)
	}
	if got := result["expires_at"]; got != "2026-08-26T12:10:00Z" {
		t.Errorf("expected the deadline read from the signature, got %v", got)
	}

	headers, ok := result["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers in the result, got %T", result["headers"])
	}
	// Signed into this URL, so the caller must repeat it.
	if got := headers["X-Amz-Acl"]; got != "public-read" {
		t.Errorf("expected the canned ACL, got %v", got)
	}
	// A chunked body is rejected by the storage service, so the length has to
	// travel with the rest even though the signature does not cover it.
	if got := headers["Content-Length"]; got != "204800" {
		t.Errorf("expected the content length among the headers, got %v", got)
	}
	if headers["Content-Type"] == nil || headers["Content-Type"] == "" {
		t.Error("expected a content type among the headers")
	}
	// A second authorization mechanism is refused by the storage service.
	for key := range headers {
		if strings.EqualFold(key, "Authorization") {
			t.Error("expected no authorization among the headers")
		}
	}
}

// TestUploadURLCreateOmitsUnsignedACL is the other half of the ACL rule: which
// headers to send is decided by the URL, and repeating an unsigned x-amz-* one
// fails the upload just as surely as leaving a signed one out.
func TestUploadURLCreateOmitsUnsignedACL(t *testing.T) {
	const presignedURL = "https://storage.example.com/tf_1a2b.pdf?" +
		"X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeef"

	mcpServer, _ := uploadURLCreateMock(t, presignedURL)

	var result map[string]any
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUploadURLCreate.String(), map[string]any{
		"name": "contract.pdf",
		"size": 1024,
	},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, message mcp.Result) {
			t.Helper()
			result = decodeToolJSON(t, message)
		}),
	)

	headers, ok := result["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers in the result, got %T", result["headers"])
	}
	if got, present := headers["X-Amz-Acl"]; present {
		t.Errorf("expected no canned ACL when it is not signed, got %v", got)
	}
	// Nothing in the URL says when it lapses, and guessing would be worse than
	// staying quiet.
	if got, present := result["expires_at"]; present {
		t.Errorf("expected no deadline when the URL does not carry one, got %v", got)
	}
}

func TestUploadURLCreateSanitizesFileName(t *testing.T) {
	const presignedURL = "https://storage.example.com/tf_1a2b.pdf?" +
		"X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeef"

	mcpServer, recorded := uploadURLCreateMock(t, presignedURL)
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUploadURLCreate.String(), map[string]any{
		"name": `C:\Users\someone\contract.pdf`,
		"size": 1024,
	})

	if len(*recorded) == 0 {
		t.Fatal("expected the reservation to be sent")
	}
	if got := (*recorded)[0].URL.Query().Get("fileName"); got != "contract.pdf" {
		t.Errorf("expected fileName %q, got %q", "contract.pdf", got)
	}
}

func TestUploadURLCreateRejectsBadInput(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		want      string
	}{{
		// The schema's minimum catches these before the handler runs, which is
		// why the handler's own check is unreachable from a validating client
		// and kept anyway for one that skips validation.
		name:      "no size",
		arguments: map[string]any{"name": "contract.pdf", "size": 0},
		want:      "size",
	}, {
		name:      "negative size",
		arguments: map[string]any{"name": "contract.pdf", "size": -1},
		want:      "size",
	}, {
		name:      "name is a path",
		arguments: map[string]any{"name": "../..", "size": 1024},
		want:      "invalid name",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const presignedURL = "https://storage.example.com/tf_1a2b.pdf?" +
				"X-Amz-SignedHeaders=host&X-Amz-Signature=deadbeef"

			mcpServer, recorded := uploadURLCreateMock(t, presignedURL)
			testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodUploadURLCreate.String(), tt.arguments,
				testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
					t.Helper()
					assertErrorResultContains(t, result, tt.want)
				}),
			)

			// Bad input is caught here, so nothing is reserved for a file that
			// is never going to be uploaded.
			if len(*recorded) != 0 {
				t.Errorf("expected no request to be sent, got %d", len(*recorded))
			}
		})
	}
}

func TestProjectFileAdd(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{{
		Match:  "/projects/777/files",
		Method: http.MethodPost,
		Status: http.StatusCreated,
		Body:   []byte(`{"id":"4242"}`),
	}}, http.StatusCreated, []byte(`{"id":"4242"}`))

	var result map[string]any
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectFileAdd.String(), map[string]any{
		"project_id":    777,
		"reference":     "  tf_1a2b  ",
		"name":          "contract.pdf",
		"description":   "Signed original",
		"category_name": "Contracts",
		"private":       true,
	},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, message mcp.Result) {
			t.Helper()
			result = decodeToolJSON(t, message)
		}),
	)

	if len(*recorded) != 1 {
		t.Fatalf("expected one request, got %d", len(*recorded))
	}

	var body struct {
		File struct {
			PendingFileRef string `json:"pendingFileRef"`
			Name           string `json:"name"`
			Description    string `json:"description"`
			CategoryName   string `json:"category-name"`
			Private        bool   `json:"private"`
		} `json:"file"`
	}
	if err := json.Unmarshal((*recorded)[0].Body, &body); err != nil {
		t.Fatalf("failed to decode the request body: %s", err)
	}

	// Whitespace is the model's, not the API's, and the endpoint would reject
	// the padded form as an unknown reference rather than as untidy.
	if body.File.PendingFileRef != "tf_1a2b" {
		t.Errorf("expected the trimmed reference on the wire, got %q", body.File.PendingFileRef)
	}
	if body.File.Name != "contract.pdf" {
		t.Errorf("expected the name on the wire, got %q", body.File.Name)
	}
	if body.File.Description != "Signed original" {
		t.Errorf("expected the description on the wire, got %q", body.File.Description)
	}
	if body.File.CategoryName != "Contracts" {
		t.Errorf("expected the category name on the wire, got %q", body.File.CategoryName)
	}
	if !body.File.Private {
		t.Error("expected the file to be marked private on the wire")
	}

	// The durable handle, which is the whole reason to file it here.
	if got := result["id"]; got != float64(4242) {
		t.Errorf("expected the created file ID, got %v", got)
	}
}

func TestProjectFileAddRejectsBlankReference(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"id":"1"}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodProjectFileAdd.String(), map[string]any{
		"project_id": 777,
		"reference":  "   ",
	},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			t.Helper()
			assertErrorResultContains(t, result, "must not be empty")
		}),
	)

	if len(*recorded) != 0 {
		t.Errorf("expected no request to be sent, got %d", len(*recorded))
	}
}

// TestTaskAttachmentFileIDsReachTheWire covers the half that makes a filed
// document worth having: an identifier survives being used, so the same file
// reaches more than one task. The mocks reply the same either way, so this has
// to assert on the encoded body.
func TestTaskAttachmentFileIDsReachTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"task":{"id":1}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), map[string]any{
		"tasklist_id":         123,
		"name":                "Countersign the contract",
		"attachment_file_ids": []any{4242, 4243},
		"attachment_refs":     []any{"tf_A"},
	})

	if len(*recorded) == 0 {
		t.Fatal("expected the task to be created")
	}

	var body struct {
		Attachments struct {
			Files []struct {
				ID int64 `json:"id"`
			} `json:"files"`
			PendingFiles []struct {
				Reference string `json:"reference"`
			} `json:"pendingFiles"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal((*recorded)[0].Body, &body); err != nil {
		t.Fatalf("failed to decode the request body: %s", err)
	}

	if len(body.Attachments.Files) != 2 {
		t.Fatalf("expected two file attachments, got %d", len(body.Attachments.Files))
	}
	if body.Attachments.Files[0].ID != 4242 || body.Attachments.Files[1].ID != 4243 {
		t.Errorf("expected the file IDs on the wire, got %+v", body.Attachments.Files)
	}
	// The two forms are independent, so naming one must not drop the other.
	if len(body.Attachments.PendingFiles) != 1 || body.Attachments.PendingFiles[0].Reference != "tf_A" {
		t.Errorf("expected the pending reference alongside, got %+v", body.Attachments.PendingFiles)
	}
}

// TestTaskAttachmentsOmittedWithoutEither pins that adding the second parameter
// did not start sending an empty attachments object on every task.
func TestTaskAttachmentsOmittedWithoutEither(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"task":{"id":1}}`))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodTaskCreate.String(), map[string]any{
		"tasklist_id": 123,
		"name":        "No attachments here",
	})

	if len(*recorded) == 0 {
		t.Fatal("expected the task to be created")
	}
	if bytes.Contains((*recorded)[0].Body, []byte(`"attachments"`)) {
		t.Errorf("expected no attachments key, got %s", (*recorded)[0].Body)
	}
}

func TestFileListReachesTheWire(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{"files":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileList.String(), map[string]any{
		"project_id":          float64(777),
		"task_id":             float64(12345),
		"ids":                 []any{float64(1), float64(2)},
		"category_id":         float64(9),
		"tag_ids":             []any{float64(3)},
		"user_ids":            []any{float64(4)},
		"search_term":         "plan",
		"search_all_fields":   true,
		"uploaded_after":      "2026-08-01",
		"uploaded_before":     "2026-08-31",
		"updated_after":       "2026-08-15T10:00:00Z",
		"show_deleted":        true,
		"skip_external_files": true,
		"page":                float64(2),
		"page_size":           float64(25),
	})

	if !strings.HasSuffix(requestURL.Path, "/projects/api/v3/projects/777/files.json") {
		t.Errorf("expected the project-scoped route, got %s", requestURL.Path)
	}
	query := requestURL.Query()
	for key, want := range map[string]string{
		"taskId":            "12345",
		"ids":               "1,2",
		"categoryId":        "9",
		"tagIds":            "3",
		"userIds":           "4",
		"searchTerm":        "plan",
		"searchAllFields":   "true",
		"uploadedStartDate": "2026-08-01",
		"uploadedEndDate":   "2026-08-31",
		"updatedAfter":      "2026-08-15T10:00:00Z",
		"showDeleted":       "true",
		"skipExternalFiles": "true",
		"page":              "2",
		"pageSize":          "25",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("expected %s=%q on the query string, got %q", key, want, got)
		}
	}
	// verbose is the default, and it is what sideloads the uploader and project.
	if got := query.Get("include"); got != "users,projects" {
		t.Errorf("expected include=users,projects, got %q", got)
	}
}

func TestFileListUnscopedUsesTheGlobalRoute(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK, []byte(`{"files":[]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileList.String(), map[string]any{
		"verbose": false,
	})

	if !strings.HasSuffix(requestURL.Path, "/projects/api/v3/files.json") {
		t.Errorf("expected the global files route, got %s", requestURL.Path)
	}
	query := requestURL.Query()
	if got := query.Get("fields[files]"); got != "id,displayName,size" {
		t.Errorf("expected the terse field set, got %q", got)
	}
	if query.Has("include") {
		t.Errorf("expected no sideloads when verbose is false, got %q", query.Get("include"))
	}
}

func TestFileGetReachesTheWire(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"file":{"id":12345}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileGet.String(), map[string]any{
		"id":               float64(12345),
		"version":          float64(2),
		"include_versions": true,
	})

	if !strings.HasSuffix(requestURL.Path, "/projects/api/v3/files/12345.json") {
		t.Errorf("expected the single file route, got %s", requestURL.Path)
	}
	query := requestURL.Query()
	for key, want := range map[string]string{
		"version":     "2",
		"getVersions": "true",
		"include":     "users",
	} {
		if got := query.Get(key); got != want {
			t.Errorf("expected %s=%q on the query string, got %q", key, want, got)
		}
	}
}

func TestFileGetSelectionDropsTheSideload(t *testing.T) {
	mcpServer, requestURL := testutil.ProjectsMCPServerMockWithRequestURL(t, http.StatusOK,
		[]byte(`{"file":{"id":12345,"displayName":"plan.md"}}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileGet.String(), map[string]any{
		"id":     float64(12345),
		"fields": []any{"displayName"},
	})

	query := requestURL.Query()
	if got := query.Get("fields[files]"); got != "displayName,id" {
		t.Errorf("expected the selection plus id, got %q", got)
	}
	if query.Has("include") {
		t.Errorf("expected no sideload under a selection, got %q", query.Get("include"))
	}
}

// fileDownloadMock answers the download route with the given body and headers,
// as storage does once the redirect has been followed. The engine mocks build
// their responses without headers, and the content type is what decides how the
// tool returns the bytes, so this one sets them itself.
func fileDownloadMock(t *testing.T, contentType, disposition string, body []byte) *mcp.Server {
	t.Helper()

	engine := twapi.NewEngine(testutil.ProjectsSessionMock{},
		twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
				resp := pkgtestutil.NewMockHTTPResponse(http.StatusOK, body)
				resp.ContentLength = int64(len(body))
				resp.Header.Set("Content-Type", contentType)
				if disposition != "" {
					resp.Header.Set("Content-Disposition", disposition)
				}
				return resp, nil
			})
		}),
	)
	return pkgtestutil.MCPServer(t, twprojects.DefaultToolsetGroup(false, true, engine))
}

// downloadContents runs the download tool and returns the result's content
// blocks, failing the test on an error result.
func downloadContents(t *testing.T, mcpServer *mcp.Server, args map[string]any) []mcp.Content {
	t.Helper()

	var contents []mcp.Content
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileDownload.String(), args,
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			testutil.CheckMessage(t, result)
			contents = result.(*mcp.CallToolResult).Content
		}),
	)
	return contents
}

func TestFileDownloadReturnsTextAsText(t *testing.T) {
	mcpServer := fileDownloadMock(t, "text/markdown; charset=utf-8", `attachment; filename="plan.md"`,
		[]byte("# Plan\n"))
	contents := downloadContents(t, mcpServer, map[string]any{"id": float64(12345)})

	if len(contents) != 2 {
		t.Fatalf("expected a description and the content, got %d blocks", len(contents))
	}
	description, ok := contents[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected the first block to be text, got %T", contents[0])
	}
	var meta struct {
		Name     string `json:"name"`
		MIMEType string `json:"mimeType"`
		Size     int64  `json:"size"`
	}
	if err := json.Unmarshal([]byte(description.Text), &meta); err != nil {
		t.Fatalf("failed to decode the description: %v", err)
	}
	if meta.Name != "plan.md" || meta.MIMEType != "text/markdown" || meta.Size != 7 {
		t.Errorf("unexpected description %+v", meta)
	}
	text, ok := contents[1].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected the content to be text, got %T", contents[1])
	}
	if text.Text != "# Plan\n" {
		t.Errorf("expected the file's text, got %q", text.Text)
	}
}

func TestFileDownloadReturnsImagesAsImages(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n")
	mcpServer := fileDownloadMock(t, "image/png", `attachment; filename="logo.png"`, png)
	contents := downloadContents(t, mcpServer, map[string]any{"id": float64(12345)})

	if len(contents) != 2 {
		t.Fatalf("expected a description and the content, got %d blocks", len(contents))
	}
	image, ok := contents[1].(*mcp.ImageContent)
	if !ok {
		t.Fatalf("expected the content to be an image, got %T", contents[1])
	}
	if image.MIMEType != "image/png" || !bytes.Equal(image.Data, png) {
		t.Errorf("expected the PNG bytes under image/png, got %s with %d bytes", image.MIMEType, len(image.Data))
	}
}

func TestFileDownloadReturnsBinariesAsResources(t *testing.T) {
	pdf := []byte("%PDF-1.7\n")
	mcpServer := fileDownloadMock(t, "application/pdf", `attachment; filename="report.pdf"`, pdf)
	contents := downloadContents(t, mcpServer, map[string]any{"id": float64(12345), "version": float64(3)})

	if len(contents) != 2 {
		t.Fatalf("expected a description and the content, got %d blocks", len(contents))
	}
	resource, ok := contents[1].(*mcp.EmbeddedResource)
	if !ok {
		t.Fatalf("expected the content to be an embedded resource, got %T", contents[1])
	}
	if resource.Resource.MIMEType != "application/pdf" || !bytes.Equal(resource.Resource.Blob, pdf) {
		t.Errorf("expected the PDF bytes under application/pdf, got %s with %d bytes",
			resource.Resource.MIMEType, len(resource.Resource.Blob))
	}
	if resource.Resource.URI != "twprojects://files/12345" {
		t.Errorf("expected the resource to be addressed by file ID, got %q", resource.Resource.URI)
	}
}

func TestFileDownloadFallsBackToTheExtension(t *testing.T) {
	// Storage answers with the type the file was uploaded under, which is a
	// generic octet-stream when the uploader's client did not know better.
	mcpServer := fileDownloadMock(t, "application/octet-stream", `attachment; filename="data.csv"`,
		[]byte("a,b\n1,2\n"))
	contents := downloadContents(t, mcpServer, map[string]any{"id": float64(12345)})

	if len(contents) != 2 {
		t.Fatalf("expected a description and the content, got %d blocks", len(contents))
	}
	if _, ok := contents[1].(*mcp.TextContent); !ok {
		t.Errorf("expected a CSV to come back as text, got %T", contents[1])
	}
}

func TestFileDownloadRefusesOversizedFiles(t *testing.T) {
	// Declared size first: the body is never read when the server announces it
	// does not fit.
	engineTooBig := twapi.NewEngine(testutil.ProjectsSessionMock{},
		twapi.WithMiddleware(func(twapi.HTTPClient) twapi.HTTPClient {
			return twapi.HTTPClientFunc(func(*http.Request) (*http.Response, error) {
				resp := pkgtestutil.NewMockHTTPResponse(http.StatusOK, nil)
				resp.ContentLength = 11 << 20
				resp.Header.Set("Content-Type", "application/zip")
				return resp, nil
			})
		}),
	)
	mcpServer := pkgtestutil.MCPServer(t, twprojects.DefaultToolsetGroup(false, true, engineTooBig))

	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileDownload.String(),
		map[string]any{"id": float64(12345)},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			toolResult := result.(*mcp.CallToolResult)
			if !toolResult.IsError {
				t.Fatal("expected an error result for a file over the limit")
			}
			text := toolResult.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, "limit") || !strings.Contains(text, twprojects.MethodFileGet.String()) {
				t.Errorf("expected the error to name the limit and the tool carrying the downloadURL, got %q", text)
			}
		}),
	)
}

func TestFileDownloadAPIFailureIsAToolResult(t *testing.T) {
	mcpServer := mcpServerMock(t, http.StatusNotFound, []byte(`{"errors":[{"title":"not found"}]}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodFileDownload.String(),
		map[string]any{"id": float64(12345)},
		testutil.ExecuteToolRequestWithCheckMessage(func(t *testing.T, result mcp.Result) {
			toolResult := result.(*mcp.CallToolResult)
			if !toolResult.IsError {
				t.Fatal("expected a 404 to surface as an error tool result")
			}
		}),
	)
}
