package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/teamwork/mcp/internal/toolsets"
	"github.com/teamwork/mcp/internal/twprojects"
)

// docPath is the committed reference doc, relative to this package's directory
// (the test working directory).
const docPath = "../../docs/tool-reference.md"

// TestGeneratedDocMatchesCommitted is the GOLDEN guard: the committed
// docs/tool-reference.md must be byte-identical to freshly generated output.
// This catches the common case of a tool being added, removed, or renamed
// without regenerating the doc.
func TestGeneratedDocMatchesCommitted(t *testing.T) {
	committed, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading committed doc %s: %v", filepath.Clean(docPath), err)
	}
	if got := generate(); got != string(committed) {
		t.Errorf("docs/tool-reference.md is stale — run `go run ./cmd/docs-gen` to regenerate it")
	}
}

// placement describes where a single tool lands in the rendered doc: either a
// matrix cell (resource + CRUD column) or an "Other actions" entry.
type placement struct {
	product string
	method  toolsets.Method
	// exactly one of {resource,column} or other is set.
	resource string // canonical singular resource key, matrix placements only
	column   string // Create/Get/List/Update, matrix placements only
	other    string // slug, "Other actions" placements only
}

// classify mirrors writeToolset's logic to determine the placement of a single
// tool slug. It must stay in lockstep with writeToolset; the GOLDEN test guards
// against divergence because both consume the same helper functions and data.
func classify(product string, method toolsets.Method, toolName string) placement {
	slug := stripPrefix(toolName)
	verb, rest, _ := strings.Cut(slug, "_")
	col, isCRUD := verbColumn[verb]
	if !isCRUD || rest == "" || forceOther[slug] {
		return placement{product: product, method: method, other: slug}
	}
	return placement{product: product, method: method, resource: singular(rest), column: col}
}

func (p placement) isOther() bool { return p.other != "" }

// TestBijection is the BIJECTION guard: every tool exposed by every doc-gen
// group must land in exactly one place in the doc (one matrix cell tick or one
// "Other actions" entry), and every rendered tick/entry must trace back to a
// real tool. It fails if a tool is silently dropped (two tools colliding into
// the same matrix cell) or duplicated.
func TestBijection(t *testing.T) {
	generated := generate()

	// Count what the renderer actually emitted, independently of classify.
	renderedTicks := strings.Count(generated, "✓")

	var matrixTools, otherTools int
	// cellOwner detects collisions: two distinct tools mapping to the same
	// matrix cell would render as a single tick, silently dropping one tool.
	cellOwner := map[placement]string{}
	otherOwner := map[placement]string{}

	for _, p := range products() {
		for method, ts := range p.group.Toolsets {
			for _, tw := range ts.GetAvailableTools() {
				name := tw.Tool.Name
				pl := classify(p.label, method, name)
				if pl.isOther() {
					if prev, ok := otherOwner[pl]; ok {
						t.Errorf("%s/%s: tools %q and %q both map to the same "+
							"Other-actions entry %q", p.label, method, prev, name, pl.other)
					}
					otherOwner[pl] = name
					otherTools++
					continue
				}
				cell := placement{product: pl.product, method: pl.method, resource: pl.resource, column: pl.column}
				if prev, ok := cellOwner[cell]; ok {
					t.Errorf("%s/%s: tools %q and %q both map to matrix cell "+
						"[%s / %s] — one would be silently dropped from the doc",
						p.label, method, prev, name, pl.resource, pl.column)
				}
				cellOwner[cell] = name
				matrixTools++
			}
		}
	}

	// Every distinct matrix cell renders as exactly one tick. With no collisions
	// (asserted above) the number of ticks must equal the number of matrix tools.
	if renderedTicks != len(cellOwner) {
		t.Errorf("rendered ✓ ticks (%d) != distinct matrix cells (%d): a tick "+
			"does not trace back to a tool, or vice versa", renderedTicks, len(cellOwner))
	}
	if matrixTools != len(cellOwner) {
		t.Errorf("matrix tools (%d) != distinct matrix cells (%d): a tool was "+
			"silently dropped from the CRUD matrix", matrixTools, len(cellOwner))
	}

	// Every "Other actions" tool must appear verbatim in the rendered doc.
	for pl, name := range otherOwner {
		if !strings.Contains(generated, "`"+pl.other+"`") {
			t.Errorf("%s/%s: Other-actions tool %q (slug %q) is not present in the "+
				"rendered doc", pl.product, pl.method, name, pl.other)
		}
	}
	if otherTools != len(otherOwner) {
		t.Errorf("other tools (%d) != distinct Other-actions entries (%d)", otherTools, len(otherOwner))
	}
}

