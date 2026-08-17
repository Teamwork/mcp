package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name       string
		title      string
		body       string
		wantLevel  bumpLevel
		wantPrefix string
		classified bool
	}{
		{
			name: "feature", title: "Feature: File attachments",
			wantLevel: bumpMinor, wantPrefix: "feature", classified: true,
		}, {
			name: "feat", title: "feat: add a thing",
			wantLevel: bumpMinor, wantPrefix: "feat", classified: true,
		}, {
			name: "fix", title: "Fix: team tools fail output-schema validation",
			wantLevel: bumpPatch, wantPrefix: "fix", classified: true,
		}, {
			name: "enhancement", title: "Enhancement: Exact counts without paging rows",
			wantLevel: bumpPatch, wantPrefix: "enhancement", classified: true,
		}, {
			name: "scoped chore", title: "Chore(deps): Bump github.com/sonh/qs from 0.6.4 to 0.7.0",
			wantLevel: bumpPatch, wantPrefix: "chore", classified: true,
		}, {
			name: "lowercase", title: "docs: tidy the readme",
			wantLevel: bumpPatch, wantPrefix: "docs", classified: true,
		}, {
			name: "bang is breaking", title: "Feature!: drop the v1 tool names",
			wantLevel: bumpMajor, wantPrefix: "feature", classified: true,
		}, {
			name: "scoped bang is breaking", title: "Fix(tasks)!: rename parent_task_id",
			wantLevel: bumpMajor, wantPrefix: "fix", classified: true,
		}, {
			name: "breaking footer", title: "Fix: rename a parameter",
			body:      "BREAKING CHANGE: parent_task_id is gone",
			wantLevel: bumpMajor, wantPrefix: "fix", classified: true,
		}, {
			name: "breaking footer hyphenated", title: "Fix: rename a parameter",
			body:      "BREAKING-CHANGE: parent_task_id is gone",
			wantLevel: bumpMajor, wantPrefix: "fix", classified: true,
		}, {
			name: "breaking footer without a prefix", title: "Rename a parameter",
			body:      "BREAKING CHANGE: parent_task_id is gone",
			wantLevel: bumpMajor, classified: true,
		},

		// Real unprefixed titles from the history. Two of them were features
		// that shipped under a patch tag, which is why an unrecognised prefix
		// is reported rather than trusted.
		{
			name: "no prefix", title: "Ordering for the projects list tools",
			wantLevel: bumpPatch,
		}, {
			name: "no prefix, was a feature", title: "Adds support for linking tasks to desks.",
			wantLevel: bumpPatch,
		}, {
			name: "sentence with a colon later", title: "Filter comments by IDs. This allows: analysis",
			wantLevel: bumpPatch,
		}, {
			name: "unknown prefix", title: "Note: something happened",
			wantLevel: bumpPatch, wantPrefix: "note",
		}, {
			name: "empty", title: "",
			wantLevel: bumpPatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, level, ok := classify(tt.title, tt.body)
			if level != tt.wantLevel {
				t.Errorf("level: want %s, got %s", tt.wantLevel, level)
			}
			if prefix != tt.wantPrefix {
				t.Errorf("prefix: want %q, got %q", tt.wantPrefix, prefix)
			}
			if ok != tt.classified {
				t.Errorf("classified: want %v, got %v", tt.classified, ok)
			}
		})
	}
}

// TestEveryContributingPrefixIsMapped keeps the prefix table and the prefixes
// CONTRIBUTING.md asks contributors to use from drifting apart: a documented
// prefix the tool does not know would be reported as unclassified on every
// release that used it.
func TestEveryContributingPrefixIsMapped(t *testing.T) {
	doc, err := os.ReadFile("../../CONTRIBUTING.md")
	if err != nil {
		t.Fatalf("read CONTRIBUTING.md: %v", err)
	}

	documented := regexp.MustCompile("(?m)^\\s+- `([A-Za-z]+):` for ").FindAllStringSubmatch(string(doc), -1)
	if len(documented) == 0 {
		t.Fatal("found no documented prefixes in CONTRIBUTING.md; has the list moved?")
	}

	for _, match := range documented {
		prefix := strings.ToLower(match[1])
		if _, ok := bumpByPrefix[prefix]; !ok {
			t.Errorf("CONTRIBUTING.md documents %q but bumpByPrefix does not map it", prefix)
		}
	}
}

