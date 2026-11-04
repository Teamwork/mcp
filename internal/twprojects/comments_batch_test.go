package twprojects_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/teamwork/mcp/internal/testutil"
	"github.com/teamwork/mcp/internal/twprojects"
)

// TestCommentCreateBatchReachesTheWire pins one create per comment on its own
// record, each with its own notify: followers by default, nobody on false.
func TestCommentCreateBatchReachesTheWire(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, []testutil.ProjectsMockRoute{
		{Method: http.MethodPost, Match: "/milestones/2/", Status: http.StatusCreated, Body: []byte(`{"id":"502"}`)},
	}, http.StatusCreated, []byte(`{"id":"501"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreateBatch.String(), map[string]any{
		"comments": []any{
			map[string]any{"object": map[string]any{"type": "tasks", "id": float64(1)}, "body": "Done"},
			map[string]any{"object": map[string]any{"type": "milestones", "id": float64(2)}, "body": "Late",
				"notify": false},
		},
	}, expectBatchResult(false, "Created 2 of 2 comments", "item 1 on task 1 → comment 501",
		"item 2 on milestone 2 → comment 502"))

	want := map[string]string{
		"/tasks/1/comments.json":      `true`,
		"/milestones/2/comments.json": ``,
	}
	writes := requestsOfMethod(*recorded, http.MethodPost)
	if len(writes) != len(want) {
		t.Fatalf("expected %d writes, got %d", len(want), len(writes))
	}
	for _, entry := range writes {
		var payload struct {
			Comment struct {
				Notify json.RawMessage `json:"notify"`
			} `json:"comment"`
		}
		if err := json.Unmarshal(entry.Body, &payload); err != nil {
			t.Fatalf("failed to decode body %q: %v", entry.Body, err)
		}
		if notify, ok := want[entry.URL.Path]; !ok || string(payload.Comment.Notify) != notify {
			t.Errorf("%s: expected notify %q, got %s", entry.URL.Path, notify, payload.Comment.Notify)
		}
	}
}

// TestCommentCreateBatchRejectsAnInvalidItemBeforeWriting keeps a bad item from
// leaving the rest half created.
func TestCommentCreateBatchRejectsAnInvalidItemBeforeWriting(t *testing.T) {
	mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"id":"1"}`))
	testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreateBatch.String(), map[string]any{
		"comments": []any{
			map[string]any{"object": map[string]any{"type": "tasks", "id": float64(1)}, "body": "Fine"},
			map[string]any{"object": map[string]any{"type": "tasks", "id": float64(2)}, "body": "Bad",
				"notify": []any{}},
		},
	}, expectBatchResult(true, "comments/items/properties/notify"))

	if len(*recorded) != 0 {
		t.Errorf("expected no requests, got %d", len(*recorded))
	}
}

// TestCommentCreateBatchRejectsAttachments keeps the batch from applying
// parameters its item schema does not advertise.
func TestCommentCreateBatchRejectsAttachments(t *testing.T) {
	for _, name := range []string{"attachment_refs", "attachment_file_ids"} {
		mcpServer, recorded := mcpServerRecordingMock(t, nil, http.StatusCreated, []byte(`{"id":"1"}`))
		testutil.ExecuteToolRequest(t, mcpServer, twprojects.MethodCommentCreateBatch.String(), map[string]any{
			"comments": []any{
				map[string]any{"object": map[string]any{"type": "tasks", "id": float64(1)}, "body": "Fine",
					name: []any{float64(5)}},
			},
		}, expectBatchResult(true, "Nothing was written", "item 1: "+name+" is not accepted here"))

		if len(*recorded) != 0 {
			t.Errorf("%s: expected no requests, got %d", name, len(*recorded))
		}
	}
}
