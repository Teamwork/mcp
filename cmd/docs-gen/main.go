// Command docs-gen generates the Teamwork MCP tool reference directly from the
// registered toolsets, in two forms: a Markdown CRUD matrix (docs/tool-reference.md)
// and a browsable HTML page published to GitHub Pages (docs/index.html).
//
// It builds each product's default toolset group with writes and deletes
// enabled so the full surface is visible, then reflects over each tool's static
// metadata (name, read-only hint). No API token or live server is required —
// the engine / HTTP client are only used inside handler closures, never at
// registration time, so nil dependencies are safe here.
//
// Usage:
//
//	go run ./cmd/docs-gen              # write both documents
//	go run ./cmd/docs-gen -o -         # write the Markdown to stdout
//	go run ./cmd/docs-gen -html -      # write the HTML page to stdout
//	go run ./cmd/docs-gen -check       # verify both committed documents are current
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/teamwork/mcp/internal/twchat"
	"github.com/teamwork/mcp/internal/twdesk"
	"github.com/teamwork/mcp/internal/twprojects"
	"github.com/teamwork/mcp/internal/twspaces"
	"github.com/teamwork/mcp/pkg/toolsets"
)

// product pairs a human-readable label with its registered toolset group.
type product struct {
	label string
	group *toolsets.ToolsetGroup
}

// products lists every Teamwork product surface in display order. Dependencies
// are nil: only static tool metadata is read, never the engine/HTTP client.
//
// The groups are built to mirror what the shipped servers (cmd/mcp-http,
// cmd/mcp-stdio) actually expose: writes enabled, deletes DISABLED
// (allowDelete=false). Delete tools exist in the codebase but no shipped server
// turns them on, so documenting them as available would be misleading — see the
// header note.
func products() []product {
	return []product{
		{"Projects", twprojects.DefaultToolsetGroup(false, false, nil)},
		{"Desk", twdesk.DefaultToolsetGroup(false, nil)},
		{"Spaces", twspaces.DefaultToolsetGroup(false, false, nil)},
		{"Chat", twchat.DefaultToolsetGroup(false, nil)},
	}
}

// verbColumn maps a leading action verb to its matrix column. Verbs not listed
// here are treated as "other actions" and listed separately. "delete" is
// deliberately absent: shipped servers never expose delete tools (see
// products()), and any that do appear should surface visibly under "Other
// actions" rather than in a column, never silently dropped.
var verbColumn = map[string]string{
	"create": "Create",
	"get":    "Get",
	"list":   "List",
	"update": "Update",
}

// matrixColumns is the fixed left-to-right order of matrix columns.
var matrixColumns = []string{"Create", "Get", "List", "Update"}

// forceOther holds tool slugs (without the product prefix) whose verb-based
// classification is misleading, so they are listed under "Other actions"
// instead of forming a spurious resource row.
var forceOther = map[string]bool{
	"get_or_create_dm": true,
}

// displayOverrides maps resource/toolset slugs whose default title-casing reads
// poorly to a hand-tuned display name.
var displayOverrides = map[string]string{
	"jobrole":      "Job Role",
	"user_me":      "Current User (me)",
	"current_user": "Current User",
}

// resourceInfo tracks, for a single resource within a toolset, which CRUD
// columns are present.
type resourceInfo struct {
	name    string          // display name, e.g. "Task"
	order   int             // first-seen index for stable ordering
	columns map[string]bool // column name -> present
}

const (
	defaultMarkdownPath = "docs/tool-reference.md"
	defaultHTMLPath     = "docs/index.html"
)

func main() {
	out := flag.String("o", defaultMarkdownPath, "Markdown output file, or - for stdout")
	htmlOut := flag.String("html", defaultHTMLPath, "HTML output file, or - for stdout")
	check := flag.Bool("check", false,
		"verify the committed docs match freshly generated output; exit non-zero if stale")
	flag.Parse()

	markdown := generate()
	page, err := generateHTML()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating HTML: %v\n", err)
		os.Exit(1)
	}

	docs := []struct {
		path     string
		fallback string
		content  string
	}{
		{*out, defaultMarkdownPath, markdown},
		{*htmlOut, defaultHTMLPath, page},
	}

	if *check {
		var stale bool
		for _, doc := range docs {
			path := doc.path
			if path == "-" {
				path = doc.fallback
			}
			committed, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading %s: %v\n", path, err)
				os.Exit(1)
			}
			if !bytes.Equal(committed, []byte(doc.content)) {
				fmt.Fprintf(os.Stderr,
					"%s is stale — run `go run ./cmd/docs-gen` to regenerate it\n", path)
				stale = true
			}
		}
		if stale {
			os.Exit(1)
		}
		return
	}

	for _, doc := range docs {
		if doc.path == "-" {
			fmt.Print(doc.content)
			continue
		}
		if err := os.WriteFile(doc.path, []byte(doc.content), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", doc.path, err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", doc.path)
	}
}

// generate renders the full tool-reference Markdown document from the
// registered toolsets and returns it. It is the single source of truth for the
// document body, shared by main() (which writes it to disk) and the tests
// (which assert the committed file matches this output).
func generate() string {
	var b strings.Builder
	writeHeader(&b)

	for _, p := range products() {
		fmt.Fprintf(&b, "\n## %s\n", p.label)
		for _, method := range sortedMethods(p.group) {
			ts, err := p.group.GetToolset(method)
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: toolset %q not found: %v\n", method, err)
				continue
			}
			writeToolset(&b, ts)
		}
	}

	return b.String()
}