// TestLintedTypesAreUnderstood pins the PR title lint to this tool. A type the
// lint accepts but bumpByPrefix does not know would sail through review and
// then be reported as unclassified at release time, which is the failure the
// lint exists to remove.
func TestLintedTypesAreUnderstood(t *testing.T) {
	types := lintedTypes(t)

	// A guard on the parsing itself: an empty or tiny list means the types
	// block moved, not that the lint accepts nothing.
	if len(types) < 10 {
		t.Fatalf("parsed only %v from the workflow's types block; has it moved?", types)
	}
	for _, want := range []string{"feature", "fix", "enhancement", "chore"} {
		if !slices.Contains(types, want) {
			t.Errorf("pr_lint.yaml should accept %q, parsed %v", want, types)
		}
	}

	for _, prefix := range types {
		if _, ok := bumpByPrefix[prefix]; !ok {
			t.Errorf("pr_lint.yaml accepts %q but bumpByPrefix does not map it", prefix)
		}
	}
}

// lintedTypes reads the words out of the `types:` block in the PR lint
// workflow. The entries are regexes ("(Feature|feature|feat)"), so every
// alphabetic run in them is a type the lint accepts.
func lintedTypes(t *testing.T) []string {
	t.Helper()

	data, err := os.ReadFile("../../.github/workflows/pr_lint.yaml")
	if err != nil {
		t.Fatalf("read pr_lint.yaml: %v", err)
	}

	var (
		types  []string
		word   = regexp.MustCompile(`[A-Za-z]+`)
		indent = -1
	)
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
		trimmed := strings.TrimSpace(line)
		depth := len(line) - len(strings.TrimLeft(line, " "))

		switch {
		case indent < 0:
			if trimmed == "types: |" {
				indent = depth
			}
		case trimmed == "":
			continue
		case depth <= indent:
			return types
		default:
			for _, w := range word.FindAllString(trimmed, -1) {
				types = append(types, strings.ToLower(w))
			}
		}
	}
	return types
}

// TestHistoricalRangesComputeTheRightBump replays real ranges from the history,
// classifying commit subjects the way the tool does when a pull-request lookup
// is unavailable. The three ranges marked as under-versioned shipped a patch
// tag over a feature; those are the tags this tool exists to prevent.
func TestHistoricalRangesComputeTheRightBump(t *testing.T) {
	tests := []struct {
		name           string
		previous       string
		subjects       []string
		wantVersion    string
		wantUnclassifd int
	}{
		{
			name:        "v1.28.3 shipped a feature as a patch",
			previous:    "v1.28.2",
			wantVersion: "v1.29.0",
			subjects: []string{
				"Fix: a predecessors field selection returns an empty array (#462)",
				"Ordering for the projects list tools (#463)",
				"Feature: File attachments",
				"Address review and simplification feedback",
				"Adapt file uploads to pre-signed URLs",
				"Fix comment",
				"Set the correct SDK version",
				"Rename mcptest to mcp-test and group its checks into suites",
				"Run `go fix ./...`",
				"Enhancement: Exact counts without paging rows",
			},
			// Every commit of the rebase-merged branch that carries no prefix
			// of its own. With a pull-request lookup they collapse into one
			// classified change; this is the subject-only fallback.
			wantUnclassifd: 7,
		},
		{
			name:        "v1.27.4 shipped a feature as a patch",
			previous:    "v1.27.3",
			wantVersion: "v1.28.0",
			subjects: []string{
				"Feature: Move a task and its subtasks in one call (#446)",
				"Enhancement: Use the tasklist project hint in the skills and roles prompt",
			},
		},
		{
			name:        "enhancement-only stays a patch, as v1.28.4 shipped",
			previous:    "v1.28.3",
			wantVersion: "v1.28.4",
			subjects: []string{
				"tw-mcp v1.28.3 (#465)",
				"Enhancement: Exact counts without paging rows",
			},
		},
		{
			name:        "fix-only stays a patch, as v1.28.2 shipped",
			previous:    "v1.28.1",
			wantVersion: "v1.28.2",
			subjects: []string{
				"Fix: team tools fail output-schema validation on every call (#458)",
			},
		},
		{
			name:        "dependency chores stay a patch, as v1.27.3 shipped",
			previous:    "v1.27.2",
			wantVersion: "v1.27.3",
			subjects: []string{
				"Fix: fields param rejected guessed names because its vocabulary was invisible",
				"Chore(deps): Bump github.com/sonh/qs from 0.6.4 to 0.7.0",
				"Chore(deps): Bump github.com/DataDog/dd-trace-go/contrib/net/http/v2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commits := make([]commit, 0, len(tt.subjects))
			for i, subject := range tt.subjects {
				commits = append(commits, commit{SHA: fmt.Sprintf("%040d", i), Subject: subject})
			}

			changes, _ := collectChanges(commits, nil, nil)

			previous, ok := parseVersion(tt.previous)
			if !ok {
				t.Fatalf("bad previous tag %q in the table", tt.previous)
			}
			if got := previous.next(highestLevel(changes)).String(); got != tt.wantVersion {
				t.Errorf("version: want %s, got %s", tt.wantVersion, got)
			}

			var unclassified int
			for _, c := range changes {
				if !c.Classified {
					unclassified++
				}
			}
			if unclassified != tt.wantUnclassifd {
				t.Errorf("unclassified: want %d, got %d", tt.wantUnclassifd, unclassified)
			}
		})
	}
}

