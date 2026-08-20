package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/teamwork/mcp/pkg/toolsets"
)

// htmlPath is the committed GitHub Pages entry point, relative to this
// package's directory (the test working directory).
const htmlPath = "../../docs/index.html"

// toolRowPattern matches one rendered tool row, capturing its access badge and
// tool name. It is deliberately literal: if the template's row markup changes,
// these tests must be updated alongside it rather than silently matching zero
// rows, which TestEveryToolAppearsInTheHTML would catch.
var toolRowPattern = regexp.MustCompile(
	`<div class="tool" data-search="[^"]*" data-access="(read|write)">\s*` +
		`<span class="badge badge--(?:read|write)">(?:read|write)</span>\s*` +
		`<span class="tool__name">([^<]+)</span>`)

// TestGeneratedHTMLMatchesCommitted is the GOLDEN guard for the published page,
// the counterpart of TestGeneratedDocMatchesCommitted. GitHub Pages serves the
// committed file, so a tool added without regenerating would ship a page that
// disagrees with the server.
func TestGeneratedHTMLMatchesCommitted(t *testing.T) {
	committed, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("reading committed page %s: %v", filepath.Clean(htmlPath), err)
	}
	got, err := generateHTML()
	if err != nil {
		t.Fatalf("generating HTML: %v", err)
	}
	if got != string(committed) {
		t.Errorf("docs/index.html is stale — run `go run ./cmd/docs-gen` to regenerate it")
	}
}

// TestGeneratedHTMLIsDeterministic pins that nothing in the page comes from a
// map iteration or a clock. Without this the golden test above would fail
// intermittently rather than meaningfully, and every regeneration would produce
// a diff.
func TestGeneratedHTMLIsDeterministic(t *testing.T) {
	first, err := generateHTML()
	if err != nil {
		t.Fatalf("generating HTML: %v", err)
	}
	second, err := generateHTML()
	if err != nil {
		t.Fatalf("regenerating HTML: %v", err)
	}
	if first != second {
		t.Error("two runs produced different pages: something iterates a map or reads the clock")
	}
}

// TestEveryToolAppearsInTheHTMLExactlyOnce is the BIJECTION guard for the page:
// the rendered rows and the registry must be the same set. It fails both ways —
// a tool missing from the page, and a row that traces back to no tool.
func TestEveryToolAppearsInTheHTMLExactlyOnce(t *testing.T) {
	page, err := generateHTML()
	if err != nil {
		t.Fatalf("generating HTML: %v", err)
	}

	rendered := map[string]string{} // tool name -> access badge
	for _, match := range toolRowPattern.FindAllStringSubmatch(page, -1) {
		access, name := match[1], match[2]
		if prev, dup := rendered[name]; dup {
			t.Errorf("tool %q is rendered twice (%s, then %s)", name, prev, access)
		}
		rendered[name] = access
	}

	var registered int
	for _, p := range products() {
		for method, ts := range p.group.Toolsets {
			for _, tw := range ts.GetAvailableTools() {
				registered++
				access, ok := rendered[tw.Tool.Name]
				if !ok {
					t.Errorf("%s/%s: tool %q is missing from the page", p.label, method, tw.Tool.Name)
					continue
				}
				// The badge is the only place a reader learns whether a tool
				// writes, so it must track the annotation, not the tool's name.
				want := "write"
				if tw.Tool.Annotations != nil && tw.Tool.Annotations.ReadOnlyHint {
					want = "read"
				}
				if access != want {
					t.Errorf("%s/%s: tool %q is badged %q but its ReadOnlyHint says %q",
						p.label, method, tw.Tool.Name, access, want)
				}
			}
		}
	}
	if len(rendered) != registered {
		t.Errorf("page renders %d tools but the registry has %d: a row does not "+
			"trace back to a tool, or vice versa", len(rendered), registered)
	}
}