func writeHeader(b *strings.Builder) {
	b.WriteString("# Teamwork MCP — Tool Reference\n\n")
	b.WriteString("Auto-generated from the registered toolsets by `cmd/docs-gen`. ")
	b.WriteString("Do not edit by hand — run `go run ./cmd/docs-gen` to regenerate.\n\n")
	b.WriteString("This reflects the tools a client actually receives from the shipped servers ")
	b.WriteString("(`cmd/mcp-http`, `cmd/mcp-stdio`) with writes enabled. **Delete operations are ")
	b.WriteString("intentionally omitted**: they exist in the codebase but are gated behind an ")
	b.WriteString("`allowDelete` flag that no shipped server enables, so no client can invoke them. ")
	b.WriteString("Running a server with `-read-only` removes the write tools, leaving the Get/List ")
	b.WriteString("operations plus any read-only entries under \"Other actions\" (e.g. `search`, ")
	b.WriteString("`summarize_timelogs`, `users_workload`).\n")
}

// sortedMethods returns a group's toolset methods in alphabetical order for
// deterministic output.
func sortedMethods(g *toolsets.ToolsetGroup) []toolsets.Method {
	methods := make([]toolsets.Method, 0, len(g.Toolsets))
	for m := range g.Toolsets {
		methods = append(methods, m)
	}
	slices.Sort(methods)
	return methods
}

func writeToolset(b *strings.Builder, ts *toolsets.Toolset) {
	fmt.Fprintf(b, "\n### %s — `%s`\n\n", toolsetTitle(ts.Method), ts.Method)
	if ts.Description != "" {
		fmt.Fprintf(b, "%s\n\n", ts.Description)
	}

	resources, others := classifyToolset(ts)

	// Matrix table.
	b.WriteString("| Resource | " + strings.Join(matrixColumns, " | ") + " |\n")
	b.WriteString("|" + strings.Repeat("---|", len(matrixColumns)+1) + "\n")
	for _, ri := range resources {
		cells := make([]string, 0, len(matrixColumns))
		for _, col := range matrixColumns {
			if ri.columns[col] {
				cells = append(cells, "✓")
			} else {
				cells = append(cells, "—")
			}
		}
		fmt.Fprintf(b, "| %s | %s |\n", ri.name, strings.Join(cells, " | "))
	}

	// Other (non-CRUD) actions.
	if len(others) > 0 {
		labels := make([]string, 0, len(others))
		for _, o := range others {
			labels = append(labels, "`"+o+"`")
		}
		fmt.Fprintf(b, "\n**Other actions:** %s\n", strings.Join(labels, ", "))
	}
}

// classifyToolset splits a toolset's tools into the CRUD matrix rows and the
// leftover "other actions" slugs. Both renderers (Markdown and HTML) consume it,
// so a tool cannot land in one document's matrix and the other's other-actions
// list. Resources come back in first-seen order; others are sorted.
func classifyToolset(ts *toolsets.Toolset) ([]*resourceInfo, []string) {
	byKey := map[string]*resourceInfo{}
	var ordered []*resourceInfo
	var others []string

	for _, tw := range ts.GetAvailableTools() {
		slug := stripPrefix(tw.Tool.Name)
		verb, rest, _ := strings.Cut(slug, "_")
		col, isCRUD := verbColumn[verb]
		if !isCRUD || rest == "" || forceOther[slug] {
			others = append(others, slug)
			continue
		}
		key := singular(rest)
		ri, ok := byKey[key]
		if !ok {
			ri = &resourceInfo{name: displayName(key), order: len(ordered), columns: map[string]bool{}}
			byKey[key] = ri
			ordered = append(ordered, ri)
		}
		ri.columns[col] = true
	}

	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].order < ordered[j].order })
	sort.Strings(others)
	return ordered, others
}

// stripPrefix removes the "tw<product>-" namespace prefix from a tool or method
// name, returning the action/toolset slug.
func stripPrefix(name string) string {
	if _, after, ok := strings.Cut(name, "-"); ok {
		return after
	}
	return name
}

func toolsetTitle(m toolsets.Method) string {
	return displayName(stripPrefix(string(m)))
}

// displayName converts a snake_case slug into a Title Case display name.
func displayName(slug string) string {
	if override, ok := displayOverrides[slug]; ok {
		return override
	}
	parts := strings.Split(slug, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// singular converts a (possibly plural) resource slug to a canonical singular
// key so that plural (list_*) and singular (get_*) tools collapse to one row —
// e.g. tasks→task, statuses→status, inboxes→inbox, categories→category. It also
// leaves already-singular forms untouched (status, address) so the two spellings
// map to the same key.
func singular(slug string) string {
	switch {
	case strings.HasSuffix(slug, "ies") && len(slug) > 4:
		return slug[:len(slug)-3] + "y"
	case strings.HasSuffix(slug, "us"), strings.HasSuffix(slug, "ss"):
		return slug // status, campus, address — already singular
	case strings.HasSuffix(slug, "es") && len(slug) > 3:
		stem := slug[:len(slug)-2]
		// -es plural of a sibilant stem (status→statuses, inbox→inboxes,
		// search→searches) drops "es"; otherwise it's a plain -s (type→types).
		switch {
		case strings.HasSuffix(stem, "s"),
			strings.HasSuffix(stem, "x"),
			strings.HasSuffix(stem, "z"),
			strings.HasSuffix(stem, "ch"),
			strings.HasSuffix(stem, "sh"):
			return stem
		default:
			return slug[:len(slug)-1]
		}
	case strings.HasSuffix(slug, "s"):
		return slug[:len(slug)-1]
	default:
		return slug
	}
}