func TestCollectChangesFoldsCommitsIntoPullRequests(t *testing.T) {
	commits := []commit{
		{SHA: "aaaaaaaa1", Subject: "Feature: File attachments"},
		{SHA: "aaaaaaaa2", Subject: "Address review feedback"},
		{SHA: "aaaaaaaa3", Subject: "Fix comment"},
		{SHA: "bbbbbbbb1", Subject: "Fix: a predecessors selection is empty (#462)"},
		{SHA: "cccccccc1", Subject: "tw-mcp v1.28.3 (#465)"},
		{SHA: "dddddddd1", Subject: "Enhancement: Exact counts"},
	}

	prs := map[string]*pullRequest{
		"aaaaaaaa1": {Number: 460, Title: "Feature: File attachments", MergedAt: "2026-08-01T00:00:00Z"},
		"aaaaaaaa2": {Number: 460, Title: "Feature: File attachments", MergedAt: "2026-08-01T00:00:00Z"},
		"aaaaaaaa3": {Number: 460, Title: "Feature: File attachments", MergedAt: "2026-08-01T00:00:00Z"},
		"bbbbbbbb1": {Number: 462, Title: "Fix: a predecessors selection is empty", MergedAt: "2026-08-02T00:00:00Z"},
		"cccccccc1": {Number: 465, Title: "chore: update Homebrew formula for v1.28.3", MergedAt: "2026-08-03T00:00:00Z"},
	}

	changes, notes := collectChanges(commits, func(sha string) (*pullRequest, error) {
		return prs[sha], nil
	}, nil)
	if len(notes) != 0 {
		t.Errorf("unexpected notes: %v", notes)
	}

	// The three commits of #460 collapse into one change; the release chore is
	// dropped; the commit with no pull request keeps its subject and short sha.
	want := []change{
		{Ref: "#460", Title: "Feature: File attachments", Prefix: "feature", Level: bumpMinor, Classified: true},
		{Ref: "#462", Title: "Fix: a predecessors selection is empty", Prefix: "fix", Level: bumpPatch, Classified: true},
		{Ref: "dddddddd", Title: "Enhancement: Exact counts", Prefix: "enhancement", Level: bumpPatch, Classified: true},
	}
	if len(changes) != len(want) {
		t.Fatalf("want %d changes, got %d: %+v", len(want), len(changes), changes)
	}
	for i, w := range want {
		if changes[i] != w {
			t.Errorf("change %d: want %+v, got %+v", i, w, changes[i])
		}
	}

	if got := highestLevel(changes); got != bumpMinor {
		t.Errorf("level: want minor, got %s", got)
	}
}

func TestCollectChangesFallsBackWhenLookupFails(t *testing.T) {
	commits := []commit{
		{SHA: "aaaaaaaa1", Subject: "Feature: File attachments"},
		{SHA: "bbbbbbbb1", Subject: "Fix: something"},
	}

	changes, notes := collectChanges(commits, func(string) (*pullRequest, error) {
		return nil, fmt.Errorf("403 Forbidden")
	}, nil)

	if len(changes) != 2 {
		t.Fatalf("want both commits classified by subject, got %+v", changes)
	}
	if changes[0].Ref != "aaaaaaaa" || changes[0].Level != bumpMinor {
		t.Errorf("first change should fall back to its subject: %+v", changes[0])
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "403 Forbidden") {
		t.Errorf("want a note naming the failure, got %v", notes)
	}
}

