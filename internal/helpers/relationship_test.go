package helpers_test

import (
	"encoding/json"
	"testing"

	"github.com/teamwork/mcp/internal/helpers"
)

func TestRelationshipMetaID(t *testing.T) {
	tests := []struct {
		name  string
		meta  map[string]any
		key   string
		want  int64
		wantR bool
	}{{
		name:  "json number decoded as float64",
		meta:  map[string]any{"projectId": float64(5)},
		key:   "projectId",
		want:  5,
		wantR: true,
	}, {
		name:  "int64",
		meta:  map[string]any{"projectId": int64(5)},
		key:   "projectId",
		want:  5,
		wantR: true,
	}, {
		name:  "int",
		meta:  map[string]any{"projectId": 5},
		key:   "projectId",
		want:  5,
		wantR: true,
	}, {
		name:  "json.Number",
		meta:  map[string]any{"projectId": json.Number("5")},
		key:   "projectId",
		want:  5,
		wantR: true,
	}, {
		name:  "string",
		meta:  map[string]any{"projectId": "5"},
		key:   "projectId",
		want:  5,
		wantR: true,
	}, {
		name: "missing key",
		meta: map[string]any{"other": float64(5)},
		key:  "projectId",
	}, {
		name: "nil meta",
		key:  "projectId",
	}, {
		name: "zero is not an ID",
		meta: map[string]any{"projectId": float64(0)},
		key:  "projectId",
	}, {
		name: "negative is not an ID",
		meta: map[string]any{"projectId": float64(-1)},
		key:  "projectId",
	}, {
		name: "unparsable string",
		meta: map[string]any{"projectId": "abc"},
		key:  "projectId",
	}, {
		name: "unexpected type",
		meta: map[string]any{"projectId": []any{float64(5)}},
		key:  "projectId",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := helpers.RelationshipMetaID(tt.meta, tt.key)
			if ok != tt.wantR {
				t.Errorf("expected ok %t, got %t", tt.wantR, ok)
			}
			if got != tt.want {
				t.Errorf("expected id %d, got %d", tt.want, got)
			}
		})
	}
}
