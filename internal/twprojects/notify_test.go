package twprojects

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// White-box: SDK schema validation normally runs first, but the binder still
// guards transports that skip it (case-insensitive "all", fallback error).
func TestParseNotify(t *testing.T) {
	tests := []struct {
		name          string
		arguments     map[string]any
		withFollowers bool
		wantChoice    notifyChoice
		wantUserIDs   []int64
	}{
		{
			name:       "absent means default",
			arguments:  map[string]any{},
			wantChoice: notifyChoiceDefault,
		},
		{
			name:       "explicit null means default",
			arguments:  map[string]any{"notify": nil},
			wantChoice: notifyChoiceDefault,
		},
		{
			name:       "string all is case-insensitive",
			arguments:  map[string]any{"notify": "ALL"},
			wantChoice: notifyChoiceAll,
		},
		{
			name:       "boolean true coerces to all",
			arguments:  map[string]any{"notify": true},
			wantChoice: notifyChoiceAll,
		},
		{
			name:          "boolean true means followers when supported",
			arguments:     map[string]any{"notify": true},
			withFollowers: true,
			wantChoice:    notifyChoiceFollowers,
		},
		{
			name:       "boolean false means nobody",
			arguments:  map[string]any{"notify": false},
			wantChoice: notifyChoiceNone,
		},
		{
			name:          "boolean false means nobody with followers too",
			arguments:     map[string]any{"notify": false},
			withFollowers: true,
			wantChoice:    notifyChoiceNone,
		},
		{
			name:        "array of user IDs coerces to a group",
			arguments:   map[string]any{"notify": []any{float64(1), float64(2)}},
			wantChoice:  notifyChoiceGroup,
			wantUserIDs: []int64{1, 2},
		},
		{
			name:        "object of user groups",
			arguments:   map[string]any{"notify": map[string]any{"user_ids": []any{float64(7)}}},
			wantChoice:  notifyChoiceGroup,
			wantUserIDs: []int64{7},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			choice, groups, toolResult := parseNotify(tt.arguments, tt.withFollowers)
			if toolResult != nil {
				t.Fatalf("unexpected error result: %v", toolResult.Content)
			}
			if choice != tt.wantChoice {
				t.Errorf("expected choice %v, got %v", tt.wantChoice, choice)
			}
			if tt.wantUserIDs == nil {
				return
			}
			if groups == nil {
				t.Fatal("expected user groups, got nil")
			}
			if len(groups.UserIDs) != len(tt.wantUserIDs) {
				t.Fatalf("expected user IDs %v, got %v", tt.wantUserIDs, groups.UserIDs)
			}
			for i, id := range tt.wantUserIDs {
				if groups.UserIDs[i] != id {
					t.Errorf("expected user IDs %v, got %v", tt.wantUserIDs, groups.UserIDs)
				}
			}
		})
	}
}

// The fallback error must list every accepted shape.
func TestParseNotifyRejectsUnknownShapes(t *testing.T) {
	for _, notify := range []any{"everyone", float64(7), []any{"bob"}, []any{}} {
		_, _, toolResult := parseNotify(map[string]any{"notify": notify}, false)
		if toolResult == nil {
			t.Fatalf("expected notify %#v to be rejected", notify)
		}
		text, ok := toolResult.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("unexpected content type: %T", toolResult.Content[0])
		}
		if !strings.Contains(text.Text, `notify must be the string "all"`) ||
			!strings.Contains(text.Text, "array of user IDs") ||
			!strings.Contains(text.Text, "user_ids, company_ids, team_ids and/or job_role_ids") {
			t.Errorf("error should list the accepted shapes, got %q", text.Text)
		}
	}
}