func TestHighestVersionTagIgnoresNonVersions(t *testing.T) {
	tags := []string{"v1.9.0", "v1.28.4", "vnext", "v1.28.3", "v2", "v1.28.10", "v1.28.4-rc1", ""}
	if got := highestVersionTag(tags); got != "v1.28.10" {
		t.Errorf("want v1.28.10 (numeric, not lexical), got %q", got)
	}
	if got := highestVersionTag(nil); got != "" {
		t.Errorf("want no tag, got %q", got)
	}
}

func TestVersionNextResetsLowerComponents(t *testing.T) {
	v := version{1, 28, 4}
	tests := []struct {
		level bumpLevel
		want  string
	}{
		{bumpPatch, "v1.28.5"},
		{bumpMinor, "v1.29.0"},
		{bumpMajor, "v2.0.0"},
		{bumpNone, "v1.28.4"},
	}
	for _, tt := range tests {
		if got := v.next(tt.level).String(); got != tt.want {
			t.Errorf("%s bump: want %s, got %s", tt.level, tt.want, got)
		}
	}
}

func TestPickPullRequestPrefersMerged(t *testing.T) {
	open := pullRequest{Number: 1, Title: "open branch that contains the commit"}
	merged := pullRequest{Number: 2, Title: "Feature: the one that shipped", MergedAt: "2026-08-01T00:00:00Z"}

	if got := pickPullRequest([]pullRequest{open, merged}); got == nil || got.Number != 2 {
		t.Errorf("want the merged pull request, got %+v", got)
	}
	if got := pickPullRequest([]pullRequest{open}); got != nil {
		t.Errorf("want nil when nothing merged, got %+v", got)
	}
	if got := pickPullRequest(nil); got != nil {
		t.Errorf("want nil for no pull requests, got %+v", got)
	}
}

func TestRepoFromRemote(t *testing.T) {
	tests := map[string]string{
		"https://github.com/teamwork/mcp.git\n": "teamwork/mcp",
		"https://github.com/teamwork/mcp":       "teamwork/mcp",
		"git@github.com:teamwork/mcp.git":       "teamwork/mcp",
		"ssh://git@github.com/teamwork/mcp.git": "teamwork/mcp",
		"https://gitlab.com/teamwork/mcp.git":   "",
	}
	for remote, want := range tests {
		if got := repoFromRemote(remote); got != want {
			t.Errorf("%q: want %q, got %q", remote, want, got)
		}
	}
}

func TestReportReportsUnclassifiedChanges(t *testing.T) {
	rep := report{
		Previous: version{1, 28, 4},
		Next:     version{1, 28, 5},
		Level:    bumpPatch,
		Changes: []change{
			{Ref: "#470", Title: "Fix: a | in a title", Prefix: "fix", Level: bumpPatch, Classified: true},
			{Ref: "#471", Title: "Adds support for something", Level: bumpPatch},
		},
	}

	outputs := rep.outputs()
	for _, want := range []string{"version=v1.28.5", "previous_tag=v1.28.4", "bump=patch", "unclassified=1"} {
		if !strings.Contains(outputs, want) {
			t.Errorf("outputs missing %q:\n%s", want, outputs)
		}
	}

	markdown := rep.markdown()
	for _, want := range []string{"[!WARNING]", "bump: minor", `a \| in a title`, "⚠️ patch"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("markdown missing %q:\n%s", want, markdown)
		}
	}

	if text := rep.text(); !strings.Contains(text, "WARNING: 1 change(s)") {
		t.Errorf("text missing the warning:\n%s", text)
	}
}

func TestParseBump(t *testing.T) {
	tests := []struct {
		in   string
		want bumpLevel
	}{
		{"auto", bumpNone},
		{"", bumpNone},
		{"patch", bumpPatch},
		{"MINOR", bumpMinor},
		{" major ", bumpMajor}, // the workflow interpolates the input as-is
	}
	for _, tt := range tests {
		got, err := parseBump(tt.in)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("%q: want %s, got %s", tt.in, tt.want, got)
		}
	}
	if _, err := parseBump("massive"); err == nil {
		t.Error("want an error for an unknown bump")
	}
}