// TestAnnotationCrossCheck is the ANNOTATION CROSS-CHECK guard: a tool placed in
// a Get/List column must be annotated read-only, and a tool placed in a
// Create/Update column must not be. This catches a mis-verb-named tool landing
// in the wrong column (e.g. a write tool named "get_*"). "Other actions" tools
// may be either read-only or write, so they are not asserted.
func TestAnnotationCrossCheck(t *testing.T) {
	for _, p := range products() {
		for method, ts := range p.group.Toolsets {
			for _, tw := range ts.GetAvailableTools() {
				name := tw.Tool.Name
				pl := classify(p.label, method, name)
				if pl.isOther() {
					continue
				}
				if tw.Tool.Annotations == nil {
					t.Errorf("%s/%s: tool %q has no annotations; cannot verify "+
						"read-only hint for column %q", p.label, method, name, pl.column)
					continue
				}
				readOnly := tw.Tool.Annotations.ReadOnlyHint
				switch pl.column {
				case "Get", "List":
					if !readOnly {
						t.Errorf("%s/%s: tool %q is in the %q column but ReadOnlyHint=false; "+
							"a write tool is mis-named with a read verb", p.label, method, name, pl.column)
					}
				case "Create", "Update":
					if readOnly {
						t.Errorf("%s/%s: tool %q is in the %q column but ReadOnlyHint=true; "+
							"a read tool is mis-named with a write verb", p.label, method, name, pl.column)
					}
				}
			}
		}
	}
}

// TestNoDeleteToolsExposed is the NO-DELETE guard: the doc-gen groups are built
// with allowDelete=false (mirroring the shipped servers), so no exposed tool
// name may contain "delete". If this fires, a shipped-config assumption changed
// and the doc's "deletes are not exposed" note has gone stale.
func TestNoDeleteToolsExposed(t *testing.T) {
	for _, p := range products() {
		for method, ts := range p.group.Toolsets {
			for _, tw := range ts.GetAvailableTools() {
				if strings.Contains(tw.Tool.Name, "delete") {
					t.Errorf("%s/%s: delete tool %q is exposed by a doc-gen group built "+
						"with allowDelete=false — the doc's 'deletes are not exposed' note is now false",
						p.label, method, tw.Tool.Name)
				}
			}
		}
	}
}

// TestDeleteToolsExistButGated proves the counterpart to TestNoDeleteToolsExposed:
// delete tools DO exist in the codebase and appear once allowDelete=true, so the
// NO-DELETE guard is meaningful (it verifies gating, not mere absence).
//
// The doc-gen groups intentionally pass allowDelete=false to mirror the shipped
// servers. If you change that assumption, also update the header note in
// writeHeader and the corresponding calls in cmd/mcp-http/main.go and
// cmd/mcp-stdio/main.go, which are the actual sources of the shipped default.
func TestDeleteToolsExistButGated(t *testing.T) {
	group := twprojects.DefaultToolsetGroup(false, true, nil)
	var found int
	for _, ts := range group.Toolsets {
		for _, tw := range ts.GetAvailableTools() {
			if strings.Contains(tw.Tool.Name, "delete") {
				found++
			}
		}
	}
	if found == 0 {
		t.Fatal("expected delete tools to appear with allowDelete=true; either they " +
			"were removed or the gating changed — the NO-DELETE guard would be vacuous")
	}
}
