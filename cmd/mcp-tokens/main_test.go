package main

import (
	"strings"
	"testing"

	"github.com/localit-io/tiktoken-go"
)

func TestCountToolsSmoke(t *testing.T) {
	enc, err := tiktoken.GetEncoding("o200k_base")
	if err != nil {
		t.Fatalf("get encoding: %v", err)
	}

	rows := countTools(allGroups(), enc)
	if len(rows) == 0 {
		t.Fatal("expected at least one tool, got 0")
	}

	prefixes := map[string]bool{"twprojects-": false, "twdesk-": false, "twspaces-": false}
	for _, r := range rows {
		if r.Tokens <= 0 {
			t.Errorf("tool %q has non-positive token count %d", r.Name, r.Tokens)
		}
		for p := range prefixes {
			if strings.HasPrefix(r.Name, p) {
				prefixes[p] = true
			}
		}
	}
	for p, seen := range prefixes {
		if !seen {
			t.Errorf("no tools registered with prefix %q", p)
		}
	}

	for i := 1; i < len(rows); i++ {
		if rows[i-1].Tokens < rows[i].Tokens {
			t.Errorf("rows not sorted by tokens desc at %d: %d < %d", i, rows[i-1].Tokens, rows[i].Tokens)
		}
	}
}

func TestBuildDiff(t *testing.T) {
	before := []toolCount{{"a", 100}, {"b", 50}, {"removed", 200}}
	after := []toolCount{{"a", 80}, {"b", 50}, {"added", 30}}

	rows := buildDiff(before, after)
	if len(rows) != 4 {
		t.Fatalf("want 4 rows (union of names), got %d", len(rows))
	}

	got := map[string]diffRow{}
	for _, r := range rows {
		got[r.Name] = r
	}
	if got["a"].Before != 100 || got["a"].After != 80 || got["a"].Delta() != -20 {
		t.Errorf("a row wrong: %+v", got["a"])
	}
	if got["b"].Delta() != 0 {
		t.Errorf("b row should have delta 0: %+v", got["b"])
	}
	if got["removed"].Before != 200 || got["removed"].After != 0 {
		t.Errorf("removed row should have After=0: %+v", got["removed"])
	}
	if got["added"].Before != 0 || got["added"].After != 30 {
		t.Errorf("added row should have Before=0: %+v", got["added"])
	}

	// Sort: most-saved first, then by name. "removed" has -200, "a" has -20,
	// "b" has 0, "added" has +30.
	wantOrder := []string{"removed", "a", "b", "added"}
	for i, name := range wantOrder {
		if rows[i].Name != name {
			t.Errorf("position %d: want %q, got %q", i, name, rows[i].Name)
		}
	}
}