// TestHTMLStatsMatchTheRenderedRows guards the hero counters, which are the
// first thing a reader takes away and the easiest number to leave stale.
func TestHTMLStatsMatchTheRenderedRows(t *testing.T) {
	page, err := generateHTML()
	if err != nil {
		t.Fatalf("generating HTML: %v", err)
	}
	_, stats := collectProducts()

	rows := toolRowPattern.FindAllStringSubmatch(page, -1)
	var read, write int
	for _, match := range rows {
		if match[1] == "read" {
			read++
		} else {
			write++
		}
	}
	if stats.Tools != len(rows) {
		t.Errorf("hero reports %d tools, page renders %d", stats.Tools, len(rows))
	}
	if stats.Read != read || stats.Write != write {
		t.Errorf("hero reports %d read / %d write, page renders %d / %d",
			stats.Read, stats.Write, read, write)
	}
	if stats.Read+stats.Write != stats.Tools {
		t.Errorf("read (%d) + write (%d) != total (%d): a tool was counted in neither",
			stats.Read, stats.Write, stats.Tools)
	}
}

// TestHTMLListsEveryProfile pins that the profile table is generated rather than
// hand-maintained. Profiles double as HTTP paths, so one missing from the page
// is an endpoint nobody can discover.
func TestHTMLListsEveryProfile(t *testing.T) {
	page, err := generateHTML()
	if err != nil {
		t.Fatalf("generating HTML: %v", err)
	}
	names := toolsets.ListProfiles()
	if len(names) == 0 {
		t.Fatal("no profiles registered: generateHTML must call cli.NewMethods first")
	}
	for _, name := range names {
		if !strings.Contains(page, endpointURL+"/"+name) {
			t.Errorf("profile %q has no row in the profile table", name)
		}
	}
}

// TestHTMLHasNoEscapingFailures catches html/template refusing to emit a value
// it could not prove safe — a data: favicon in an href, or the inlined script,
// would be replaced with this marker rather than failing the build.
func TestHTMLHasNoEscapingFailures(t *testing.T) {
	page, err := generateHTML()
	if err != nil {
		t.Fatalf("generating HTML: %v", err)
	}
	if strings.Contains(page, "ZgotmplZ") {
		t.Error("page contains ZgotmplZ: html/template refused to emit a value in a URL or script context")
	}
}

func TestFirstSentence(t *testing.T) {
	tests := []struct {
		name string
		desc string
		want string
	}{
		{"plain", "Get task.", "Get task."},
		{"stops at the first sentence", "Get task. Returns the full object.", "Get task."},
		{"no terminator", "Get task", "Get task"},
		{"keeps e.g.", "Filter tasks, e.g. by tag. And more.", "Filter tasks, e.g. by tag."},
		{"keeps i.e.", "One call, i.e. no paging. Second sentence.", "One call, i.e. no paging."},
		{"keeps decimals", "Reads v3.json for the project. Next.", "Reads v3.json for the project."},
		{"collapses whitespace", "Get\n  task.\n\nMore.", "Get task."},
		{"question", "How many? Ask this.", "How many?"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := firstSentence(tt.desc); got != tt.want {
				t.Errorf("firstSentence(%q) = %q, want %q", tt.desc, got, tt.want)
			}
		})
	}
}

// TestHTMLToolDescriptionsAreNotEmpty guards the one thing the page adds over
// the Markdown matrix: a per-tool description. A tool whose description is blank
// renders as a bare name and tells a reader nothing.
func TestHTMLToolDescriptionsAreNotEmpty(t *testing.T) {
	prods, _ := collectProducts()
	for _, p := range prods {
		for _, ts := range p.Toolsets {
			for _, tool := range ts.Tools {
				if strings.TrimSpace(tool.Description) == "" {
					t.Errorf("%s/%s: tool %q has no description to render", p.Label, ts.Method, tool.Name)
				}
			}
		}
	}
}
